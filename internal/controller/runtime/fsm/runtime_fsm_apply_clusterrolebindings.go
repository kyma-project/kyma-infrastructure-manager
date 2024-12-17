package fsm

import (
	"context"
	"fmt"
	"slices"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
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
	shootAdminClient, err := GetShootClient(ctx, m.Client, s.instance)
	if err != nil {
		updateCRBApplyFailed(&s.instance)
		return updateStatusAndStopWithError(err)
	}
	// list existing cluster role bindings
	var crbList rbacv1.ClusterRoleBindingList
	if err := shootAdminClient.List(ctx, &crbList); err != nil {
		updateCRBApplyFailed(&s.instance)
		m.log.Info("Cannot list Cluster Role Bindings on shoot, scheduling for retry")
		return requeue()
	}

	removed := getRemoved(crbList.Items, s.instance.Spec.Security.Administrators)
	missing := getMissing(crbList.Items, s.instance.Spec.Security.Administrators)

	for _, fn := range []func() error{
		newDelCRBs(ctx, shootAdminClient, removed),
		newAddCRBs(ctx, shootAdminClient, missing),
	} {
		if err := fn(); err != nil {
			updateCRBApplyFailed(&s.instance)
			m.log.Info("Cannot setup Cluster Role Bindings on shoot, scheduling for retry")
			return requeue()
		}
		logDeletedClusterRoleBindings(removed, m, s)
	}

	s.instance.UpdateStateReady(
		imv1.ConditionTypeRuntimeConfigured,
		imv1.ConditionReasonAdministratorsConfigured,
		"Cluster admin configuration complete",
	)

	return updateStatusAndStop()
}

func logDeletedClusterRoleBindings(removed []rbacv1.ClusterRoleBinding, m *fsm, s *systemState) {
	if len(removed) > 0 {
		var crbsNames []string
		for _, binding := range removed {
			crbsNames = append(crbsNames, binding.Name)
		}
		m.log.Info("Following CRBs were deleted", "deletedCRBs", crbsNames)
	}
}

//nolint:gochecknoglobals
var GetShootClient = func(ctx context.Context, cnt client.Client, runtime imv1.Runtime) (client.Client, error) {
	runtimeID := runtime.Labels[imv1.LabelKymaRuntimeID]

	secret, err := getKubeconfigSecret(ctx, cnt, runtimeID, runtime.Namespace)
	if err != nil {
		return nil, err
	}

	restConfig, err := clientcmd.RESTConfigFromKubeConfig(secret.Data[kubeconfigSecretKey])
	if err != nil {
		return nil, err
	}

	shootClientWithAdmin, err := client.New(restConfig, client.Options{})
	if err != nil {
		return nil, err
	}

	return shootClientWithAdmin, nil
}

func getKubeconfigSecret(ctx context.Context, cnt client.Client, runtimeID, namespace string) (corev1.Secret, error) {
	secretName := fmt.Sprintf("kubeconfig-%s", runtimeID)

	var kubeconfigSecret corev1.Secret
	secretKey := types.NamespacedName{Name: secretName, Namespace: namespace}

	err := cnt.Get(ctx, secretKey, &kubeconfigSecret)

	if err != nil {
		return corev1.Secret{}, err
	}

	if kubeconfigSecret.Data == nil {
		return corev1.Secret{}, fmt.Errorf("kubeconfig secret `%s` does not contain kubeconfig data", kubeconfigSecret.Name)
	}
	return kubeconfigSecret, nil
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
var newDelCRBs = func(ctx context.Context, shootClient client.Client, crbs []rbacv1.ClusterRoleBinding) func() error {
	return func() error {
		for _, crb := range crbs {
			if err := shootClient.Delete(ctx, &crb); err != nil {
				return err
			}
		}

		return nil
	}
}

//nolint:gochecknoglobals
var newAddCRBs = func(ctx context.Context, shootClient client.Client, crbs []rbacv1.ClusterRoleBinding) func() error {
	return func() error {
		for _, crb := range crbs {
			if err := shootClient.Create(ctx, &crb); err != nil {
				return err
			}
		}
		return nil
	}
}

func updateCRBApplyFailed(rt *imv1.Runtime) {
	rt.UpdateStatePending(
		imv1.ConditionTypeRuntimeConfigured,
		imv1.ConditionReasonConfigurationErr,
		string(metav1.ConditionFalse),
		"failed to update kubeconfig admin access",
	)
}
