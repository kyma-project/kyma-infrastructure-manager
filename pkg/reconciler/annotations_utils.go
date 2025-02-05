package reconciler

const (
	ForceReconcileAnnotation   = "operator.kyma-project.io/force-patch-reconciliation"
	SuspendReconcileAnnotation = "operator.kyma-project.io/suspend-patch-reconciliation"
)

func ShouldSuspendReconciliation(annotations map[string]string) bool {
	suspendValue, found := annotations[SuspendReconcileAnnotation]
	if found && suspendValue == "true" {
		return true
	}
	return false
}

func ShouldForceReconciliation(annotations map[string]string) bool {
	forceReconciliation, found := annotations[ForceReconcileAnnotation]
	if found && forceReconciliation == "true" {
		return true
	}
	return false
}
