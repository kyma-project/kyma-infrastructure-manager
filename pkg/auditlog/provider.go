package auditlog

import (
	"context"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DataProvider provides audit logging configuration data
// It abstracts the source - either shared configuration from a file
// or dedicated configuration from AuditLog custom resources
type DataProvider interface {
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

	data.IsDedicated = false
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
		TenantID:    auditLogCR.Spec.SubaccountID, // Use subaccount ID as tenant ID
		ServiceURL:  auditLogCR.Spec.Config.ServiceURL,
		SecretName:  auditLogCR.Spec.Config.GardenerSecretName,
		IsDedicated: true,
	}, nil
}
