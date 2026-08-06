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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const dedicatedAuditlogSecretReference = "dedicated-auditlog-credentials"

// patchShootAuditLog patches the shoot with dedicated audit log configuration.
// It delegates all data mutations to applyDedicatedAuditLogToShoot and then
// persists the result via the Gardener client.
func patchShootAuditLog(ctx context.Context, m *fsm, s *systemState, auditLogData auditlog.AuditLogData) error {
	patchedShoot := s.shoot.DeepCopy()

	if err := applyDedicatedAuditLogToShoot(patchedShoot, auditLogData); err != nil {
		return err
	}

	if err := m.GardenClient.Patch(ctx, patchedShoot, client.MergeFrom(s.shoot), &client.PatchOptions{
		FieldManager: fieldManagerName,
	}); err != nil {
		return fmt.Errorf("failed to patch shoot: %w", err)
	}

	s.shoot = patchedShoot
	return nil
}

// applyDedicatedAuditLogToShoot mutates a shoot with the dedicated audit log configuration.
// It is a pure function (no I/O) that can be unit-tested in isolation.
func applyDedicatedAuditLogToShoot(shoot *gardener.Shoot, auditLogData auditlog.AuditLogData) error {
	if err := updateAuditLogExtensionConfig(shoot, auditLogData); err != nil {
		return err
	}
	upsertAuditLogSecretReference(shoot, auditLogData.SecretName)
	return nil
}

// updateAuditLogExtensionConfig finds the audit log extension in the shoot and updates
// its provider config with the dedicated audit log settings.
// If no audit log extension exists, it creates a new one.
func updateAuditLogExtensionConfig(shoot *gardener.Shoot, auditLogData auditlog.AuditLogData) error {
	for i := range shoot.Spec.Extensions {
		if shoot.Spec.Extensions[i].Type != extensions.AuditlogExtensionType {
			continue
		}
		if err := updateAuditLogExtension(&shoot.Spec.Extensions[i], auditLogData); err != nil {
			return fmt.Errorf("failed to update audit log extension: %w", err)
		}
		return nil
	}

	// Extension not found - create a new one
	newExt, err := createAuditLogExtension(auditLogData)
	if err != nil {
		return fmt.Errorf("failed to create audit log extension: %w", err)
	}
	shoot.Spec.Extensions = append(shoot.Spec.Extensions, *newExt)
	return nil
}

// upsertAuditLogSecretReference adds or updates the NamedResourceReference that maps
// dedicatedAuditlogSecretReference to the actual Gardener secret.
func upsertAuditLogSecretReference(shoot *gardener.Shoot, secretName string) {
	resource := gardener.NamedResourceReference{
		Name: dedicatedAuditlogSecretReference,
		ResourceRef: v1.CrossVersionObjectReference{
			Name:       secretName,
			Kind:       "Secret",
			APIVersion: "v1",
		},
	}
	index := slices.IndexFunc(shoot.Spec.Resources, func(r gardener.NamedResourceReference) bool {
		return r.Name == dedicatedAuditlogSecretReference
	})
	if index == -1 {
		shoot.Spec.Resources = append(shoot.Spec.Resources, resource)
		return
	}
	shoot.Spec.Resources[index] = resource
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

// createAuditLogExtension creates a new audit log extension with dedicated settings.
// Uses the constant dedicatedAuditlogSecretReference for the secret reference name.
func createAuditLogExtension(auditLogData auditlog.AuditLogData) (*gardener.Extension, error) {
	cfg := extensions.AuditlogExtensionConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AuditlogConfig",
			APIVersion: "service.auditlog.extensions.gardener.cloud/v1alpha1",
		},
		Type:                "standard",
		TenantID:            auditLogData.TenantID,
		ServiceURL:          auditLogData.ServiceURL,
		SecretReferenceName: dedicatedAuditlogSecretReference,
	}

	configJSON, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal audit log config: %w", err)
	}

	return &gardener.Extension{
		Type: extensions.AuditlogExtensionType,
		ProviderConfig: &runtime.RawExtension{
			Raw: configJSON,
		},
	}, nil
}

// getShootAuditLogConfig extracts the current audit log configuration from the shoot
func getShootAuditLogConfig(shoot *gardener.Shoot) (*auditlog.AuditLogData, error) {
	if shoot == nil {
		return nil, fmt.Errorf("shoot is nil")
	}

	// Find the audit log extension
	for i := range shoot.Spec.Extensions {
		if shoot.Spec.Extensions[i].Type != extensions.AuditlogExtensionType {
			continue
		}

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

		return &auditlog.AuditLogData{
			TenantID:   config.TenantID,
			ServiceURL: config.ServiceURL,
			SecretName: secretName,
		}, nil
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
