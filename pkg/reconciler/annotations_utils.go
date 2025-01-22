package reconciler

const (
	ForceReconcileAnnotation = "operator.kyma-project.io/force-shoot-reconciliation"
	SuspendReconcileAnnotation = "operator.kyma-project.io/suspend-shoot-reconciliation"
)

func ShouldSuspendReconciliation(annotations map[string]string) bool {
	suspendValue, found := annotations[SuspendReconcileAnnotation]
	if found == true && suspendValue == "true" {
		return true
	}
	return false
}

func ShouldForceReconciliation(annotations map[string]string) bool {
	forceReconciliation, found := annotations[ForceReconcileAnnotation]
	if found == true && forceReconciliation == "true" {
		return true
	}
	return false
}
