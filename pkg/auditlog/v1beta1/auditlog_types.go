/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// State defines the observed state of AuditLog
// +kubebuilder:validation:Enum=Pending;RegistrationReady;SiemApproved;Assigned;Orphaned;PendingDelete;UnrecoverableError
type State string

// AuditLogConfig contains configuration for audit log service
type AuditLogConfig struct {
	// ServiceURL is the Audit Log Service URL for Gardener shoot configuration. Once set it cannot be changed
	// +optional
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="ServiceURL is immutable"
	// +kubebuilder:validation:MaxLength=128
	ServiceURL string `json:"serviceURL,omitempty"`
	// GardenerSecretName is the name of Gardener secret with write credentials. Once set it cannot be changed
	// +optional
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="GardenerSecretName is immutable"
	// +kubebuilder:validation:MaxLength=128
	GardenerSecretName string `json:"gardenerSecretName,omitempty"`
	// ReadCredsSecretName is the name of the secret containing OAuth read credentials
	// This secret will be copied to SKR to allow users to access their audit logs. Once set it cannot be changed
	// +optional
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="ReadCredsSecretName is immutable"
	// +kubebuilder:validation:MaxLength=128
	ReadCredsSecretName string `json:"readCredsSecretName,omitempty"`
}

// AuditLogSpec defines the desired state of AuditLog
type AuditLogSpec struct {
	// PlatformRegion is the BTP region where the audit log subaccount is provisioned. Cannot be empty and cannot be changed.
	// The region must be configured in the controller's region-config.json file.
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="PlatformRegion is immutable"
	// +kubebuilder:validation:MaxLength=64
	PlatformRegion string `json:"platformRegion"`
	// Regions contains the hyperscaler regions (e.g. ap-northeast-1, westeurope, us-central1) that are
	// geographically close to the platform region and will be served by this audit log subaccount.
	// At least one region must be specified. Cannot be empty and cannot be changed.
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Regions is immutable"
	Regions []string `json:"regions"`
	// SubaccountID is the BTP subaccount ID (provisioned by controller). Can be initially empty, but once set it cannot be changed or deleted
	// +optional
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="SubaccountID is immutable once set"
	// +kubebuilder:validation:MaxLength=128
	SubaccountID string `json:"subaccountID,omitempty"`
	// AssignedToRuntimeID contains ID of the Kyma Runtime that owns this Auditlog resource. Can be initially empty, but once set it cannot be changed or deleted
	// +optional
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="AssignedToRuntimeID is immutable once set"
	// +kubebuilder:validation:MaxLength=128
	AssignedToRuntimeID string `json:"assignedToRuntimeID,omitempty"`
	// Orphaned indicates that the Kyma Runtime associated with this AuditLog has been deleted.
	// When set to true by KIM, KALM will transition the AuditLog to Orphaned state.
	// Once set to true, this field cannot be changed back to false.
	// Can only be set to true when the AuditLog is in Assigned state (enforced by controller).
	// +optional
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:XValidation:rule="!oldSelf || self",message="Orphaned cannot be unset once true"
	Orphaned bool `json:"orphaned,omitempty"`
	// Config contains configuration for audit log service
	// +optional
	Config AuditLogConfig `json:"config,omitempty"`
	// RetentionDays is the retention period in days (default: 90)
	// +kubebuilder:default=90
	// +optional
	RetentionDays int `json:"retentionDays,omitempty"`
}

// Valid AuditLog States.
const (
	// StatePending signifies AuditLog is pending provisioning of BTP resources.
	StatePending State = "Pending"

	// StateRegistrationReady signifies BTP resources are fully provisioned and ready for SIEM registration.
	StateRegistrationReady State = "RegistrationReady"

	// StateSiemApproved signifies the subaccount has been approved by SIEM team and is available in the pool.
	StateSiemApproved State = "SiemApproved"

	// StateAssigned signifies AuditLog is assigned to and in use by a Kyma Runtime.
	StateAssigned State = "Assigned"

	// StatePendingDelete signifies AuditLog and all its resources is being deleted.
	StatePendingDelete State = "PendingDelete"

	// StateOrphaned signifies the runtime has been deleted but audit logs are retained for the retention period.
	StateOrphaned State = "Orphaned"

	// StateUnrecoverableError signifies the critical failure of logging stack that cannot be revovered.
	StateUnrecoverableError State = "UnrecoverableError"
)

// Condition types for detailed status tracking
const (
	// ConditionTypeSubaccountReady indicates if BTP subaccount exists and is accessible
	ConditionTypeSubaccountReady = "SubaccountReady"

	// ConditionTypeSubaccountEntitlementReady indicates if BTP subaccount is configured with audit log service entitlement
	ConditionTypeSubaccountEntitlementReady = "SubaccountEntitlementReady"

	// ConditionTypeServiceManagerReady indicates if Service Manager binding for subaccount exists and is accessible
	ConditionTypeServiceManagerReady = "ServiceManagerReady"

	// ConditionTypeAuditlogMgmtInstanceReady indicates if auditlog-management service instance is provisioned
	ConditionTypeAuditlogMgmtInstanceReady = "AuditlogMgmtInstanceReady"

	// ConditionTypeAuditlogInstanceReady indicates if auditlog service instance is provisioned
	ConditionTypeAuditlogInstanceReady = "AuditlogInstanceReady"

	// ConditionTypeBindingsReady indicates if service binding exists and credentials are retrievable
	ConditionTypeBindingsReady = "BindingsReady"

	// ConditionTypeCredentialsStored indicates if credentials are stored in Gardener and AuditLog CR
	ConditionTypeCredentialsStored = "CredentialsStored"

	// ConditionTypeBTPResourcesProvisioned indicates all BTP resources are fully provisioned
	ConditionTypeBTPResourcesProvisioned = "ConditionTypeBTPResourcesProvisioned"

	// ConditionTypeCredentialsDeleted indicates if credentials have been deleted from Gardener and AuditLog CR
	ConditionTypeCredentialsDeleted = "CredentialsDeleted"

	// ConditionTypeBindingsDeleted indicates if service binding are deleted
	ConditionTypeBindingsDeleted = "BindingsDeleted"

	// ConditionTypeAuditlogInstanceDeleted indicates if auditlog service instance is deleted
	ConditionTypeAuditlogInstanceDeleted = "AuditlogInstanceDeleted"

	// ConditionTypeAuditlogMgmtInstanceDeleted indicates if auditlog-management service instance is deleted
	ConditionTypeAuditlogMgmtInstanceDeleted = "AuditlogMgmtInstanceDeleted"

	// ConditionTypeServiceManagerBindingDeleted indicates if Service Manager binding is deleted
	ConditionTypeServiceManagerBindingDeleted = "ServiceManagerBindingDeleted"

	// ConditionTypeSubaccountDeleted indicates if BTP subaccount is deleted
	ConditionTypeSubaccountDeleted = "SubaccountDeleted"
)

// Condition reasons
const (
	ConditionReasonCreated       = "Created"
	ConditionReasonProvisioning  = "Provisioning"
	ConditionReasonReady         = "Ready"
	ConditionReasonFailed        = "Failed"
	ConditionReasonCISAPIError   = "CISAPIError"
	ConditionReasonQuotaExceeded = "QuotaExceeded"
	ConditionReasonDeleting      = "Deleting"
	ConditionReasonDeleted       = "Deleted"
)

// Annotations used by external systems to signal state changes
const (
	// AnnotationSkipRetentionCheck allows operators to bypass retention period validation.
	// Value must be "true" to skip; any other value or missing = enforce retention.
	// Use case: Emergency cleanup, testing environments, manual intervention.
	// Warning: Skipping retention may result in audit log data loss.
	// See: docs/contributor/architecture/adr/004-skip-retention-check-annotation.md
	AnnotationSkipRetentionCheck = "auditlogmanager.kyma-project.io/skip-retention-check"
)

// AuditLogStatus defines the observed state of AuditLog.
type AuditLogStatus struct {
	// conditions represent the current state of the AuditLog resource.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// state represents the current state of the AuditLog resource.
	// +kubebuilder:validation:Enum=Pending;RegistrationReady;SiemApproved;Assigned;Orphaned;PendingDelete;UnrecoverableError
	// +kubebuilder:default=Pending
	State State `json:"state"`

	// createdAt is the timestamp when the AuditLog was created
	// +optional
	CreatedAt *metav1.Time `json:"createdAt,omitempty"`

	// assignedAt is the timestamp when the AuditLog was assigned to a runtime
	// +optional
	AssignedAt *metav1.Time `json:"assignedAt,omitempty"`

	// orphanedAt is the timestamp when the AuditLog became orphaned
	// +optional
	OrphanedAt *metav1.Time `json:"orphanedAt,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=".status.state",description="State of the AuditLog"
// +kubebuilder:printcolumn:name="PlatformRegion",type=string,JSONPath=".spec.platformRegion",description="BTP Platform Region"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// AuditLog is the Schema for the auditlogs API
type AuditLog struct {
	metav1.TypeMeta `json:",inline"`

	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// +required
	Spec AuditLogSpec `json:"spec"`

	// status defines the observed state of AuditLog
	// +optional
	Status AuditLogStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// AuditLogList contains a list of AuditLog
type AuditLogList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []AuditLog `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AuditLog{}, &AuditLogList{})
}

func (s *AuditLogStatus) WithState(state State) *AuditLogStatus {
	s.State = state
	return s
}

func (s *AuditLogStatus) WithCondition(conditionType string, status metav1.ConditionStatus, reason, message string, objGeneration int64) *AuditLogStatus {
	if s.Conditions == nil {
		s.Conditions = make([]metav1.Condition, 0, 1)
	}

	condition := meta.FindStatusCondition(s.Conditions, conditionType)

	if condition == nil {
		condition = &metav1.Condition{
			Type:    conditionType,
			Reason:  reason,
			Message: message,
		}
	} else {
		condition.Reason = reason
		condition.Message = message
	}

	condition.Status = status
	condition.ObservedGeneration = objGeneration
	meta.SetStatusCondition(&s.Conditions, *condition)
	return s
}
