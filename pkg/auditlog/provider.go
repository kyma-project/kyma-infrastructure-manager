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
	ReserveAuditLog(ctx context.Context, providerRegion string, runtimeID string) error

	// GetDedicatedAuditLogData returns audit log configuration from AuditLogCR
	// When claim=true, performs Phase 2 of two-phase claim (upgrades reservation to full claim by setting assignedToRuntimeID)
	// When claim=false, only retrieves data from already claimed/reserved resource
	GetDedicatedAuditLogData(ctx context.Context, runtimeID string, claim bool) (AuditLogData, error)

	// GetSharedAuditLogData returns audit log configuration from shared configuration file
	GetSharedAuditLogData(ctx context.Context, providerType, region string) (AuditLogData, error)

	// ReleaseDedicated releases the claimed AuditLogCR for the runtime
	ReleaseDedicated(ctx context.Context, runtimeID string) error
}

// DefaultDataProvider implements DataProvider
type DefaultDataProvider struct {
	client       client.Client
	sharedConfig Configuration
	logger       logr.Logger
}

// NewDataProvider creates a new DataProvider instance
func NewDataProvider(
	client client.Client,
	sharedConfig Configuration,
	logger logr.Logger,
) DataProvider {
	return &DefaultDataProvider{
		client:       client,
		sharedConfig: sharedConfig,
		logger:       logger,
	}
}

// ReserveAuditLog performs Phase 1 of two-phase claim: reserves an AuditLogCR by adding labels
func (p *DefaultDataProvider) ReserveAuditLog(ctx context.Context, providerRegion string, runtimeID string) error {
	return p.reserveAuditLogCR(ctx, runtimeID, providerRegion)
}

// GetDedicatedAuditLogData returns audit log configuration from AuditLogCR
// When claim=true, performs Phase 2 of two-phase claim (upgrades reservation to full claim)
func (p *DefaultDataProvider) GetDedicatedAuditLogData(ctx context.Context, runtimeID string, claim bool) (AuditLogData, error) {
	if claim {
		// Phase 2: Find the reserved AuditLogCR and upgrade to claim
		reserved, err := p.findAuditLogCRByReservation(ctx, runtimeID)
		if err != nil {
			return AuditLogData{}, fmt.Errorf("failed to find reserved AuditLogCR: %w", err)
		}
		if reserved == nil {
			return AuditLogData{}, fmt.Errorf("no reservation found for runtime %s", runtimeID)
		}

		// Upgrade to claim if not already claimed (idempotent)
		if reserved.Spec.AssignedToRuntimeID != runtimeID {
			reserved.Spec.AssignedToRuntimeID = runtimeID
			if err := p.client.Update(ctx, reserved); err != nil {
				return AuditLogData{}, fmt.Errorf("failed to claim AuditLogCR: %w", err)
			}
			p.logger.Info("Successfully claimed AuditLogCR", "runtimeID", runtimeID, "auditLogCR", reserved.Name)
		} else {
			p.logger.Info("AuditLogCR already claimed", "runtimeID", runtimeID)
		}

		// Return the data
		return AuditLogData{
			TenantID:   reserved.Spec.SubaccountID,
			ServiceURL: reserved.Spec.Config.ServiceURL,
			SecretName: reserved.Spec.Config.GardenerSecretName,
		}, nil
	}

	// claim=false: just retrieve data from already claimed/reserved resource
	// First try to find by claim
	auditLogCR, err := p.findAuditLogCRByRuntimeID(ctx, runtimeID)
	if err != nil {
		return AuditLogData{}, fmt.Errorf("failed to find claimed AuditLogCR: %w", err)
	}
	if auditLogCR == nil {
		// Try to find by reservation
		auditLogCR, err = p.findAuditLogCRByReservation(ctx, runtimeID)
		if err != nil {
			return AuditLogData{}, fmt.Errorf("failed to find reserved AuditLogCR: %w", err)
		}
		if auditLogCR == nil {
			return AuditLogData{}, fmt.Errorf("no AuditLogCR found for runtime %s", runtimeID)
		}
	}

	return AuditLogData{
		TenantID:   auditLogCR.Spec.SubaccountID,
		ServiceURL: auditLogCR.Spec.Config.ServiceURL,
		SecretName: auditLogCR.Spec.Config.GardenerSecretName,
	}, nil
}

// GetSharedAuditLogData returns audit log configuration from shared configuration
func (p *DefaultDataProvider) GetSharedAuditLogData(_ context.Context, providerType, region string) (AuditLogData, error) {
	data, err := p.sharedConfig.GetAuditLogData(providerType, region)
	if err != nil {
		return AuditLogData{}, err
	}
	return data, nil
}

// ReleaseDedicated releases the claimed AuditLogCR for the runtime
func (p *DefaultDataProvider) ReleaseDedicated(ctx context.Context, runtimeID string) error {
	auditLogCR, err := p.findAuditLogCRByRuntimeID(ctx, runtimeID)

	if err != nil {
		return fmt.Errorf("failed to find assigned AuditLogCR for runtime %s: %w", runtimeID, err)
	}
	if auditLogCR == nil {
		return nil
	}

	return p.releaseAuditLogCR(ctx, auditLogCR)
}
