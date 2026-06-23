package fsm

import (
	"context"
	"encoding/json"
	"testing"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/kyma-project/infrastructure-manager/pkg/auditlog"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/extensions"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestPatchShootAuditLog(t *testing.T) {
	t.Run("should update audit log extension and add resource reference", func(t *testing.T) {
		// given
		ctx := context.Background()
		auditLogData := auditlog.AuditLogData{
			TenantID:   "test-tenant-id",
			ServiceURL: "https://test.auditlog.example.com",
			SecretName: "test-gardener-secret",
		}

		// Create initial shoot with audit log extension
		existingConfig := extensions.AuditlogExtensionConfig{
			Type:                "standard",
			TenantID:            "old-tenant-id",
			ServiceURL:          "https://old.example.com",
			SecretReferenceName: "auditlog-credentials",
		}
		configJSON, _ := json.Marshal(existingConfig)

		shoot := &gardener.Shoot{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-shoot",
				Namespace: "garden-test",
			},
			Spec: gardener.ShootSpec{
				Extensions: []gardener.Extension{
					{
						Type: extensions.AuditlogExtensionType,
						ProviderConfig: &runtime.RawExtension{
							Raw: configJSON,
						},
					},
				},
				Resources: []gardener.NamedResourceReference{
					{
						Name: "auditlog-credentials",
						ResourceRef: v1.CrossVersionObjectReference{
							Name:       "old-secret",
							Kind:       "Secret",
							APIVersion: "v1",
						},
					},
				},
			},
		}

		scheme, _ := newCreateTestScheme()
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(shoot).Build()

		testFsm := &fsm{
			K8s: K8s{
				GardenClient: fakeClient,
			},
		}

		systemState := &systemState{
			shoot: shoot,
		}

		// when
		err := patchShootAuditLog(ctx, testFsm, systemState, auditLogData)

		// then
		require.NoError(t, err)

		// Verify the audit log extension was updated
		require.Len(t, systemState.shoot.Spec.Extensions, 1)
		ext := systemState.shoot.Spec.Extensions[0]
		require.Equal(t, extensions.AuditlogExtensionType, ext.Type)

		var updatedConfig extensions.AuditlogExtensionConfig
		err = json.Unmarshal(ext.ProviderConfig.Raw, &updatedConfig)
		require.NoError(t, err)
		require.Equal(t, "test-tenant-id", updatedConfig.TenantID)
		require.Equal(t, "https://test.auditlog.example.com", updatedConfig.ServiceURL)
		require.Equal(t, dedicatedAuditlogSecretReference, updatedConfig.SecretReferenceName)
		require.Equal(t, "standard", updatedConfig.Type)

		// Verify the resource reference was added
		require.Len(t, systemState.shoot.Spec.Resources, 2)

		// Find the dedicated audit log resource reference
		var dedicatedResource *gardener.NamedResourceReference
		for i := range systemState.shoot.Spec.Resources {
			if systemState.shoot.Spec.Resources[i].Name == dedicatedAuditlogSecretReference {
				dedicatedResource = &systemState.shoot.Spec.Resources[i]
				break
			}
		}

		require.NotNil(t, dedicatedResource)
		require.Equal(t, dedicatedAuditlogSecretReference, dedicatedResource.Name)
		require.Equal(t, "test-gardener-secret", dedicatedResource.ResourceRef.Name)
		require.Equal(t, "Secret", dedicatedResource.ResourceRef.Kind)
		require.Equal(t, "v1", dedicatedResource.ResourceRef.APIVersion)
	})

	t.Run("should update existing dedicated resource reference", func(t *testing.T) {
		// given
		ctx := context.Background()
		auditLogData := auditlog.AuditLogData{
			TenantID:   "new-tenant-id",
			ServiceURL: "https://new.auditlog.example.com",
			SecretName: "new-gardener-secret",
		}

		existingConfig := extensions.AuditlogExtensionConfig{
			Type:                "standard",
			TenantID:            "old-tenant-id",
			ServiceURL:          "https://old.example.com",
			SecretReferenceName: dedicatedAuditlogSecretReference,
		}
		configJSON, _ := json.Marshal(existingConfig)

		shoot := &gardener.Shoot{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-shoot",
				Namespace: "garden-test",
			},
			Spec: gardener.ShootSpec{
				Extensions: []gardener.Extension{
					{
						Type: extensions.AuditlogExtensionType,
						ProviderConfig: &runtime.RawExtension{
							Raw: configJSON,
						},
					},
				},
				Resources: []gardener.NamedResourceReference{
					{
						Name: dedicatedAuditlogSecretReference,
						ResourceRef: v1.CrossVersionObjectReference{
							Name:       "old-gardener-secret",
							Kind:       "Secret",
							APIVersion: "v1",
						},
					},
				},
			},
		}

		scheme, _ := newCreateTestScheme()
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(shoot).Build()

		testFsm := &fsm{
			K8s: K8s{
				GardenClient: fakeClient,
			},
		}

		systemState := &systemState{
			shoot: shoot,
		}

		// when
		err := patchShootAuditLog(ctx, testFsm, systemState, auditLogData)

		// then
		require.NoError(t, err)

		// Verify only one resource reference exists (updated, not duplicated)
		require.Len(t, systemState.shoot.Spec.Resources, 1)
		require.Equal(t, dedicatedAuditlogSecretReference, systemState.shoot.Spec.Resources[0].Name)
		require.Equal(t, "new-gardener-secret", systemState.shoot.Spec.Resources[0].ResourceRef.Name)
	})

	t.Run("should return error when audit log extension not found", func(t *testing.T) {
		// given
		ctx := context.Background()
		auditLogData := auditlog.AuditLogData{
			TenantID:   "test-tenant-id",
			ServiceURL: "https://test.auditlog.example.com",
			SecretName: "test-gardener-secret",
		}

		shoot := &gardener.Shoot{
			Spec: gardener.ShootSpec{
				Extensions: []gardener.Extension{
					{
						Type: "some-other-extension",
					},
				},
			},
		}

		scheme, _ := newCreateTestScheme()
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(shoot).Build()

		testFsm := &fsm{
			K8s: K8s{
				GardenClient: fakeClient,
			},
		}

		systemState := &systemState{
			shoot: shoot,
		}

		// when
		err := patchShootAuditLog(ctx, testFsm, systemState, auditLogData)

		// then
		require.Error(t, err)
		require.Contains(t, err.Error(), "audit log extension not found")
	})
}

func TestUpdateAuditLogExtension(t *testing.T) {
	t.Run("should update extension config with dedicated settings", func(t *testing.T) {
		// given
		existingConfig := extensions.AuditlogExtensionConfig{
			Type:                "standard",
			TenantID:            "old-tenant",
			ServiceURL:          "https://old.example.com",
			SecretReferenceName: "auditlog-credentials",
		}
		configJSON, _ := json.Marshal(existingConfig)

		ext := &gardener.Extension{
			Type: extensions.AuditlogExtensionType,
			ProviderConfig: &runtime.RawExtension{
				Raw: configJSON,
			},
		}

		auditLogData := auditlog.AuditLogData{
			TenantID:   "new-tenant",
			ServiceURL: "https://new.example.com",
			SecretName: "new-secret",
		}

		// when
		err := updateAuditLogExtension(ext, auditLogData)

		// then
		require.NoError(t, err)

		var updatedConfig extensions.AuditlogExtensionConfig
		err = json.Unmarshal(ext.ProviderConfig.Raw, &updatedConfig)
		require.NoError(t, err)

		require.Equal(t, "new-tenant", updatedConfig.TenantID)
		require.Equal(t, "https://new.example.com", updatedConfig.ServiceURL)
		require.Equal(t, dedicatedAuditlogSecretReference, updatedConfig.SecretReferenceName)
		require.Equal(t, "standard", updatedConfig.Type)
	})

	t.Run("should return error when provider config is nil", func(t *testing.T) {
		// given
		ext := &gardener.Extension{
			Type:           extensions.AuditlogExtensionType,
			ProviderConfig: nil,
		}

		auditLogData := auditlog.AuditLogData{
			TenantID:   "test-tenant",
			ServiceURL: "https://test.example.com",
			SecretName: "test-secret",
		}

		// when
		err := updateAuditLogExtension(ext, auditLogData)

		// then
		require.Error(t, err)
		require.Contains(t, err.Error(), "provider config is nil")
	})
}

func TestGetShootAuditLogConfig(t *testing.T) {
	t.Run("should extract audit log config with secret name from resources", func(t *testing.T) {
		// given
		config := extensions.AuditlogExtensionConfig{
			Type:                "standard",
			TenantID:            "test-tenant-id",
			ServiceURL:          "https://test.auditlog.example.com",
			SecretReferenceName: dedicatedAuditlogSecretReference,
		}
		configJSON, _ := json.Marshal(config)

		shoot := &gardener.Shoot{
			Spec: gardener.ShootSpec{
				Extensions: []gardener.Extension{
					{
						Type: extensions.AuditlogExtensionType,
						ProviderConfig: &runtime.RawExtension{
							Raw: configJSON,
						},
					},
				},
				Resources: []gardener.NamedResourceReference{
					{
						Name: dedicatedAuditlogSecretReference,
						ResourceRef: v1.CrossVersionObjectReference{
							Name:       "actual-gardener-secret",
							Kind:       "Secret",
							APIVersion: "v1",
						},
					},
				},
			},
		}

		// when
		auditLogData, err := getShootAuditLogConfig(shoot)

		// then
		require.NoError(t, err)
		require.NotNil(t, auditLogData)
		require.Equal(t, "test-tenant-id", auditLogData.TenantID)
		require.Equal(t, "https://test.auditlog.example.com", auditLogData.ServiceURL)
		require.Equal(t, "actual-gardener-secret", auditLogData.SecretName)
	})

	t.Run("should return error when resource reference not found", func(t *testing.T) {
		// given
		config := extensions.AuditlogExtensionConfig{
			Type:                "standard",
			TenantID:            "test-tenant-id",
			ServiceURL:          "https://test.auditlog.example.com",
			SecretReferenceName: dedicatedAuditlogSecretReference,
		}
		configJSON, _ := json.Marshal(config)

		shoot := &gardener.Shoot{
			Spec: gardener.ShootSpec{
				Extensions: []gardener.Extension{
					{
						Type: extensions.AuditlogExtensionType,
						ProviderConfig: &runtime.RawExtension{
							Raw: configJSON,
						},
					},
				},
				Resources: []gardener.NamedResourceReference{
					{
						Name: "some-other-resource",
						ResourceRef: v1.CrossVersionObjectReference{
							Name:       "other-secret",
							Kind:       "Secret",
							APIVersion: "v1",
						},
					},
				},
			},
		}

		// when
		auditLogData, err := getShootAuditLogConfig(shoot)

		// then
		require.Error(t, err)
		require.Nil(t, auditLogData)
		require.Contains(t, err.Error(), "resource reference 'dedicated-auditlog-credentials' not found")
	})

	t.Run("should return error when audit log extension not found", func(t *testing.T) {
		// given
		shoot := &gardener.Shoot{
			Spec: gardener.ShootSpec{
				Extensions: []gardener.Extension{
					{
						Type: "some-other-extension",
					},
				},
			},
		}

		// when
		auditLogData, err := getShootAuditLogConfig(shoot)

		// then
		require.Error(t, err)
		require.Nil(t, auditLogData)
		require.Contains(t, err.Error(), "audit log extension not found")
	})
}

func TestGetSecretNameFromResources(t *testing.T) {
	t.Run("should find secret name from resources", func(t *testing.T) {
		// given
		shoot := &gardener.Shoot{
			Spec: gardener.ShootSpec{
				Resources: []gardener.NamedResourceReference{
					{
						Name: "other-resource",
						ResourceRef: v1.CrossVersionObjectReference{
							Name:       "other-secret",
							Kind:       "Secret",
							APIVersion: "v1",
						},
					},
					{
						Name: dedicatedAuditlogSecretReference,
						ResourceRef: v1.CrossVersionObjectReference{
							Name:       "actual-secret-name",
							Kind:       "Secret",
							APIVersion: "v1",
						},
					},
				},
			},
		}

		// when
		secretName, err := getSecretNameFromResources(shoot, dedicatedAuditlogSecretReference)

		// then
		require.NoError(t, err)
		require.Equal(t, "actual-secret-name", secretName)
	})

	t.Run("should return error when resource reference not found", func(t *testing.T) {
		// given
		shoot := &gardener.Shoot{
			Spec: gardener.ShootSpec{
				Resources: []gardener.NamedResourceReference{
					{
						Name: "other-resource",
						ResourceRef: v1.CrossVersionObjectReference{
							Name:       "other-secret",
							Kind:       "Secret",
							APIVersion: "v1",
						},
					},
				},
			},
		}

		// when
		secretName, err := getSecretNameFromResources(shoot, "non-existent-reference")

		// then
		require.Error(t, err)
		require.Empty(t, secretName)
		require.Contains(t, err.Error(), "resource reference 'non-existent-reference' not found")
	})
}
