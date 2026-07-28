package fsm

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode"

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

const maxAdministratorLength = 253

// blockedPrefixes contains prefixes that are not allowed for administrators.
// All Kubernetes built-in identities start with "system:" and granting them
// cluster-admin would be a security risk.
var blockedPrefixes = []string{
	"system:",
}

// validateAdministrator checks if an administrator string is valid.
// It blocks:
// - Empty or whitespace-only strings
// - Strings with leading/trailing whitespace
// - Strings exceeding 253 characters
// - Strings containing control characters
// - Kubernetes built-in identities (anything starting with "system:")
func validateAdministrator(admin string) error {
	if strings.TrimSpace(admin) == "" {
		return fmt.Errorf("cannot be empty or whitespace-only")
	}

	if admin != strings.TrimSpace(admin) {
		return fmt.Errorf("cannot have leading or trailing whitespace")
	}

	if len([]rune(admin)) > maxAdministratorLength {
		return fmt.Errorf("exceeds maximum length of %d characters", maxAdministratorLength)
	}

	for _, r := range admin {
		if unicode.IsControl(r) {
			return fmt.Errorf("contains invalid control character")
		}
	}

	adminLower := strings.ToLower(admin)
	for _, prefix := range blockedPrefixes {
		if strings.HasPrefix(adminLower, prefix) {
			return fmt.Errorf("cannot start with '%s' (Kubernetes built-in identity)", prefix)
		}
	}

	return nil
}

// validateAdministrators validates a list of administrator strings.
func validateAdministrators(admins []string) error {
	for i, admin := range admins {
		if err := validateAdministrator(admin); err != nil {
			return fmt.Errorf("administrator[%d] %q: %w", i, admin, err)
		}
	}
	return nil
}

func sFnApplyClusterRoleBindings(ctx context.Context, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	runtimeClient, err := m.RuntimeClientGetter.Get(ctx, s.instance)
	if err != nil {
		s.instance.UpdateStatePending(
			imv1.ConditionTypeRuntimeConfigured,
			imv1.ConditionReasonConfigurationErr,
			metav1.ConditionFalse,
			"failed to get runtime client",
		)
		m.log.Error(err, "Failed to get runtime client")

		return updateStatusAndRequeueAfter(m.ControlPlaneRequeueDuration)
	}
	// list existing cluster role bindings
	var crbList rbacv1.ClusterRoleBindingList
	if err := runtimeClient.List(ctx, &crbList); err != nil {
		updateCRBApplyPending(&s.instance)
		m.log.Info("Cannot list Cluster Role Bindings on shoot, scheduling for retry")
		return updateStatusAndRequeueAfter(m.ControlPlaneRequeueDuration)
	}

	// Validate administrators before processing
	if err := validateAdministrators(s.instance.Spec.Security.Administrators); err != nil {
		s.instance.UpdateStateFailed(
			imv1.ConditionTypeRuntimeConfigured,
			imv1.ConditionReasonConfigurationErr,
			fmt.Sprintf("invalid administrator: %s", err.Error()),
		)
		m.log.Error(err, "Invalid administrator configuration")
		return updateStatusAndStopWithError(err)
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

	m.log.Info("Finished configuring shoot")

	// Only proceed to audit log migration if both flags are enabled
	if m.DedicatedAuditLoggingEnabled && s.instance.IsDedicatedAuditLogEnabled() {

		s.instance.UpdateStatePending(
			imv1.ConditionTypeRuntimeConfigured,
			imv1.ConditionReasonAdministratorsConfigured,
			metav1.ConditionTrue,
			"Cluster admin configuration completed",
		)

		m.log.Info("Proceeding to dedicated audit log infrastructure configuration")
		return switchState(sFnMigrateToDedicatedAuditLog)
	}

	s.instance.UpdateStateReady(
		imv1.ConditionTypeRuntimeConfigured,
		imv1.ConditionReasonAdministratorsConfigured,
		"Cluster admin configuration completed",
	)

	// Complete provisioning without migration
	if !s.instance.IsProvisioningCompletedStatusSet() {
		s.instance.UpdateStateProvisioningCompleted()
	}

	m.log.Info("Finished configuring shoot without audit log migration")
	return updateStatusAndStop()
}

func logDeletedClusterRoleBindings(removed []rbacv1.ClusterRoleBinding, m *fsm, _ *systemState) {
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

		if crb.RoleRef.Kind != "ClusterRole" || crb.RoleRef.Name != "cluster-admin" {
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
