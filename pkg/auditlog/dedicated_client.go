package auditlog

import (
	"context"
	"fmt"

	auditlogv1 "github.com/kyma-project/infrastructure-manager/pkg/auditlog/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// getOrClaimAuditLogCR attempts to get an already claimed AuditLogCR or claim a new one
func (p *DefaultDataProvider) getOrClaimAuditLogCR(ctx context.Context, runtimeID string) (*auditlogv1.AuditLog, error) {
	// Check if already claimed
	claimed, err := p.findAuditLogCRByRuntimeID(ctx, runtimeID)
	if err != nil {
		return nil, fmt.Errorf("failed to find claimed AuditLogCR: %w", err)
	}
	if claimed != nil {
		p.logger.Info("Found already claimed AuditLogCR", "name", claimed.Name, "runtimeID", runtimeID)
		return claimed, nil
	}

	// Find available AuditLogCR
	available, err := p.findAvailableAuditLogCR(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find available AuditLogCR: %w", err)
	}
	if available == nil {
		return nil, fmt.Errorf("no available AuditLogCR in the pool")
	}

	// Claim it (optimistic concurrency via resourceVersion)
	available.Spec.AssignedToRuntimeID = runtimeID
	if err := p.client.Update(ctx, available); err != nil {
		if apierrors.IsConflict(err) {
			// Someone else claimed it concurrently, that's ok - caller can retry
			return nil, fmt.Errorf("conflict claiming AuditLogCR %s: another runtime claimed it concurrently", available.Name)
		}
		return nil, fmt.Errorf("failed to claim AuditLogCR %s: %w", available.Name, err)
	}

	p.logger.Info("Successfully claimed AuditLogCR", "name", available.Name, "runtimeID", runtimeID)
	return available, nil
}

// findAuditLogCRByRuntimeID finds an AuditLogCR that is assigned to the given runtime ID
func (p *DefaultDataProvider) findAuditLogCRByRuntimeID(ctx context.Context, runtimeID string) (*auditlogv1.AuditLog, error) {
	var auditLogList auditlogv1.AuditLogList
	err := p.client.List(ctx, &auditLogList, client.MatchingFields{
		"spec.assignedToRuntimeID": runtimeID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list AuditLog CRs: %w", err)
	}

	if len(auditLogList.Items) == 0 {
		return nil, nil
	}

	if len(auditLogList.Items) > 1 {
		p.logger.Info("Warning: multiple AuditLog CRs found for runtime, using first one",
			"runtimeID", runtimeID, "count", len(auditLogList.Items))
	}

	return &auditLogList.Items[0], nil
}

// findAvailableAuditLogCR finds an available AuditLogCR from the pool
// An AuditLogCR is available if it's in SiemApproved state and not assigned to any runtime
func (p *DefaultDataProvider) findAvailableAuditLogCR(ctx context.Context) (*auditlogv1.AuditLog, error) {
	var auditLogList auditlogv1.AuditLogList
	err := p.client.List(ctx, &auditLogList)
	if err != nil {
		return nil, fmt.Errorf("failed to list AuditLog CRs: %w", err)
	}

	// Find first available CR in SiemApproved state without assigned runtime
	for i := range auditLogList.Items {
		auditLog := &auditLogList.Items[i]
		if auditLog.Status.State == auditlogv1.StateSiemApproved &&
			auditLog.Spec.AssignedToRuntimeID == "" {
			return auditLog, nil
		}
	}

	return nil, nil
}

// releaseAuditLogCR releases an AuditLogCR by marking it as orphaned
func (p *DefaultDataProvider) releaseAuditLogCR(ctx context.Context, auditLog *auditlogv1.AuditLog) error {
	if auditLog.Status.State != auditlogv1.StateAssigned {
		p.logger.Info("AuditLogCR is not in Assigned state, skipping release",
			"name", auditLog.Name, "state", auditLog.Status.State)
		return nil
	}

	// Mark as orphaned - KALM will handle the retention period and cleanup
	auditLog.Spec.Orphaned = true
	if err := p.client.Update(ctx, auditLog); err != nil {
		return fmt.Errorf("failed to mark AuditLogCR %s as orphaned: %w", auditLog.Name, err)
	}

	p.logger.Info("Successfully marked AuditLogCR as orphaned",
		"name", auditLog.Name,
		"runtimeID", auditLog.Spec.AssignedToRuntimeID)
	return nil
}
