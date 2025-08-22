package testing

import (
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func PendingStatusShootPatched() imv1.RuntimeStatus {
	var result imv1.RuntimeStatus
	result.State = imv1.RuntimeStatePending
	result.ProvisioningCompleted = false

	condition := metav1.Condition{
		Type:    string(imv1.ConditionTypeRuntimeProvisioned),
		Status:  metav1.ConditionStatus("Unknown"),
		Reason:  string(imv1.ConditionReasonProcessing),
		Message: "Shoot is pending for update after patch",
	}
	meta.SetStatusCondition(&result.Conditions, condition)
	return result
}

func PendingStatusShootNoChanged() imv1.RuntimeStatus {
	var result imv1.RuntimeStatus
	result.State = imv1.RuntimeStatePending
	result.ProvisioningCompleted = false

	condition := metav1.Condition{
		Type:    string(imv1.ConditionTypeRuntimeProvisioned),
		Status:  metav1.ConditionStatus("True"),
		Reason:  string(imv1.ConditionReasonProcessing),
		Message: "Shoot patched without changes",
	}
	meta.SetStatusCondition(&result.Conditions, condition)
	return result
}

func PendingStatusAfterConflictErr() imv1.RuntimeStatus {
	var result imv1.RuntimeStatus
	result.State = imv1.RuntimeStatePending
	result.ProvisioningCompleted = false

	condition := metav1.Condition{
		Type:    string(imv1.ConditionTypeRuntimeProvisioned),
		Status:  metav1.ConditionStatus("Unknown"),
		Reason:  string(imv1.ConditionReasonProcessing),
		Message: "Shoot is pending for update after conflict error",
	}
	meta.SetStatusCondition(&result.Conditions, condition)
	return result
}

func PendingStatusAfterForbiddenErr() imv1.RuntimeStatus {
	var result imv1.RuntimeStatus
	result.State = imv1.RuntimeStatePending
	result.ProvisioningCompleted = false

	condition := metav1.Condition{
		Type:    string(imv1.ConditionTypeRuntimeProvisioned),
		Status:  metav1.ConditionStatus("Unknown"),
		Reason:  string(imv1.ConditionReasonProcessing),
		Message: "Shoot is pending for update after forbidden error",
	}
	meta.SetStatusCondition(&result.Conditions, condition)
	return result
}

func FailedStatusPatchErr() imv1.RuntimeStatus {
	var result imv1.RuntimeStatus
	result.State = imv1.RuntimeStateFailed
	result.ProvisioningCompleted = false

	condition := metav1.Condition{
		Type:    string(imv1.ConditionTypeRuntimeProvisioned),
		Status:  metav1.ConditionStatus("False"),
		Reason:  string(imv1.ConditionReasonProcessingErr),
		Message: "Gardener API shoot patch error: test unauthorized",
	}
	meta.SetStatusCondition(&result.Conditions, condition)
	return result
}

func FailedStatusUpdateError() imv1.RuntimeStatus {
	var result imv1.RuntimeStatus
	result.State = imv1.RuntimeStateFailed
	result.ProvisioningCompleted = false

	condition := metav1.Condition{
		Type:    string(imv1.ConditionTypeRuntimeProvisioned),
		Status:  metav1.ConditionStatus("False"),
		Reason:  string(imv1.ConditionReasonProcessingErr),
		Message: "Gardener API shoot update error: test unauthorized",
	}
	meta.SetStatusCondition(&result.Conditions, condition)
	return result
}

func FailedStatusAuditLogError() imv1.RuntimeStatus {
	var result imv1.RuntimeStatus
	result.State = imv1.RuntimeStateFailed
	result.ProvisioningCompleted = false

	condition := metav1.Condition{
		Type:    string(imv1.ConditionTypeRuntimeProvisioned),
		Status:  metav1.ConditionStatus("False"),
		Reason:  string(imv1.ConditionReasonAuditLogError),
		Message: "Failed to configure audit logs",
	}
	meta.SetStatusCondition(&result.Conditions, condition)
	return result
}

func FailedStatusRegistryCache() imv1.RuntimeStatus {
	var result imv1.RuntimeStatus
	result.State = imv1.RuntimeStateFailed
	result.ProvisioningCompleted = false

	condition := metav1.Condition{
		Type:    string(imv1.ConditionTypeRuntimeProvisioned),
		Status:  metav1.ConditionStatus("False"),
		Reason:  string(imv1.ConditionReasonRegistryCacheError),
		Message: "Failed to configure registry cache",
	}
	meta.SetStatusCondition(&result.Conditions, condition)
	return result
}
