package extensions

import (
	"encoding/json"
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestDNSExtensionsExtender(t *testing.T) {
	for _, testcase := range []struct {
		name         string
		shootName    string
		secretName   string
		prefix       string
		providerType string
		expectLocal  bool
	}{
		{
			name:         "Should generate DNS extension for provided external DNS configuration",
			shootName:    "myshoot",
			secretName:   "aws-route53-secret-dev",
			prefix:       "dev.kyma.ondemand.com",
			providerType: "aws-route53",
			expectLocal:  false,
		},
		{
			name:        "Should generate DNS extension for internal Gardener DNS configuration",
			shootName:   "myshoot",
			expectLocal: true,
		},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			if testcase.expectLocal {
				ext, err := NewDNSExtensionInternal()
				require.NoError(t, err)
				verifyLocalDNSExtension(t, ext)
			} else {
				ext, err := NewDNSExtensionExternal(testcase.shootName, testcase.secretName, testcase.prefix, testcase.providerType)
				require.NoError(t, err)
				verifyExternalDNSExtension(t, ext)
			}
		})
	}
}

func verifyExternalDNSExtension(t *testing.T, ext *gardener.Extension) {
	require.NotNil(t, ext)
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

	assert.Equal(t, "aws-route53-secret-dev", *provider.SecretName)
	assert.Equal(t, "aws-route53", *provider.Type)

	require.Len(t, provider.Domains.Include, 1)
	assert.Equal(t, "myshoot.dev.kyma.ondemand.com", provider.Domains.Include[0])
}

func verifyLocalDNSExtension(t *testing.T, ext *gardener.Extension) {
	require.NotNil(t, ext)
	require.NotNil(t, ext.ProviderConfig)
	require.NotNil(t, ext.ProviderConfig.Raw)

	var dnsConfig DNSExtensionProviderConfig

	err := json.Unmarshal(ext.ProviderConfig.Raw, &dnsConfig)
	require.NoError(t, err)
	require.Nil(t, dnsConfig.DNSProviderReplication)
	require.NotNil(t, dnsConfig.SyncProvidersFromShootSpecDNS)

	assert.Equal(t, "service.dns.extensions.gardener.cloud/v1alpha1", dnsConfig.APIVersion)
	assert.Equal(t, true, *dnsConfig.SyncProvidersFromShootSpecDNS)
	assert.Equal(t, "DNSConfig", dnsConfig.Kind)

	require.Len(t, dnsConfig.Providers, 0)
}
