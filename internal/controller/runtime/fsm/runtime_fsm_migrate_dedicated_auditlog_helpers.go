package fsm

import (
	"context"
	"encoding/json"
	"fmt"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/kyma-project/infrastructure-manager/pkg/auditlog"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/extensions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// patchShootAuditLog patches the shoot with dedicated audit log configuration
func patchShootAuditLog(ctx context.Context, m *fsm, s *systemState, auditLogData auditlog.AuditLogData) error {
	// Create a copy of the shoot to modify
	patchedShoot := s.shoot.DeepCopy()

	// Find and update the audit log extension
	found := false
	for i := range patchedShoot.Spec.Extensions {
		if patchedShoot.Spec.Extensions[i].Type == extensions.AuditlogExtensionType {
			// Update the audit log configuration with dedicated settings
			if err := updateAuditLogExtension(&patchedShoot.Spec.Extensions[i], auditLogData); err != nil {
				return fmt.Errorf("failed to update audit log extension: %w", err)
			}
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("audit log extension not found in shoot spec")
	}

	// Patch the shoot resource
	if err := m.GardenClient.Patch(ctx, patchedShoot, client.MergeFrom(s.shoot)); err != nil {
		return fmt.Errorf("failed to patch shoot: %w", err)
	}

	// Update the systemState with the patched shoot
	s.shoot = patchedShoot

	return nil
}

// updateAuditLogExtension updates the extension's provider config with dedicated audit log settings
func updateAuditLogExtension(ext *gardener.Extension, auditLogData auditlog.AuditLogData) error {
	if ext.ProviderConfig == nil {
		return fmt.Errorf("provider config is nil")
	}

	// Parse existing config
	var config extensions.AuditlogExtensionConfig
	if err := json.Unmarshal(ext.ProviderConfig.Raw, &config); err != nil {
		return fmt.Errorf("failed to unmarshal provider config: %w", err)
	}

	// Update with dedicated audit log settings
	config.TenantID = auditLogData.TenantID
	config.ServiceURL = auditLogData.ServiceURL
	config.SecretReferenceName = auditLogData.SecretName
	config.Type = "standard"

	// Marshal back to JSON
	updatedConfig, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal updated config: %w", err)
	}

	// Update the provider config
	ext.ProviderConfig.Raw = updatedConfig

	return nil
}

// getShootAuditLogConfig extracts the current audit log configuration from the shoot
func getShootAuditLogConfig(shoot *gardener.Shoot) (*auditlog.AuditLogData, error) {
	if shoot == nil {
		return nil, fmt.Errorf("shoot is nil")
	}

	// Find the audit log extension
	for i := range shoot.Spec.Extensions {
		if shoot.Spec.Extensions[i].Type == extensions.AuditlogExtensionType {
			if shoot.Spec.Extensions[i].ProviderConfig == nil {
				return nil, fmt.Errorf("audit log extension has nil provider config")
			}

			// Parse the extension config
			var config extensions.AuditlogExtensionConfig
			if err := json.Unmarshal(shoot.Spec.Extensions[i].ProviderConfig.Raw, &config); err != nil {
				return nil, fmt.Errorf("failed to unmarshal audit log config: %w", err)
			}

			// Return the audit log data
			return &auditlog.AuditLogData{
				TenantID:   config.TenantID,
				ServiceURL: config.ServiceURL,
				SecretName: config.SecretReferenceName,
			}, nil
		}
	}

	return nil, fmt.Errorf("audit log extension not found in shoot spec")
}

// auditLogConfigsEqual compares two audit log configurations
func auditLogConfigsEqual(current *auditlog.AuditLogData, desired auditlog.AuditLogData) bool {
	if current == nil {
		return false
	}

	return current.TenantID == desired.TenantID &&
		current.ServiceURL == desired.ServiceURL &&
		current.SecretName == desired.SecretName
}
