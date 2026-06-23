package fsm

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/kyma-project/infrastructure-manager/pkg/auditlog"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/extensions"
	v1 "k8s.io/api/autoscaling/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const dedicatedAuditlogSecretReference = "dedicated-auditlog-credentials"

// patchShootAuditLog patches the shoot with dedicated audit log configuration
// This is a two-part operation:
// 1. Updates the AuditlogExtension's secretReferenceName to use "dedicated-auditlog-credentials"
// 2. Adds/updates a NamedResourceReference that maps "dedicated-auditlog-credentials" to the actual Gardener secret
func patchShootAuditLog(ctx context.Context, m *fsm, s *systemState, auditLogData auditlog.AuditLogData) error {
	// Create a copy of the shoot to modify
	patchedShoot := s.shoot.DeepCopy()

	// Part 1: Find and update the audit log extension
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

	// Part 2: Add or update the secret resource reference
	resource := gardener.NamedResourceReference{
		Name: dedicatedAuditlogSecretReference,
		ResourceRef: v1.CrossVersionObjectReference{
			Name:       auditLogData.SecretName,
			Kind:       "Secret",
			APIVersion: "v1",
		},
	}

	// Find if the resource reference already exists
	index := slices.IndexFunc(patchedShoot.Spec.Resources, func(r gardener.NamedResourceReference) bool {
		return r.Name == dedicatedAuditlogSecretReference
	})

	if index == -1 {
		// Add new resource reference
		patchedShoot.Spec.Resources = append(patchedShoot.Spec.Resources, resource)
	} else {
		// Update existing resource reference
		patchedShoot.Spec.Resources[index] = resource
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
	// Use constant resource reference name, not the actual secret name
	config.SecretReferenceName = dedicatedAuditlogSecretReference
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

			// Look up the actual secret name from shoot.Spec.Resources using the SecretReferenceName
			secretName, err := getSecretNameFromResources(shoot, config.SecretReferenceName)
			if err != nil {
				return nil, fmt.Errorf("failed to get secret name from resources: %w", err)
			}

			// Return the audit log data
			return &auditlog.AuditLogData{
				TenantID:   config.TenantID,
				ServiceURL: config.ServiceURL,
				SecretName: secretName,
			}, nil
		}
	}

	return nil, fmt.Errorf("audit log extension not found in shoot spec")
}

// getSecretNameFromResources looks up the actual Gardener secret name from shoot.Spec.Resources
// using the resource reference name (e.g., "dedicated-auditlog-credentials" or "auditlog-credentials")
func getSecretNameFromResources(shoot *gardener.Shoot, resourceReferenceName string) (string, error) {
	for i := range shoot.Spec.Resources {
		if shoot.Spec.Resources[i].Name == resourceReferenceName {
			return shoot.Spec.Resources[i].ResourceRef.Name, nil
		}
	}
	return "", fmt.Errorf("resource reference '%s' not found in shoot.Spec.Resources", resourceReferenceName)
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
