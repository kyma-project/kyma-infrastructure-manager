package auditlog

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DataProvider provides audit logging configuration data
// It abstracts the source - either shared configuration from a file
// or dedicated configuration from AuditLog custom resources
type DataProvider interface {
	// ReserveAuditLog performs Phase 1 of the two-phase claim: reserves an AuditLogCR by adding labels
	// This should be called before shoot creation to ensure a resource is available
	// Returns error if no available AuditLogCR is found
	ReserveAuditLog(ctx context.Context, providerType, region string, runtimeID string) error

	// ConfirmReservation performs Phase 2 of the two-phase claim: upgrades reservation to full claim
	// This upgrades the light lock (label) to heavy lock (assignedToRuntimeID)
	// Returns error if no reservation is found or claim fails
	ConfirmReservation(ctx context.Context, runtimeID string) error

	// GetReservedAuditLogData returns audit log configuration from a reserved AuditLogCR
	// This retrieves data from a CR that has the reservation label but hasn't been fully claimed yet
	// Used in Phase 2 (migration) to get config before claiming
	GetReservedAuditLogData(ctx context.Context, runtimeID string) (AuditLogData, error)

	// GetAuditLogData returns audit log configuration for the given runtime
	// When dedicated=true, attempts to use dedicated AuditLogCR
	// Falls back to shared config if dedicated is unavailable
	GetAuditLogData(ctx context.Context, providerType, region string, runtimeID string, dedicated bool) (AuditLogData, error)

	// IsDedicated checks if the runtime is using dedicated audit logging
	IsDedicated(ctx context.Context, runtimeID string) (bool, error)

	// ReleaseDedicated releases the claimed AuditLogCR for the runtime
	ReleaseDedicated(ctx context.Context, runtimeID string) error
}

// DefaultDataProvider implements DataProvider
type DefaultDataProvider struct {
	client           client.Client
	sharedConfig     Configuration
	dedicatedEnabled bool
	logger           logr.Logger
}

// NewDataProvider creates a new DataProvider instance
func NewDataProvider(
	client client.Client,
	sharedConfig Configuration,
	dedicatedEnabled bool,
	logger logr.Logger,
) DataProvider {
	return &DefaultDataProvider{
		client:           client,
		sharedConfig:     sharedConfig,
		dedicatedEnabled: dedicatedEnabled,
		logger:           logger,
	}
}

// ReserveAuditLog performs Phase 1 of two-phase claim: reserves an AuditLogCR by adding labels
func (p *DefaultDataProvider) ReserveAuditLog(ctx context.Context, providerType, region string, runtimeID string) error {
	if !p.dedicatedEnabled {
		return nil // Feature disabled, nothing to reserve
	}

	return p.reserveAuditLogCR(ctx, runtimeID)
}

// ConfirmReservation performs Phase 2 of two-phase claim: upgrades reservation to full claim
func (p *DefaultDataProvider) ConfirmReservation(ctx context.Context, runtimeID string) error {
	if !p.dedicatedEnabled {
		return fmt.Errorf("dedicated audit logging is not enabled")
	}

	// Find the reserved AuditLogCR
	reserved, err := p.findAuditLogCRByReservation(ctx, runtimeID)
	if err != nil {
		return fmt.Errorf("failed to find reserved AuditLogCR: %w", err)
	}
	if reserved == nil {
		return fmt.Errorf("no reservation found for runtime %s", runtimeID)
	}

	// Check if already claimed (idempotent)
	if reserved.Spec.AssignedToRuntimeID == runtimeID {
		p.logger.Info("Reservation already confirmed (already claimed)", "runtimeID", runtimeID)
		return nil
	}

	// Upgrade reservation to claim by setting assignedToRuntimeID
	reserved.Spec.AssignedToRuntimeID = runtimeID
	if err := p.client.Update(ctx, reserved); err != nil {
		return fmt.Errorf("failed to confirm reservation (set assignedToRuntimeID): %w", err)
	}

	p.logger.Info("Successfully confirmed reservation (upgraded to claim)",
		"runtimeID", runtimeID,
		"auditLogCR", reserved.Name)

	return nil
}

// GetReservedAuditLogData returns audit log configuration from a reserved AuditLogCR
func (p *DefaultDataProvider) GetReservedAuditLogData(ctx context.Context, runtimeID string) (AuditLogData, error) {
	if !p.dedicatedEnabled {
		return AuditLogData{}, fmt.Errorf("dedicated audit logging is not enabled")
	}

	// Find the reserved AuditLogCR by label
	auditLogCR, err := p.findAuditLogCRByReservation(ctx, runtimeID)
	if err != nil {
		return AuditLogData{}, fmt.Errorf("failed to find reserved AuditLogCR: %w", err)
	}
	if auditLogCR == nil {
		return AuditLogData{}, fmt.Errorf("no reserved AuditLogCR found for runtime %s", runtimeID)
	}

	// Map AuditLogCR config to AuditLogData
	return AuditLogData{
		TenantID:   auditLogCR.Spec.SubaccountID,
		ServiceURL: auditLogCR.Spec.Config.ServiceURL,
		SecretName: auditLogCR.Spec.Config.GardenerSecretName,
	}, nil
}

// GetAuditLogData returns audit log configuration
func (p *DefaultDataProvider) GetAuditLogData(ctx context.Context, providerType, region string, runtimeID string, dedicated bool) (AuditLogData, error) {
	// If dedicated requested and enabled, try to get/claim dedicated config
	if dedicated && p.dedicatedEnabled {
		auditLogData, err := p.getDedicatedAuditLogData(ctx, runtimeID)
		if err != nil {
			p.logger.Info("Failed to get dedicated audit log, falling back to shared config",
				"runtimeID", runtimeID, "error", err.Error())
			return p.getSharedAuditLogData(providerType, region)
		}
		return auditLogData, nil
	}

	// Use shared configuration
	return p.getSharedAuditLogData(providerType, region)
}

// IsDedicated checks if the runtime is using dedicated audit logging
func (p *DefaultDataProvider) IsDedicated(ctx context.Context, runtimeID string) (bool, error) {
	if !p.dedicatedEnabled {
		return false, nil
	}

	auditLogCR, err := p.findAuditLogCRByRuntimeID(ctx, runtimeID)
	if err != nil {
		return false, err
	}
	return auditLogCR != nil, nil
}

// ReleaseDedicated releases the claimed AuditLogCR for the runtime
func (p *DefaultDataProvider) ReleaseDedicated(ctx context.Context, runtimeID string) error {
	if !p.dedicatedEnabled {
		return nil
	}

	auditLogCR, err := p.findAuditLogCRByRuntimeID(ctx, runtimeID)
	if err != nil || auditLogCR == nil {
		return nil // Nothing to release
	}

	return p.releaseAuditLogCR(ctx, auditLogCR)
}

// getSharedAuditLogData retrieves audit log data from shared configuration
func (p *DefaultDataProvider) getSharedAuditLogData(providerType, region string) (AuditLogData, error) {
	data, err := p.sharedConfig.GetAuditLogData(providerType, region)
	if err != nil {
		return AuditLogData{}, err
	}

	return data, nil
}

// getDedicatedAuditLogData retrieves or claims dedicated audit log configuration
func (p *DefaultDataProvider) getDedicatedAuditLogData(ctx context.Context, runtimeID string) (AuditLogData, error) {
	auditLogCR, err := p.getOrClaimAuditLogCR(ctx, runtimeID)
	if err != nil {
		return AuditLogData{}, err
	}

	// Map AuditLogCR config to AuditLogData
	// The Config field contains the Gardener shoot configuration
	return AuditLogData{
		TenantID:   auditLogCR.Spec.SubaccountID, // Use subaccount ID as tenant ID
		ServiceURL: auditLogCR.Spec.Config.ServiceURL,
		SecretName: auditLogCR.Spec.Config.GardenerSecretName,
	}, nil
}
