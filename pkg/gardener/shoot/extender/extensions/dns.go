package extensions

import (
	"encoding/json"
	"fmt"
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	apimachineryruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
)

const DNSExtensionType = "shoot-dns-service"

// The types were copied from the following file: https://github.com/gardener/gardener-extension-shoot-dns-service/blob/master/pkg/apis/service/types.go
type DNSExtensionProviderConfig struct {
	// APIVersion is gardener extension api version
	APIVersion string `json:"apiVersion"`
	// Kind is extension type
	Kind string `json:"kind"`

	// DnsProviderReplication indicates whether dnsProvider replication is on
	DNSProviderReplication *DNSProviderReplication `json:"dnsProviderReplication,omitempty"`
	// Providers is a list of additional DNS providers that shall be enabled for this shoot cluster.
	// The primary ("external") provider at `spec.dns.provider` is added automatically
	Providers []DNSProvider `json:"providers"`
	// SyncProvidersFromShootSpecDNS is an optional flag for migrating and synchronising the providers given in the
	// shoot manifest at section `spec.dns.providers`. If true, any direct changes on the `providers` section
	// are overwritten with the content of section `spec.dns.providers`.
	SyncProvidersFromShootSpecDNS *bool `json:"syncProvidersFromShootSpecDNS,omitempty"`
}

// DNSProvider contains information about a DNS provider.
type DNSProvider struct {
	// Domains contains information about which domains shall be included/excluded for this provider.
	Domains *DNSIncludeExclude `json:"domains,omitempty"`
	// SecretName is a name of a secret containing credentials for the stated domain and the
	// provider.
	SecretName *string `json:"secretName,omitempty"`
	// Type is the DNS provider type.
	Type *string `json:"type,omitempty"`
	// Zones contains information about which hosted zones shall be included/excluded for this provider.
	Zones *DNSIncludeExclude `json:"zones,omitempty"`
}

// DNSIncludeExclude contains information about which domains shall be included/excluded.
type DNSIncludeExclude struct {
	// Include is a list of domains that shall be included.
	Include []string `json:"include,omitempty"`
	// Exclude is a list of domains that shall be excluded.
	Exclude []string `json:"exclude,omitempty"`
}

type DNSProviderReplication struct {
	// Enabled indicates whether replication is on
	Enabled bool `json:"enabled"`
}

func newDNSExtensionConfig(domain, secretName, dnsProviderType string) *DNSExtensionProviderConfig {
	return &DNSExtensionProviderConfig{
		APIVersion:                    "service.dns.extensions.gardener.cloud/v1alpha1",
		Kind:                          "DNSConfig",
		DNSProviderReplication:        &DNSProviderReplication{Enabled: true},
		SyncProvidersFromShootSpecDNS: ptr.To(true),
		Providers: []DNSProvider{
			{
				Domains: &DNSIncludeExclude{
					Include: []string{domain},
				},
				SecretName: ptr.To(secretName),
				Type:       ptr.To(dnsProviderType),
			},
		},
	}
}

func NewDNSExtension(shootName, secretName, domainSuffix, dnsProviderType string) (*gardener.Extension, error) {
	var providerConfig *DNSExtensionProviderConfig

	if secretName != "" && dnsProviderType != "" {
		domain := fmt.Sprintf("%s.%s", shootName, domainSuffix)
		providerConfig = newDNSExtensionConfig(domain, secretName, dnsProviderType)
	} else {
		// special case for Gardener's DNS solution
		providerConfig = &DNSExtensionProviderConfig{
			APIVersion:                    "service.dns.extensions.gardener.cloud/v1alpha1",
			Kind:                          "DNSConfig",
			SyncProvidersFromShootSpecDNS: ptr.To(true),
		}
	}

	extensionJSON, err := json.Marshal(providerConfig)
	if err != nil {
		return nil, err
	}

	return &gardener.Extension{
		Type: DNSExtensionType,
		ProviderConfig: &apimachineryruntime.RawExtension{
			Raw: extensionJSON,
		},
	}, nil
}
