package fsm

import (
	"context"
	"encoding/json"
	"testing"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/kyma-project/infrastructure-manager/pkg/auditlog"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/auditlogs"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/extensions"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	gardenerSharedSecretName    = "shared-secret"
	gardenerDedicatedSecretName = "dedicated-secret"
)

func TestPatchShootAuditLog(t *testing.T) {
	t.Run("should update audit log extension and add resource reference", func(t *testing.T) {
		// given
		ctx := context.Background()
		auditLogData := auditlog.AuditLogData{
			TenantID:   "test-tenant-id",
			ServiceURL: "https://test.auditlog.example.com",
			SecretName: gardenerDedicatedSecretName,
		}

		// Create initial shoot with audit log extension
		existingConfig := extensions.AuditlogExtensionConfig{
			Type:                "standard",
			TenantID:            "old-tenant-id",
			ServiceURL:          "https://old.example.com",
			SecretReferenceName: auditlogs.SharedAuditlogSecretReference,
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
						Name: auditlogs.SharedAuditlogSecretReference,
						ResourceRef: v1.CrossVersionObjectReference{
							Name:       gardenerSharedSecretName,
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
		require.Equal(t, auditlogs.DedicatedAuditlogSecretReference, updatedConfig.SecretReferenceName)
		require.Equal(t, "standard", updatedConfig.Type)

		// Verify the resource reference was added and both references are present
		require.Len(t, systemState.shoot.Spec.Resources, 2)

		// Find the dedicated and shared audit log resource references
		dedicatedResource := auditlogResource(systemState, auditlogs.DedicatedAuditlogSecretReference)
		sharedResource := auditlogResource(systemState, auditlogs.SharedAuditlogSecretReference)

		require.NotNil(t, dedicatedResource)
		require.Equal(t, auditlogs.DedicatedAuditlogSecretReference, dedicatedResource.Name)
		require.Equal(t, gardenerDedicatedSecretName, dedicatedResource.ResourceRef.Name)
		require.Equal(t, "Secret", dedicatedResource.ResourceRef.Kind)
		require.Equal(t, "v1", dedicatedResource.ResourceRef.APIVersion)

		require.NotNil(t, sharedResource)
		require.Equal(t, auditlogs.SharedAuditlogSecretReference, sharedResource.Name)
		require.Equal(t, gardenerSharedSecretName, sharedResource.ResourceRef.Name)
	})
	t.Run("should update existing dedicated resource reference", func(t *testing.T) {
		// given
		const expectedDedicatedSecretName = "new-dedicated-secret"
		ctx := context.Background()
		auditLogData := auditlog.AuditLogData{
			TenantID:   "new-tenant-id",
			ServiceURL: "https://new.auditlog.example.com",
			SecretName: expectedDedicatedSecretName,
		}

		existingConfig := extensions.AuditlogExtensionConfig{
			Type:                "standard",
			TenantID:            "old-tenant-id",
			ServiceURL:          "https://old.example.com",
			SecretReferenceName: auditlogs.DedicatedAuditlogSecretReference,
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
						Name: auditlogs.DedicatedAuditlogSecretReference,
						ResourceRef: v1.CrossVersionObjectReference{
							Name:       gardenerDedicatedSecretName,
							Kind:       "Secret",
							APIVersion: "v1",
						},
					},
					{
						Name: auditlogs.SharedAuditlogSecretReference,
						ResourceRef: v1.CrossVersionObjectReference{
							Name:       gardenerSharedSecretName,
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

		// Verify both resource references exist (updated, not duplicated)
		require.Len(t, systemState.shoot.Spec.Resources, 2)

		dedicatedResource := auditlogResource(systemState, auditlogs.DedicatedAuditlogSecretReference)
		sharedResource := auditlogResource(systemState, auditlogs.SharedAuditlogSecretReference)

		require.NotNil(t, dedicatedResource)
		require.Equal(t, auditlogs.DedicatedAuditlogSecretReference, dedicatedResource.Name)
		require.Equal(t, expectedDedicatedSecretName, dedicatedResource.ResourceRef.Name)

		require.NotNil(t, sharedResource)
		require.Equal(t, auditlogs.SharedAuditlogSecretReference, sharedResource.Name)
		require.Equal(t, gardenerSharedSecretName, sharedResource.ResourceRef.Name)
	})

	t.Run("should create audit log extension when not found and add resource references", func(t *testing.T) {
		// given
		ctx := context.Background()
		auditLogData := auditlog.AuditLogData{
			TenantID:   "test-tenant-id",
			ServiceURL: "https://test.auditlog.example.com",
			SecretName: gardenerDedicatedSecretName,
		}

		shoot := &gardener.Shoot{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-shoot",
				Namespace: "garden-test",
			},
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
		require.NoError(t, err)

		// Verify the audit log extension was created
		require.Len(t, systemState.shoot.Spec.Extensions, 2)

		// Find the audit log extension
		var auditlogExt *gardener.Extension
		for i := range systemState.shoot.Spec.Extensions {
			if systemState.shoot.Spec.Extensions[i].Type == extensions.AuditlogExtensionType {
				auditlogExt = &systemState.shoot.Spec.Extensions[i]
				break
			}
		}

		require.NotNil(t, auditlogExt)
		require.Equal(t, extensions.AuditlogExtensionType, auditlogExt.Type)

		var createdConfig extensions.AuditlogExtensionConfig
		err = json.Unmarshal(auditlogExt.ProviderConfig.Raw, &createdConfig)
		require.NoError(t, err)
		require.Equal(t, "test-tenant-id", createdConfig.TenantID)
		require.Equal(t, "https://test.auditlog.example.com", createdConfig.ServiceURL)
		require.Equal(t, auditlogs.DedicatedAuditlogSecretReference, createdConfig.SecretReferenceName)
		require.Equal(t, "standard", createdConfig.Type)

		// Verify the resource reference was added and both exist
		require.Len(t, systemState.shoot.Spec.Resources, 2)

		dedicatedResource := auditlogResource(systemState, auditlogs.DedicatedAuditlogSecretReference)
		sharedResource := auditlogResource(systemState, auditlogs.SharedAuditlogSecretReference)

		require.NotNil(t, dedicatedResource)
		require.Equal(t, auditlogs.DedicatedAuditlogSecretReference, dedicatedResource.Name)
		require.Equal(t, gardenerDedicatedSecretName, dedicatedResource.ResourceRef.Name)

		require.NotNil(t, sharedResource)
		require.Equal(t, auditlogs.SharedAuditlogSecretReference, sharedResource.Name)
		require.Equal(t, gardenerDedicatedSecretName, sharedResource.ResourceRef.Name) // The shared resource reference is created with the same secret name as the dedicated one in this case
	})

	t.Run("should create audit log extension when shoot has no extensions", func(t *testing.T) {
		// given
		ctx := context.Background()
		auditLogData := auditlog.AuditLogData{
			TenantID:   "test-tenant-id",
			ServiceURL: "https://test.auditlog.example.com",
			SecretName: gardenerDedicatedSecretName,
		}

		shoot := &gardener.Shoot{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-shoot",
				Namespace: "garden-test",
			},
			Spec: gardener.ShootSpec{
				Extensions: []gardener.Extension{},
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

		// Verify the audit log extension was created
		require.Len(t, systemState.shoot.Spec.Extensions, 1)
		ext := systemState.shoot.Spec.Extensions[0]
		require.Equal(t, extensions.AuditlogExtensionType, ext.Type)

		var createdConfig extensions.AuditlogExtensionConfig
		err = json.Unmarshal(ext.ProviderConfig.Raw, &createdConfig)
		require.NoError(t, err)
		require.Equal(t, "test-tenant-id", createdConfig.TenantID)
		require.Equal(t, "https://test.auditlog.example.com", createdConfig.ServiceURL)
		require.Equal(t, auditlogs.DedicatedAuditlogSecretReference, createdConfig.SecretReferenceName)
		require.Equal(t, "standard", createdConfig.Type)

		require.Len(t, systemState.shoot.Spec.Resources, 2)

		dedicatedResource := auditlogResource(systemState, auditlogs.DedicatedAuditlogSecretReference)
		sharedResource := auditlogResource(systemState, auditlogs.SharedAuditlogSecretReference)

		require.NotNil(t, dedicatedResource)
		require.Equal(t, auditlogs.DedicatedAuditlogSecretReference, dedicatedResource.Name)
		require.Equal(t, gardenerDedicatedSecretName, dedicatedResource.ResourceRef.Name)

		require.NotNil(t, sharedResource)
		require.Equal(t, auditlogs.SharedAuditlogSecretReference, sharedResource.Name)
		require.Equal(t, gardenerDedicatedSecretName, sharedResource.ResourceRef.Name)
	})

	t.Run("should keep both shared and dedicated resource entries valid when migrating to dedicated", func(t *testing.T) {
		// given
		const expectedSecretName = "new-dedicated-secret"
		ctx := context.Background()
		auditLogData := auditlog.AuditLogData{
			TenantID:   "test-tenant-id",
			ServiceURL: "https://test.auditlog.example.com",
			SecretName: expectedSecretName,
		}

		existingConfig := extensions.AuditlogExtensionConfig{
			Type:                "standard",
			TenantID:            "old-tenant-id",
			ServiceURL:          "https://old.example.com",
			SecretReferenceName: auditlogs.SharedAuditlogSecretReference,
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
						Name: auditlogs.SharedAuditlogSecretReference,
						ResourceRef: v1.CrossVersionObjectReference{
							Name:       gardenerSharedSecretName,
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

		// Verify audit log extension configuration
		require.Len(t, systemState.shoot.Spec.Extensions, 1)
		ext := systemState.shoot.Spec.Extensions[0]
		require.Equal(t, extensions.AuditlogExtensionType, ext.Type)

		var updatedConfig extensions.AuditlogExtensionConfig
		err = json.Unmarshal(ext.ProviderConfig.Raw, &updatedConfig)
		require.NoError(t, err)
		require.Equal(t, auditlogs.DedicatedAuditlogSecretReference, updatedConfig.SecretReferenceName)

		// Verify both shared and dedicated resource references remain present
		require.Len(t, systemState.shoot.Spec.Resources, 2)

		dedicatedResource := auditlogResource(systemState, auditlogs.DedicatedAuditlogSecretReference)
		sharedResource := auditlogResource(systemState, auditlogs.SharedAuditlogSecretReference)

		require.NotNil(t, dedicatedResource)
		require.Equal(t, expectedSecretName, dedicatedResource.ResourceRef.Name)

		require.NotNil(t, sharedResource, "shared auditlog-credentials resource entry should be kept")
		require.Equal(t, gardenerSharedSecretName, sharedResource.ResourceRef.Name)
	})
}

func auditlogResource(s *systemState, resourceName string) *gardener.NamedResourceReference {
	var resource *gardener.NamedResourceReference
	for i := range s.shoot.Spec.Resources {
		if s.shoot.Spec.Resources[i].Name == resourceName {
			resource = &s.shoot.Spec.Resources[i]
			break
		}
	}
	return resource
}

func TestUpdateAuditLogExtension(t *testing.T) {
	t.Run("should update extension config with dedicated settings", func(t *testing.T) {
		// given
		existingConfig := extensions.AuditlogExtensionConfig{
			Type:                "standard",
			TenantID:            "old-tenant",
			ServiceURL:          "https://old.example.com",
			SecretReferenceName: auditlogs.SharedAuditlogSecretReference,
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
		require.Equal(t, auditlogs.DedicatedAuditlogSecretReference, updatedConfig.SecretReferenceName)
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
			SecretReferenceName: auditlogs.DedicatedAuditlogSecretReference,
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
						Name: auditlogs.DedicatedAuditlogSecretReference,
						ResourceRef: v1.CrossVersionObjectReference{
							Name:       gardenerDedicatedSecretName,
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
		require.Equal(t, gardenerDedicatedSecretName, auditLogData.SecretName)
	})

	t.Run("should return error when resource reference not found", func(t *testing.T) {
		// given
		config := extensions.AuditlogExtensionConfig{
			Type:                "standard",
			TenantID:            "test-tenant-id",
			ServiceURL:          "https://test.auditlog.example.com",
			SecretReferenceName: auditlogs.DedicatedAuditlogSecretReference,
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
						Name: auditlogs.DedicatedAuditlogSecretReference,
						ResourceRef: v1.CrossVersionObjectReference{
							Name:       gardenerDedicatedSecretName,
							Kind:       "Secret",
							APIVersion: "v1",
						},
					},
				},
			},
		}

		// when
		secretName, err := getSecretNameFromResources(shoot, auditlogs.DedicatedAuditlogSecretReference)

		// then
		require.NoError(t, err)
		require.Equal(t, gardenerDedicatedSecretName, secretName)
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
