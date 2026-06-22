package auditlog

import (
	"context"
	"fmt"
	"time"

	auditlogv1 "github.com/kyma-project/infrastructure-manager/pkg/auditlog/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// Label keys for reservation (Phase 1 of two-phase claim)
	LabelReservedForRuntimeID = "reserved-for-runtime-id"
	LabelReservedAt           = "reserved-for-runtime-at"
)

// reserveAuditLogCR performs Phase 1 of two-phase claim: adds reservation labels to an AuditLogCR
// This creates a "light lock" that prevents other runtimes from selecting this CR during provisioning
func (p *DefaultDataProvider) reserveAuditLogCR(ctx context.Context, runtimeID string) error {
	// Check if we already have a reservation
	reserved, err := p.findAuditLogCRByReservation(ctx, runtimeID)
	if err != nil {
		return fmt.Errorf("failed to find reserved AuditLogCR: %w", err)
	}
	if reserved != nil {
		p.logger.Info("Found existing reservation for runtime", "name", reserved.Name, "runtimeID", runtimeID)
		return nil // Already reserved for us
	}

	// Find available AuditLogCR (SiemApproved, not assigned, not reserved)
	available, err := p.findAvailableAuditLogCR(ctx)
	if err != nil {
		return fmt.Errorf("failed to find available AuditLogCR: %w", err)
	}
	if available == nil {
		return fmt.Errorf("no available AuditLogCR in the pool")
	}

	// Add reservation labels (light lock)
	if available.Labels == nil {
		available.Labels = make(map[string]string)
	}
	available.Labels[LabelReservedForRuntimeID] = runtimeID
	available.Labels[LabelReservedAt] = time.Now().UTC().Format(time.RFC3339)

	// Update with optimistic concurrency
	if err := p.client.Update(ctx, available); err != nil {
		if apierrors.IsConflict(err) {
			// Someone else reserved it concurrently, caller should retry
			return fmt.Errorf("conflict reserving AuditLogCR %s: another runtime reserved it concurrently", available.Name)
		}
		return fmt.Errorf("failed to reserve AuditLogCR %s: %w", available.Name, err)
	}

	p.logger.Info("Successfully reserved AuditLogCR",
		"name", available.Name,
		"runtimeID", runtimeID,
		"reservedAt", available.Labels[LabelReservedAt])
	return nil
}

// findAuditLogCRByReservation finds an AuditLogCR that is reserved for the given runtime ID
func (p *DefaultDataProvider) findAuditLogCRByReservation(ctx context.Context, runtimeID string) (*auditlogv1.AuditLog, error) {
	var auditLogList auditlogv1.AuditLogList
	err := p.client.List(ctx, &auditLogList, client.MatchingLabels{
		LabelReservedForRuntimeID: runtimeID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list AuditLog CRs by reservation label: %w", err)
	}

	if len(auditLogList.Items) == 0 {
		return nil, nil
	}

	if len(auditLogList.Items) > 1 {
		p.logger.Info("Warning: multiple reserved AuditLog CRs found for runtime, using first one",
			"runtimeID", runtimeID, "count", len(auditLogList.Items))
	}

	// Return a copy to avoid dangling pointer after auditLogList goes out of scope
	result := auditLogList.Items[0]
	return &result, nil
}

// getOrClaimAuditLogCR performs Phase 2 of two-phase claim: upgrades reservation to full claim
func (p *DefaultDataProvider) getOrClaimAuditLogCR(ctx context.Context, runtimeID string) (*auditlogv1.AuditLog, error) {
	// Check if already claimed (heavy lock via assignedToRuntimeID)
	claimed, err := p.findAuditLogCRByRuntimeID(ctx, runtimeID)
	if err != nil {
		return nil, fmt.Errorf("failed to find claimed AuditLogCR: %w", err)
	}
	if claimed != nil {
		p.logger.Info("Found already claimed AuditLogCR", "name", claimed.Name, "runtimeID", runtimeID)
		return claimed, nil
	}

	// Find our reserved CR (light lock via label)
	reserved, err := p.findAuditLogCRByReservation(ctx, runtimeID)
	if err != nil {
		return nil, fmt.Errorf("failed to find reserved AuditLogCR: %w", err)
	}
	if reserved == nil {
		return nil, fmt.Errorf("no reserved AuditLogCR found for runtime %s (reservation might have been cleaned up or never created)", runtimeID)
	}

	// Upgrade reservation to full claim (heavy lock)
	reserved.Spec.AssignedToRuntimeID = runtimeID
	if err := p.client.Update(ctx, reserved); err != nil {
		if apierrors.IsConflict(err) {
			// Unlikely since we have reservation label, but possible - caller can retry
			return nil, fmt.Errorf("conflict claiming AuditLogCR %s: concurrent update detected", reserved.Name)
		}
		return nil, fmt.Errorf("failed to claim reserved AuditLogCR %s: %w", reserved.Name, err)
	}

	p.logger.Info("Successfully upgraded reservation to claim",
		"name", reserved.Name,
		"runtimeID", runtimeID)
	return reserved, nil
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

	// Return a copy to avoid dangling pointer after auditLogList goes out of scope
	result := auditLogList.Items[0]
	return &result, nil
}

// findAvailableAuditLogCR finds an available AuditLogCR from the pool
// An AuditLogCR is available if it's in SiemApproved state, not assigned to any runtime, and not reserved
func (p *DefaultDataProvider) findAvailableAuditLogCR(ctx context.Context) (*auditlogv1.AuditLog, error) {
	var auditLogList auditlogv1.AuditLogList
	err := p.client.List(ctx, &auditLogList)
	if err != nil {
		return nil, fmt.Errorf("failed to list AuditLog CRs: %w", err)
	}

	// Find first available CR: SiemApproved state, no assignment, no reservation
	for i := range auditLogList.Items {
		auditLog := &auditLogList.Items[i]

		// Check if in correct state and not assigned
		if auditLog.Status.State != auditlogv1.StateSiemApproved ||
			auditLog.Spec.AssignedToRuntimeID != "" {
			continue
		}

		// Check if not reserved by checking for reservation label
		if auditLog.Labels != nil {
			if _, hasReservation := auditLog.Labels[LabelReservedForRuntimeID]; hasReservation {
				continue // Skip reserved CRs
			}
		}

		return auditLog, nil
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
