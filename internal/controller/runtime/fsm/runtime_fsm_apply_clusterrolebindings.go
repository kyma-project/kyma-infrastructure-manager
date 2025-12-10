package fsm

import (
	"context"
	"slices"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/log_level"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	//nolint:gochecknoglobals
	labelsManagedByKIM = map[string]string{
		"reconciler.kyma-project.io/managed-by": "infrastructure-manager",
	}
)

func sFnApplyClusterRoleBindings(ctx context.Context, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	runtimeClient, err := m.RuntimeClientGetter.Get(ctx, s.instance)
	if err != nil {
		// TODO: This probably should be replaced with requeue logic, as we do in other places
		s.instance.UpdateStateFailed(
			imv1.ConditionTypeRuntimeConfigured,
			imv1.ConditionReasonConfigurationErr,
			"failed to update kubeconfig admin access",
		)

		return updateStatusAndStopWithError(err)
	}
	// list existing cluster role bindings
	var crbList rbacv1.ClusterRoleBindingList
	if err := runtimeClient.List(ctx, &crbList); err != nil {
		updateCRBApplyPending(&s.instance)
		m.log.Info("Cannot list Cluster Role Bindings on shoot, scheduling for retry")
		return updateStatusAndRequeueAfter(m.ControlPlaneRequeueDuration)
	}

	removed := getRemoved(crbList.Items, s.instance.Spec.Security.Administrators)
	missing := getMissing(crbList.Items, s.instance.Spec.Security.Administrators)

	for _, fn := range []func() error{
		newDelCRBs(ctx, runtimeClient, removed),
		newAddCRBs(ctx, runtimeClient, missing),
	} {
		if err := fn(); err != nil {
			updateCRBApplyPending(&s.instance)
			m.log.Info("Cannot setup Cluster Role Bindings on shoot, scheduling for retry")
			return updateStatusAndRequeueAfter(m.ControlPlaneRequeueDuration)
		}
		logDeletedClusterRoleBindings(removed, m, s)
	}

	s.instance.UpdateStateReady(
		imv1.ConditionTypeRuntimeConfigured,
		imv1.ConditionReasonAdministratorsConfigured,
		"Cluster admin configuration complete",
	)

	if !s.instance.IsProvisioningCompletedStatusSet() {
		s.instance.UpdateStateProvisioningCompleted()
	}

	m.log.Info("Finished configuring shoot")

	return updateStatusAndStop()
}

func logDeletedClusterRoleBindings(removed []rbacv1.ClusterRoleBinding, m *fsm, s *systemState) {
	if len(removed) > 0 {
		var crbsNames []string
		for _, binding := range removed {
			crbsNames = append(crbsNames, binding.Name)
		}
		m.log.V(log_level.DEBUG).Info("Following CRBs were deleted", "deletedCRBs", crbsNames)
	}
}

func isRBACUserKind() func(rbacv1.Subject) bool {
	return func(s rbacv1.Subject) bool {
		return s.Kind == rbacv1.UserKind
	}
}

func isRBACUserKindOneOf(names []string) func(rbacv1.Subject) bool {
	return func(s rbacv1.Subject) bool {
		return slices.Contains(names, s.Name)
	}
}

func getRemoved(crbs []rbacv1.ClusterRoleBinding, admins []string) (removed []rbacv1.ClusterRoleBinding) {
	// iterate over cluster role bindings to find out removed administrators
	for _, crb := range crbs {
		if !managedByKIM(crb) {
			// cluster role binding is not controlled by KIM
			continue
		}

		if crb.RoleRef.Kind != "ClusterRole" && crb.RoleRef.Name != "cluster-admin" {
			// cluster role binding is not admin
			continue
		}

		if !slices.ContainsFunc(crb.Subjects, isRBACUserKind()) {
			// cluster role binding is not user kind
			continue
		}

		if slices.ContainsFunc(crb.Subjects, isRBACUserKindOneOf(admins)) {
			// the administrator was not removed
			continue
		}

		// administrator was removed
		removed = append(removed, crb)
	}

	return removed
}

func managedByKIM(crb rbacv1.ClusterRoleBinding) bool {
	selector := labels.Set(labelsManagedByKIM).AsSelector()
	isManagedByKIM := selector.Matches(labels.Set(crb.Labels))
	return isManagedByKIM
}

//nolint:gochecknoglobals
var newContainsAdmin = func(admin string) func(rbacv1.ClusterRoleBinding) bool {
	return func(crb rbacv1.ClusterRoleBinding) bool {
		if !managedByKIM(crb) {
			return false
		}
		isAdmin := isRBACUserKindOneOf([]string{admin})
		return slices.ContainsFunc(crb.Subjects, isAdmin)
	}
}

func getMissing(crbs []rbacv1.ClusterRoleBinding, admins []string) (missing []rbacv1.ClusterRoleBinding) {
	for _, admin := range admins {
		containsAdmin := newContainsAdmin(admin)
		if slices.ContainsFunc(crbs, containsAdmin) {
			continue
		}
		crb := toAdminClusterRoleBinding(admin)
		missing = append(missing, crb)
	}

	return missing
}

func toAdminClusterRoleBindingWithLabel(name string, key, value string) rbacv1.ClusterRoleBinding {
	// initialize labels
	labels := map[string]string{}
	if key != "" {
		labels[key] = value
	}
	// build CRB
	return rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "admin-",
			Labels:       labels,
		},
		Subjects: []rbacv1.Subject{{
			Kind:     rbacv1.UserKind,
			Name:     name,
			APIGroup: rbacv1.GroupName,
		}},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     "cluster-admin",
		},
	}
}

func toAdminClusterRoleBindingNoLabels(name string) rbacv1.ClusterRoleBinding {
	return toAdminClusterRoleBindingWithLabel(name, "", "")
}

func toAdminClusterRoleBinding(name string) rbacv1.ClusterRoleBinding {
	return toAdminClusterRoleBindingWithLabel(name, "reconciler.kyma-project.io/managed-by", "infrastructure-manager")
}

//nolint:gochecknoglobals
var newDelCRBs = func(ctx context.Context, runtimeClient client.Client, crbs []rbacv1.ClusterRoleBinding) func() error {
	return func() error {
		for _, crb := range crbs {
			if err := runtimeClient.Delete(ctx, &crb); err != nil {
				return err
			}
		}

		return nil
	}
}

//nolint:gochecknoglobals
var newAddCRBs = func(ctx context.Context, runtimeClient client.Client, crbs []rbacv1.ClusterRoleBinding) func() error {
	return func() error {
		for _, crb := range crbs {
			if err := runtimeClient.Create(ctx, &crb); err != nil {
				return err
			}
		}
		return nil
	}
}

func updateCRBApplyPending(rt *imv1.Runtime) {
	rt.UpdateStatePending(
		imv1.ConditionTypeRuntimeConfigured,
		imv1.ConditionReasonConfigurationErr,
		metav1.ConditionFalse,
		"failed to update kubeconfig admin access",
	)
}
