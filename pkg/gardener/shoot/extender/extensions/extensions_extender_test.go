package extensions

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	registrycacheext "github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha3"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/hyperscaler"
	registrycache "github.com/kyma-project/registry-cache/api/v1beta1"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/yaml"

	"testing"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/config"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/auditlogs"
	"github.com/stretchr/testify/assert"
)

func TestNewExtensionsExtenderForCreate(t *testing.T) {
	config := config.ConverterConfig{
		DNS: config.DNSConfig{
			SecretName:   "test-dns-secret",
			DomainPrefix: "test-domain",
			ProviderType: "test-provider",
		},
		Kubernetes: config.KubernetesConfig{
			KubeApiServer: config.KubeApiServer{
				ACL: config.ACL{
					ConfigMapName: "acl-ip-list",
				},
			},
		},
	}

	newAuditLogData := auditlogs.AuditLogData{
		TenantID:   "test-auditlog-tenant",
		ServiceURL: "test-auditlog-service-url",
		SecretName: "doesnt matter",
	}

	registryCache := []imv1.ImageRegistryCache{
		{
			UID: "id1",
			Config: registrycache.RegistryCacheConfigSpec{
				Upstream: "ghcr.io",
			},
		},
	}

	for _, testcase := range []struct {
		name                string
		inputAuditLogData   auditlogs.AuditLogData
		enableNetworkFilter bool
		registryCache       []imv1.ImageRegistryCache
		apiServerACL        []string
		apiServerACLEnabled bool
		extensionOrderMap   map[string]int
		providerType        string
	}{
		{
			name:                "Should create all extensions for new Shoot in the right order, network filter is enabled",
			inputAuditLogData:   newAuditLogData,
			enableNetworkFilter: true,
			registryCache:       registryCache,
			apiServerACL:        []string{"1.1.1.1/32", "2.2.2.2/32"},
			apiServerACLEnabled: true,
			extensionOrderMap:   getExpectedExtensionsOrderMapForCreate(),
			providerType:        hyperscaler.TypeAWS,
		},
		{
			name:                "Should create all extensions for new Shoot in the right order, network filter is disabled",
			inputAuditLogData:   newAuditLogData,
			enableNetworkFilter: false,
			registryCache:       registryCache,
			apiServerACL:        []string{"1.1.1.1/32", "2.2.2.2/32"},
			apiServerACLEnabled: true,
			extensionOrderMap:   getExpectedExtensionsOrderMapForCreate(),
			providerType:        hyperscaler.TypeAzure,
		},
		{
			name:                "Should not include AuditLog extension for new Shoot when input auditLogData is empty",
			inputAuditLogData:   auditlogs.AuditLogData{},
			extensionOrderMap:   getExpectedExtensionsOrderMapForCreateWithoutOptional(),
			providerType:        hyperscaler.TypeAWS,
			apiServerACLEnabled: false,
		},
		{
			name:                "Should not include ACL extension for new Shoot when feature flag in disabled",
			inputAuditLogData:   auditlogs.AuditLogData{},
			apiServerACL:        []string{"1.1.1.1/32", "2.2.2.2/32"},
			apiServerACLEnabled: false,
			extensionOrderMap:   getExpectedExtensionsOrderMapForCreateWithoutOptional(),
			providerType:        hyperscaler.TypeAWS,
		},
		{
			name:                "Should not include ACL extension for new Shoot when ACL is empty on Runtime CR",
			inputAuditLogData:   auditlogs.AuditLogData{},
			apiServerACL:        []string{},
			apiServerACLEnabled: true,
			extensionOrderMap:   getExpectedExtensionsOrderMapForCreateWithoutOptional(),
			providerType:        hyperscaler.TypeAWS,
		},
		{
			name:                "Should not include ACL extension for new Shoot when hyperscaler type is not supported",
			inputAuditLogData:   auditlogs.AuditLogData{},
			apiServerACL:        []string{"1.1.1.1/32", "2.2.2.2/32"},
			apiServerACLEnabled: true,
			extensionOrderMap:   getExpectedExtensionsOrderMapForCreateWithoutOptional(),
			providerType:        hyperscaler.TypeGCP,
		},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			providerType := testcase.providerType
			testRuntime := fixRuntimeCRForExtensionExtenderTests(testcase.enableNetworkFilter, testcase.registryCache, testcase.apiServerACL, providerType)

			configMapGetCalled := false
			fakeClient := buildFakeClientWithACLConfigMap(t, &configMapGetCalled)

			shoot := &gardener.Shoot{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-shoot-name",
				},
			}

			extender := NewExtensionsExtenderForCreate(config, fakeClient, context.Background(), testcase.inputAuditLogData, testcase.registryCache, testcase.apiServerACLEnabled)

			err := extender(testRuntime, shoot)
			assert.NoError(t, err)
			assert.NotNil(t, shoot.Spec.Extensions)
			assert.Equal(t, aclNeedsToBeEnabled(testcase.apiServerACLEnabled, testRuntime), configMapGetCalled)

			orderMap := testcase.extensionOrderMap
			require.Len(t, shoot.Spec.Extensions, len(orderMap))

			for idx, ext := range shoot.Spec.Extensions {
				assert.NotEmpty(t, ext.Type)
				assert.Equal(t, orderMap[ext.Type], idx)

				switch ext.Type {
				case NetworkFilterType:
					verifyNetworkFilterExtension(t, ext, testcase.enableNetworkFilter)

				case CertExtensionType:
					verifyCertExtension(t, ext)

				case DNSExtensionType:
					verifyDNSExtension(t, ext)

				case OidcExtensionType:
					verifyOIDCExtension(t, ext)

				case RegistryCacheExtensionType:
					verifyRegistryCacheExtension(t, &ext, testcase.registryCache)
				case ApiServerACLExtensionType:
					mergedACL := testcase.apiServerACL
					mergedACL = append(mergedACL, "2.2.2.2/29", "3.3.3.3/29", "4.4.4.4/29")
					mergedACL = append(mergedACL, "1.1.1.1/32")

					verifyACLExtension(t, &ext, mergedACL)
				}
			}
		})
	}
}

func TestNewExtensionsExtenderForPatch(t *testing.T) {
	config := config.ConverterConfig{
		DNS: config.DNSConfig{
			SecretName:   "test-dns-secret",
			DomainPrefix: "test-domain",
			ProviderType: "test-provider",
		},
		Kubernetes: config.KubernetesConfig{
			KubeApiServer: config.KubeApiServer{
				ACL: config.ACL{
					ConfigMapName: "acl-ip-list",
				},
			},
		},
	}

	oldAuditLogData := auditlogs.AuditLogData{
		TenantID:   "test-auditlog-tenant",
		ServiceURL: "test-auditlog-service-url",
		SecretName: "doesnt matter",
	}

	newAuditLogData := auditlogs.AuditLogData{
		TenantID:   "test-auditlog-new-tenant",
		ServiceURL: "test-auditlog-new-service",
		SecretName: "doesnt matter",
	}

	oldCaches := []imv1.ImageRegistryCache{
		{
			Config: registrycache.RegistryCacheConfigSpec{Upstream: "quay.io"},
		},
	}

	newCaches := []imv1.ImageRegistryCache{
		{
			Config: registrycache.RegistryCacheConfigSpec{Upstream: "gcr.io"},
		},
	}

	for _, testCase := range []struct {
		name                 string
		previousExtensions   []gardener.Extension
		inputAuditLogData    auditlogs.AuditLogData
		expectedAuditLogData auditlogs.AuditLogData
		registryCaches       []imv1.ImageRegistryCache
		enableNetworkFilter  bool
		apiServerACL         []string
		apiServerACLEnabled  bool
		providerType         string
	}{
		{
			name:                 "Should add AuditLog extension at the end without changing order and data of other extensions",
			previousExtensions:   []gardener.Extension{fixNetworkExtension(), fixDNSExtension(), fixCertExtension(), fixOIDCExtensions()},
			inputAuditLogData:    oldAuditLogData,
			expectedAuditLogData: oldAuditLogData,
			registryCaches:       nil,
			enableNetworkFilter:  false,
		},
		{
			name:                 "Should not add AuditLog extension to existing shoot extensions when input auditLogData is empty",
			previousExtensions:   []gardener.Extension{fixNetworkExtension(), fixDNSExtension(), fixCertExtension(), fixOIDCExtensions()},
			inputAuditLogData:    auditlogs.AuditLogData{},
			expectedAuditLogData: auditlogs.AuditLogData{},
			registryCaches:       nil,
			enableNetworkFilter:  false,
		},
		{
			name:                 "Should add Network filter extension at the end without changing order and data of other extensions",
			previousExtensions:   []gardener.Extension{fixDNSExtension(), fixCertExtension(), fixOIDCExtensions()},
			inputAuditLogData:    auditlogs.AuditLogData{},
			expectedAuditLogData: auditlogs.AuditLogData{},
			registryCaches:       nil,
			enableNetworkFilter:  true,
		},
		{
			name:                 "Should add RegistryCache extension at the end without changing order and data of other extensions",
			previousExtensions:   []gardener.Extension{fixNetworkExtension(), fixDNSExtension(), fixCertExtension(), fixOIDCExtensions()},
			inputAuditLogData:    auditlogs.AuditLogData{},
			expectedAuditLogData: auditlogs.AuditLogData{},
			registryCaches:       newCaches,
			enableNetworkFilter:  false,
		},
		{
			name:                 "Should not add RegistryCache extension when cache list is empty",
			previousExtensions:   []gardener.Extension{fixNetworkExtension(), fixDNSExtension(), fixCertExtension(), fixOIDCExtensions()},
			inputAuditLogData:    auditlogs.AuditLogData{},
			expectedAuditLogData: auditlogs.AuditLogData{},
			registryCaches:       []imv1.ImageRegistryCache{},
			enableNetworkFilter:  false,
		},
		{
			name:                 "Should not add RegistryCache extension when cache is not enabled on Runtime CR",
			previousExtensions:   []gardener.Extension{fixNetworkExtension(), fixDNSExtension(), fixCertExtension(), fixOIDCExtensions()},
			inputAuditLogData:    auditlogs.AuditLogData{},
			expectedAuditLogData: auditlogs.AuditLogData{},
			registryCaches:       newCaches,
			enableNetworkFilter:  false,
		},
		{
			name:                 "Existing extensions should not change order during patching if nothing has changed",
			previousExtensions:   fixAllExtensionsOnTheShoot(true),
			inputAuditLogData:    oldAuditLogData,
			expectedAuditLogData: oldAuditLogData,
			registryCaches:       newCaches,
			enableNetworkFilter:  true,
		},
		{
			name:                 "Should update Audit Log extension without changing order and data of other extensions",
			previousExtensions:   fixAllExtensionsOnTheShoot(true),
			inputAuditLogData:    newAuditLogData,
			expectedAuditLogData: newAuditLogData,
			registryCaches:       oldCaches,
			enableNetworkFilter:  false,
		},
		{
			name:                 "Should update Network filter extension without changing order and data of other extensions",
			previousExtensions:   fixAllExtensionsOnTheShoot(true),
			inputAuditLogData:    oldAuditLogData,
			expectedAuditLogData: oldAuditLogData,
			registryCaches:       oldCaches,
			enableNetworkFilter:  true,
		},
		{
			name:                 "Should update RegistryCache extension without changing order and data of other extensions",
			previousExtensions:   fixAllExtensionsOnTheShoot(true),
			inputAuditLogData:    oldAuditLogData,
			expectedAuditLogData: oldAuditLogData,
			registryCaches:       newCaches,
			enableNetworkFilter:  false,
		},
		{
			name:                 "Should disable RegistryCache extension when cache is not enabled on Runtime CR without changing order and data of other extensions",
			previousExtensions:   fixAllExtensionsOnTheShoot(true),
			inputAuditLogData:    oldAuditLogData,
			expectedAuditLogData: oldAuditLogData,
			registryCaches:       newCaches,
			enableNetworkFilter:  false,
		},
		{
			name:                 "Should disable RegistryCache extension when cache is not enabled on Runtime CR without changing order and data of other extensions",
			previousExtensions:   fixAllExtensionsOnTheShoot(true),
			inputAuditLogData:    oldAuditLogData,
			expectedAuditLogData: oldAuditLogData,
			registryCaches:       []imv1.ImageRegistryCache{},
			enableNetworkFilter:  false,
		},
		{
			name:                 "Should not update existing AuditLog extension when input auditLogData is empty",
			previousExtensions:   fixAllExtensionsOnTheShoot(true),
			inputAuditLogData:    auditlogs.AuditLogData{},
			expectedAuditLogData: oldAuditLogData,
			registryCaches:       oldCaches,
			enableNetworkFilter:  false,
		},
		{
			name:                 "Should update ACL extension without changing order and data of other extensions",
			previousExtensions:   fixAllExtensionsOnTheShoot(true),
			inputAuditLogData:    oldAuditLogData,
			expectedAuditLogData: oldAuditLogData,
			registryCaches:       newCaches,
			enableNetworkFilter:  false,
			apiServerACL:         []string{"1.1.1.1/32", "2.2.2.2/32"},
			apiServerACLEnabled:  true,
			providerType:         hyperscaler.TypeAWS,
		},
		{
			name:                 "Should disable ACL extension without changing order and data of other extensions when acl is empty on Runtime CR",
			previousExtensions:   fixAllExtensionsOnTheShoot(true),
			inputAuditLogData:    oldAuditLogData,
			expectedAuditLogData: oldAuditLogData,
			registryCaches:       newCaches,
			enableNetworkFilter:  false,
			apiServerACL:         []string{},
			apiServerACLEnabled:  true,
			providerType:         hyperscaler.TypeAWS,
		},
		{
			name:                 "Should not add ACL extension when acl is disabled",
			previousExtensions:   fixAllExtensionsOnTheShoot(false),
			inputAuditLogData:    oldAuditLogData,
			expectedAuditLogData: oldAuditLogData,
			registryCaches:       newCaches,
			enableNetworkFilter:  false,
			apiServerACL:         []string{"1.1.1.1/32", "2.2.2.2/32"},
			apiServerACLEnabled:  false,
			providerType:         hyperscaler.TypeAWS,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			testRuntime := fixRuntimeCRForExtensionExtenderTests(testCase.enableNetworkFilter, testCase.registryCaches, testCase.apiServerACL, testCase.providerType)

			configMapGetCalled := false
			fakeClient := buildFakeClientWithACLConfigMap(t, &configMapGetCalled)

			shoot := &gardener.Shoot{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-shoot-name",
				},
			}

			auditLogDataProvided := testCase.inputAuditLogData != (auditlogs.AuditLogData{})
			registryCacheDataProvided := len(testCase.registryCaches) != 0
			kubeApiServerACLEnabled := aclNeedsToBeEnabled(testCase.apiServerACLEnabled, testRuntime)

			extender := NewExtensionsExtenderForPatch(config, fakeClient, context.Background(), testCase.inputAuditLogData, testCase.previousExtensions, testCase.apiServerACLEnabled)
			orderMap := getExpectedExtensionsOrderMapForPatch(testCase.previousExtensions, testCase.enableNetworkFilter, auditLogDataProvided, registryCacheDataProvided, kubeApiServerACLEnabled)

			err := extender(testRuntime, shoot)
			assert.NoError(t, err)
			assert.NotNil(t, shoot.Spec.Extensions)
			require.Len(t, shoot.Spec.Extensions, len(orderMap))
			assert.Equal(t, kubeApiServerACLEnabled, configMapGetCalled)

			for idx, ext := range shoot.Spec.Extensions {
				assert.NotEmpty(t, ext.Type)
				assert.Equal(t, orderMap[ext.Type], idx)

				switch ext.Type {
				case NetworkFilterType:
					verifyNetworkFilterExtension(t, ext, testCase.enableNetworkFilter)

				case CertExtensionType:
					verifyCertExtension(t, ext)

				case DNSExtensionType:
					verifyDNSExtension(t, ext)

				case OidcExtensionType:
					verifyOIDCExtension(t, ext)

				case AuditlogExtensionType:
					verifyAuditLogExtension(t, ext, testCase.expectedAuditLogData)

				case RegistryCacheExtensionType:
					verifyRegistryCacheExtension(t, &ext, testCase.registryCaches)
				case ApiServerACLExtensionType:
					mergedACL := make([]string, 0)
					if len(testCase.apiServerACL) != 0 {
						mergedACL = append(mergedACL, testCase.apiServerACL...)
						mergedACL = append(mergedACL, "2.2.2.2/29", "3.3.3.3/29", "4.4.4.4/29")
						mergedACL = append(mergedACL, "1.1.1.1/32")
					}

					verifyACLExtension(t, &ext, mergedACL)

				}
			}
		})
	}
}

func fixAllExtensionsOnTheShoot(aclEnabled bool) []gardener.Extension {
	extensions := []gardener.Extension{
		fixAuditLogExtensions(),
		fixDNSExtension(),
		fixCertExtension(),
		fixNetworkExtension(),
		fixOIDCExtensions(),
		fixRegistryCacheExtension(),
	}

	if aclEnabled {
		extensions = append(extensions, fixKubeApiServerACLExtension())
	}

	return extensions
}

func fixAuditLogExtensions() gardener.Extension {
	return gardener.Extension{
		Type: AuditlogExtensionType,
		ProviderConfig: &runtime.RawExtension{
			Raw: []byte(`{"apiVersion":"service.auditlog.extensions.gardener.cloud/v1alpha1","kind":"AuditlogConfig","type":"standard","tenantID":"test-auditlog-tenant","serviceURL":"test-auditlog-service-url","secretReferenceName":"auditlog-credentials"}`),
		},
	}
}

func fixDNSExtension() gardener.Extension {
	return gardener.Extension{
		Type: DNSExtensionType,
		ProviderConfig: &runtime.RawExtension{
			Raw: []byte(`{"apiVersion":"service.dns.extensions.gardener.cloud/v1alpha1","dnsProviderReplication":{"enabled":true},"syncProvidersFromShootSpecDNS":true,"providers":[{"domains":{"include":["test-shoot-name.test-domain"],"exclude":null},"secretName":"test-dns-secret","type":"test-provider"}],"kind":"DNSConfig"}`),
		},
	}
}

func fixCertExtension() gardener.Extension {
	return gardener.Extension{
		Type: CertExtensionType,
		ProviderConfig: &runtime.RawExtension{
			Raw: []byte(`{"apiVersion":"service.cert.extensions.gardener.cloud/v1alpha1","kind":"CertConfig","shootIssuers":{"enabled":true}}`),
		},
	}
}

func fixNetworkExtension() gardener.Extension {
	return gardener.Extension{
		Type:     NetworkFilterType,
		Disabled: ptr.To(true),
	}
}

func fixOIDCExtensions() gardener.Extension {
	return gardener.Extension{
		Type:     OidcExtensionType,
		Disabled: ptr.To(false),
	}
}

func fixRegistryCacheExtension() gardener.Extension {
	return gardener.Extension{
		Type:     RegistryCacheExtensionType,
		Disabled: ptr.To(false),
		ProviderConfig: &runtime.RawExtension{
			Raw: []byte(`{"apiVersion":"registry.extensions.gardener.cloud/v1alpha3","kind":"RegistryConfig","caches":[{"upstream":"quay.io"}]}`),
		},
	}
}

func fixKubeApiServerACLExtension() gardener.Extension {
	return gardener.Extension{
		Type:     ApiServerACLExtensionType,
		Disabled: ptr.To(false),
		ProviderConfig: &runtime.RawExtension{
			Raw: []byte(`{"rule": {"action": "ALLOW","type": "remote_ip", "cidrs": ["3.3.3.3/32", "4.4.4.4/32"]}}`),
		},
	}
}

func getExpectedExtensionsOrderMapForPatch(previousExtensions []gardener.Extension, networkExtAdded bool, auditLogExtAdded bool, registryCacheExtAdded bool, kubeApiServerACLEnabled bool) map[string]int {
	extensionOrderMap := make(map[string]int)

	for idx, ext := range previousExtensions {
		extensionOrderMap[ext.Type] = idx
	}

	if auditLogExtAdded {
		_, found := extensionOrderMap[AuditlogExtensionType]

		if !found {
			extensionOrderMap[AuditlogExtensionType] = len(extensionOrderMap)
		}
	}

	if networkExtAdded {
		_, found := extensionOrderMap[NetworkFilterType]
		if !found {
			extensionOrderMap[NetworkFilterType] = len(extensionOrderMap)
		}
	}

	if registryCacheExtAdded {
		_, found := extensionOrderMap[RegistryCacheExtensionType]

		if !found {
			extensionOrderMap[RegistryCacheExtensionType] = len(extensionOrderMap)
		}
	}

	if kubeApiServerACLEnabled {
		_, found := extensionOrderMap[ApiServerACLExtensionType]
		if !found {
			extensionOrderMap[ApiServerACLExtensionType] = len(extensionOrderMap)
		}
	}

	return extensionOrderMap
}

// returns a map with the expected index order of extensions for ExtenderForCreate create unit test
func getExpectedExtensionsOrderMapForCreate() map[string]int {
	extensionOrderMap := make(map[string]int)

	extensionOrderMap[NetworkFilterType] = 0
	extensionOrderMap[CertExtensionType] = 1
	extensionOrderMap[DNSExtensionType] = 2
	extensionOrderMap[OidcExtensionType] = 3
	extensionOrderMap[AuditlogExtensionType] = 4
	extensionOrderMap[RegistryCacheExtensionType] = 5
	extensionOrderMap[ApiServerACLExtensionType] = 6

	return extensionOrderMap
}

func getExpectedExtensionsOrderMapForCreateWithoutOptional() map[string]int {
	extensionOrderMap := make(map[string]int)

	extensionOrderMap[NetworkFilterType] = 0
	extensionOrderMap[CertExtensionType] = 1
	extensionOrderMap[DNSExtensionType] = 2
	extensionOrderMap[OidcExtensionType] = 3

	return extensionOrderMap
}

func verifyAuditLogExtension(t *testing.T, ext gardener.Extension, expected auditlogs.AuditLogData) {
	var auditlogConfig AuditlogExtensionConfig

	err := json.Unmarshal(ext.ProviderConfig.Raw, &auditlogConfig)
	require.NoError(t, err)

	assert.Equal(t, "standard", auditlogConfig.Type)
	assert.Equal(t, expected.TenantID, auditlogConfig.TenantID)
	assert.Equal(t, expected.ServiceURL, auditlogConfig.ServiceURL)
	assert.Equal(t, auditlogReferenceName, auditlogConfig.SecretReferenceName)
	assert.Equal(t, "service.auditlog.extensions.gardener.cloud/v1alpha1", auditlogConfig.APIVersion)
	assert.Equal(t, "AuditlogConfig", auditlogConfig.Kind)
}

func verifyOIDCExtension(t *testing.T, ext gardener.Extension) {
	require.NotNil(t, ext.Disabled)
	assert.Equal(t, false, *ext.Disabled)
}

func verifyDNSExtension(t *testing.T, ext gardener.Extension) {
	require.NotNil(t, ext.ProviderConfig)
	require.NotNil(t, ext.ProviderConfig.Raw)

	var dnsConfig DNSExtensionProviderConfig

	err := json.Unmarshal(ext.ProviderConfig.Raw, &dnsConfig)
	require.NoError(t, err)
	require.NotNil(t, dnsConfig.DNSProviderReplication)
	require.NotNil(t, dnsConfig.SyncProvidersFromShootSpecDNS)

	assert.Equal(t, "service.dns.extensions.gardener.cloud/v1alpha1", dnsConfig.APIVersion)
	assert.Equal(t, true, dnsConfig.DNSProviderReplication.Enabled)
	assert.Equal(t, true, *dnsConfig.SyncProvidersFromShootSpecDNS)
	assert.Equal(t, "DNSConfig", dnsConfig.Kind)

	require.Len(t, dnsConfig.Providers, 1)
	provider := dnsConfig.Providers[0]

	require.NotNil(t, provider.Domains)
	require.NotNil(t, provider.SecretName)
	require.NotNil(t, provider.Type)

	assert.Equal(t, "test-dns-secret", *provider.SecretName)
	assert.Equal(t, "test-provider", *provider.Type)

	require.Len(t, provider.Domains.Include, 1)
	assert.Equal(t, "test-shoot-name.test-domain", provider.Domains.Include[0])
}

func verifyCertExtension(t *testing.T, ext gardener.Extension) {
	require.NotNil(t, ext.ProviderConfig)
	require.NotNil(t, ext.ProviderConfig.Raw)

	var certConfig ExtensionProviderConfig

	err := json.Unmarshal(ext.ProviderConfig.Raw, &certConfig)
	require.NoError(t, err)
	require.NotNil(t, certConfig.ShootIssuers)
	assert.Equal(t, "service.cert.extensions.gardener.cloud/v1alpha1", certConfig.APIVersion)
	assert.Equal(t, true, certConfig.ShootIssuers.Enabled)
	assert.Equal(t, "CertConfig", certConfig.Kind)
}

func verifyNetworkFilterExtension(t *testing.T, ext gardener.Extension, isEnabled bool) {
	require.NotNil(t, ext.Disabled)
	assert.Equal(t, !isEnabled, *ext.Disabled)
}

func verifyRegistryCacheExtension(t *testing.T, ext *gardener.Extension, caches []imv1.ImageRegistryCache) {
	if len(caches) == 0 {
		assert.True(t, ext != nil && (ext.ProviderConfig != nil && *ext.Disabled))

		return
	}

	require.NotNil(t, ext.Disabled)
	require.Equal(t, false, *ext.Disabled)

	var registryConfig registrycacheext.RegistryConfig

	err := yaml.Unmarshal(ext.ProviderConfig.Raw, &registryConfig)
	require.NoError(t, err)

	assert.Equal(t, "registry.extensions.gardener.cloud/v1alpha3", registryConfig.APIVersion)
	assert.Equal(t, "RegistryConfig", registryConfig.Kind)
	assert.Equal(t, caches[0].Config.Upstream, registryConfig.Caches[0].Upstream)
	assert.Nil(t, caches[0].Config.GarbageCollection)

	if caches[0].Config.SecretReferenceName != nil {
		assert.Equal(t, ptr.To(fmt.Sprintf(RegistryCacheSecretNameFmt, caches[0].UID)), registryConfig.Caches[0].SecretReferenceName)
	} else {
		assert.Nil(t, registryConfig.Caches[0].SecretReferenceName)
	}

	assert.Nil(t, registryConfig.Caches[0].Proxy)
}

func verifyACLExtension(t *testing.T, ext *gardener.Extension, acl []string) {
	if len(acl) == 0 {
		assert.True(t, ext != nil && (ext.ProviderConfig == nil && *ext.Disabled))

		return
	}

	require.NotNil(t, ext.Disabled)
	require.Equal(t, false, *ext.Disabled)

	var aclConfig aclProviderConfig
	err := json.Unmarshal(ext.ProviderConfig.Raw, &aclConfig)
	require.NoError(t, err)

	assert.Equal(t, "ALLOW", aclConfig.Rule.Action)
	assert.Equal(t, "remote_ip", aclConfig.Rule.Type)
	assert.Equal(t, acl, aclConfig.Rule.Cidrs)
}

func buildFakeClientWithACLConfigMap(t *testing.T, configMapGetCalled *bool) client.Client {
	ipData, err := os.ReadFile("testdata/config-map-ips.yaml")
	require.NoError(t, err)
	var cm corev1.ConfigMap
	err = yaml.Unmarshal(ipData, &cm)
	require.NoError(t, err)

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	return fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(&cm).
		WithInterceptorFuncs(interceptor.Funcs{
			Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				if _, ok := obj.(*corev1.ConfigMap); ok {
					*configMapGetCalled = true
				}
				return c.Get(ctx, key, obj, opts...)
			},
		}).Build()
}

func fixRuntimeCRForExtensionExtenderTests(networkFilterEnabled bool, registryCache []imv1.ImageRegistryCache, apiServerACL []string, providerType string) imv1.Runtime {
	runtime := imv1.Runtime{
		Spec: imv1.RuntimeSpec{
			Shoot: imv1.RuntimeShoot{
				Name: "myshoot",
				Provider: imv1.Provider{
					Type: providerType,
				},
				Kubernetes: imv1.Kubernetes{
					KubeAPIServer: imv1.APIServer{
						ACL: &imv1.ACL{
							AllowedCIDRs: apiServerACL,
						},
					},
				},
			},
			Caching: registryCache,
			Security: imv1.Security{
				Networking: imv1.NetworkingSecurity{
					Filter: imv1.Filter{
						Egress: imv1.Egress{
							Enabled: networkFilterEnabled,
						},
					},
				},
			},
		},
	}

	return runtime
}
