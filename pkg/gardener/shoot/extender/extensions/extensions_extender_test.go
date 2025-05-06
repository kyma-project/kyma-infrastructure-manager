package extensions

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"

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
	}
	newAuditLogData := auditlogs.AuditLogData{
		TenantID:   "test-auditlog-tenant",
		ServiceURL: "test-auditlog-service-url",
		SecretName: "doesnt matter",
	}

	for _, testcase := range []struct {
		name                 string
		inputAuditLogData    auditlogs.AuditLogData
		disableNetworkFilter bool
		extensionOrderMap    map[string]int
	}{
		{
			name:                 "Should create all extensions for new Shoot in the right order, network filter is enabled",
			inputAuditLogData:    newAuditLogData,
			disableNetworkFilter: false,
			extensionOrderMap:    getExpectedExtensionsOrderMapForCreate(),
		},
		{
			name:                 "Should create all extensions for new Shoot in the right order, network filter is disabled",
			inputAuditLogData:    newAuditLogData,
			disableNetworkFilter: true,
			extensionOrderMap:    getExpectedExtensionsOrderMapForCreate(),
		},
		{
			name:              "Should not include AuditLog extension for new Shoot when input auditLogData is empty",
			inputAuditLogData: auditlogs.AuditLogData{},
			extensionOrderMap: getExpectedExtensionsOrderMapForCreateWithoutAuditLogs(),
		},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			runtime := fixRuntimeCRForExtensionExtenderTests(!testcase.disableNetworkFilter)

			shoot := &gardener.Shoot{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-shoot-name",
				},
			}

			extender := NewExtensionsExtenderForCreate(config, testcase.inputAuditLogData)

			err := extender(runtime, shoot)
			assert.NoError(t, err)
			assert.NotNil(t, shoot.Spec.Extensions)

			orderMap := testcase.extensionOrderMap
			require.Len(t, shoot.Spec.Extensions, len(orderMap))

			for idx, ext := range shoot.Spec.Extensions {
				assert.NotEmpty(t, ext.Type)
				assert.Equal(t, orderMap[ext.Type], idx)

				switch ext.Type {
				case NetworkFilterType:
					verifyNetworkFilterExtension(t, ext, testcase.disableNetworkFilter)

				case CertExtensionType:
					verifyCertExtension(t, ext)

				case DNSExtensionType:
					verifyDNSExtension(t, ext)

				case OidcExtensionType:
					verifyOIDCExtension(t, ext)
				}
			}
		})
	}
}

func TestNewExtensionsExtenderForPatch(t *testing.T) {
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

	for _, testCase := range []struct {
		name                 string
		inputAuditLogData    auditlogs.AuditLogData
		expectedAuditLogData auditlogs.AuditLogData
		disableNetworkFilter bool
		previousExtensions   []gardener.Extension
	}{
		{
			name:                 "Existing extensions should not change order during patching if nothing has changed",
			previousExtensions:   fixAllExtensionsOnTheShoot(),
			inputAuditLogData:    oldAuditLogData,
			expectedAuditLogData: oldAuditLogData,
			disableNetworkFilter: true,
		},
		{
			name:                 "Should update Audit Log extension without changing order and data of other extensions",
			previousExtensions:   fixAllExtensionsOnTheShoot(),
			inputAuditLogData:    newAuditLogData,
			expectedAuditLogData: newAuditLogData,
			disableNetworkFilter: true,
		},
		{
			name:                 "Should update Network filter extension without changing order and data of other extensions",
			previousExtensions:   fixAllExtensionsOnTheShoot(),
			inputAuditLogData:    oldAuditLogData,
			expectedAuditLogData: oldAuditLogData,
			disableNetworkFilter: false,
		},
		{
			name:                 "Should add Network filter extension at the end without changing order and data of other extensions",
			previousExtensions:   fixExtensionsOnTheShootWithoutNetworkFilter(),
			inputAuditLogData:    oldAuditLogData,
			expectedAuditLogData: oldAuditLogData,
			disableNetworkFilter: true,
		},
		{
			name:                 "Should add AuditLog extension at the end without changing order and data of other extensions",
			previousExtensions:   fixExtensionsOnTheShootWithoutAuditLogs(),
			inputAuditLogData:    oldAuditLogData,
			expectedAuditLogData: oldAuditLogData,
			disableNetworkFilter: true,
		},
		{
			name:                 "Should add AuditLog and Network filter extensions at the end without changing order and data of other extensions",
			previousExtensions:   fixExtensionsOnTheShootWithoutAuditLogsAndNetworkFilter(),
			inputAuditLogData:    oldAuditLogData,
			expectedAuditLogData: oldAuditLogData,
			disableNetworkFilter: true,
		},
		{
			name:                 "Should not update existing AuditLog extension when input auditLogData is empty",
			previousExtensions:   fixAllExtensionsOnTheShoot(),
			inputAuditLogData:    auditlogs.AuditLogData{},
			expectedAuditLogData: oldAuditLogData,
			disableNetworkFilter: true,
		},
		{
			name:                 "Should not add AuditLog extension to existing shoot extensions when input auditLogData is empty",
			previousExtensions:   fixExtensionsOnTheShootWithoutAuditLogs(),
			inputAuditLogData:    auditlogs.AuditLogData{},
			disableNetworkFilter: false,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			runtime := fixRuntimeCRForExtensionExtenderTests(!testCase.disableNetworkFilter)

			shoot := &gardener.Shoot{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-shoot-name",
				},
			}

			isEmptyAuditLogData := testCase.inputAuditLogData == (auditlogs.AuditLogData{})

			extender := NewExtensionsExtenderForPatch(testCase.inputAuditLogData, testCase.previousExtensions)
			orderMap := getExpectedExtensionsOrderMapForPatch(testCase.previousExtensions, isEmptyAuditLogData)

			err := extender(runtime, shoot)
			assert.NoError(t, err)
			assert.NotNil(t, shoot.Spec.Extensions)
			require.Len(t, shoot.Spec.Extensions, len(orderMap))

			for idx, ext := range shoot.Spec.Extensions {
				assert.NotEmpty(t, ext.Type)
				assert.Equal(t, orderMap[ext.Type], idx)

				switch ext.Type {
				case NetworkFilterType:
					verifyNetworkFilterExtension(t, ext, testCase.disableNetworkFilter)

				case CertExtensionType:
					verifyCertExtension(t, ext)

				case DNSExtensionType:
					verifyDNSExtension(t, ext)

				case OidcExtensionType:
					verifyOIDCExtension(t, ext)

				case AuditlogExtensionType:
					verifyAuditLogExtension(t, ext, testCase.expectedAuditLogData)
				}
			}
		})
	}
}

func fixAllExtensionsOnTheShoot() []gardener.Extension {
	return []gardener.Extension{
		{
			Type: AuditlogExtensionType,
			ProviderConfig: &runtime.RawExtension{
				Raw: []byte(`{"apiVersion":"service.auditlog.extensions.gardener.cloud/v1alpha1","kind":"AuditlogConfig","type":"standard","tenantID":"test-auditlog-tenant","serviceURL":"test-auditlog-service-url","secretReferenceName":"auditlog-credentials"}`),
			},
		},
		{
			Type: DNSExtensionType,
			ProviderConfig: &runtime.RawExtension{
				Raw: []byte(`{"apiVersion":"service.dns.extensions.gardener.cloud/v1alpha1","dnsProviderReplication":{"enabled":true},"syncProvidersFromShootSpecDNS":true,"providers":[{"domains":{"include":["test-shoot-name.test-domain"],"exclude":null},"secretName":"test-dns-secret","type":"test-provider"}],"kind":"DNSConfig"}`),
			},
		},
		{
			Type: CertExtensionType,
			ProviderConfig: &runtime.RawExtension{
				Raw: []byte(`{"apiVersion":"service.cert.extensions.gardener.cloud/v1alpha1","kind":"CertConfig","shootIssuers":{"enabled":true}}`),
			},
		},
		{
			Type:     NetworkFilterType,
			Disabled: ptr.To(true),
		},
		{
			Type:     OidcExtensionType,
			Disabled: ptr.To(false),
		},
	}
}

func fixExtensionsOnTheShootWithoutAuditLogs() []gardener.Extension {
	return []gardener.Extension{
		{
			Type: DNSExtensionType,
			ProviderConfig: &runtime.RawExtension{
				Raw: []byte(`{"apiVersion":"service.dns.extensions.gardener.cloud/v1alpha1","dnsProviderReplication":{"enabled":true},"syncProvidersFromShootSpecDNS":true,"providers":[{"domains":{"include":["test-shoot-name.test-domain"],"exclude":null},"secretName":"test-dns-secret","type":"test-provider"}],"kind":"DNSConfig"}`),
			},
		},
		{
			Type: CertExtensionType,
			ProviderConfig: &runtime.RawExtension{
				Raw: []byte(`{"apiVersion":"service.cert.extensions.gardener.cloud/v1alpha1","kind":"CertConfig","shootIssuers":{"enabled":true}}`),
			},
		},
		{
			Type:     NetworkFilterType,
			Disabled: ptr.To(true),
		},
		{
			Type:     OidcExtensionType,
			Disabled: ptr.To(false),
		},
	}
}

func fixExtensionsOnTheShootWithoutNetworkFilter() []gardener.Extension {
	return []gardener.Extension{
		{
			Type: AuditlogExtensionType,
			ProviderConfig: &runtime.RawExtension{
				Raw: []byte(`{"apiVersion":"service.auditlog.extensions.gardener.cloud/v1alpha1","kind":"AuditlogConfig","type":"standard","tenantID":"test-auditlog-tenant","serviceURL":"test-auditlog-service-url","secretReferenceName":"auditlog-credentials"}`),
			},
		},
		{
			Type: DNSExtensionType,
			ProviderConfig: &runtime.RawExtension{
				Raw: []byte(`{"apiVersion":"service.dns.extensions.gardener.cloud/v1alpha1","dnsProviderReplication":{"enabled":true},"syncProvidersFromShootSpecDNS":true,"providers":[{"domains":{"include":["test-shoot-name.test-domain"],"exclude":null},"secretName":"test-dns-secret","type":"test-provider"}],"kind":"DNSConfig"}`),
			},
		},
		{
			Type: CertExtensionType,
			ProviderConfig: &runtime.RawExtension{
				Raw: []byte(`{"apiVersion":"service.cert.extensions.gardener.cloud/v1alpha1","kind":"CertConfig","shootIssuers":{"enabled":true}}`),
			},
		},
		{
			Type:     OidcExtensionType,
			Disabled: ptr.To(false),
		},
	}
}

func fixExtensionsOnTheShootWithoutAuditLogsAndNetworkFilter() []gardener.Extension {
	return []gardener.Extension{
		{
			Type: DNSExtensionType,
			ProviderConfig: &runtime.RawExtension{
				Raw: []byte(`{"apiVersion":"service.dns.extensions.gardener.cloud/v1alpha1","dnsProviderReplication":{"enabled":true},"syncProvidersFromShootSpecDNS":true,"providers":[{"domains":{"include":["test-shoot-name.test-domain"],"exclude":null},"secretName":"test-dns-secret","type":"test-provider"}],"kind":"DNSConfig"}`),
			},
		},
		{
			Type: CertExtensionType,
			ProviderConfig: &runtime.RawExtension{
				Raw: []byte(`{"apiVersion":"service.cert.extensions.gardener.cloud/v1alpha1","kind":"CertConfig","shootIssuers":{"enabled":true}}`),
			},
		},
		{
			Type:     OidcExtensionType,
			Disabled: ptr.To(false),
		},
	}
}

// returns a map with the expected index order of extensions for different ExtenderForPatch unit tests
func getExpectedExtensionsOrderMapForPatch(extensions []gardener.Extension, emptyAuditlogData bool) map[string]int {
	extensionOrderMap := make(map[string]int)

	for idx, ext := range extensions {
		extensionOrderMap[ext.Type] = idx
	}

	if len(extensions) == 4 {
		// add missing one at the end with an exception for Auditlog extension when input auditLogData is empty
		if _, ok := extensionOrderMap[AuditlogExtensionType]; !ok {
			if !emptyAuditlogData { // case when Auditlog extension is missing and should not be added
				extensionOrderMap[AuditlogExtensionType] = 4
			}
		}

		if _, ok := extensionOrderMap[NetworkFilterType]; !ok {
			extensionOrderMap[NetworkFilterType] = 4
		}
	}

	if len(extensions) == 3 {
		// add missing two at the end
		extensionOrderMap[AuditlogExtensionType] = 3
		extensionOrderMap[NetworkFilterType] = 4
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

	return extensionOrderMap
}

func getExpectedExtensionsOrderMapForCreateWithoutAuditLogs() map[string]int {
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

func verifyNetworkFilterExtension(t *testing.T, ext gardener.Extension, isDisabled bool) {
	require.NotNil(t, ext.Disabled)
	assert.Equal(t, isDisabled, *ext.Disabled)
}

func fixRuntimeCRForExtensionExtenderTests(networkFilterEnabled bool) imv1.Runtime {
	runtime := imv1.Runtime{
		Spec: imv1.RuntimeSpec{
			Shoot: imv1.RuntimeShoot{
				Name: "myshoot",
			},
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
