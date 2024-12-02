package extensions

import (
	"encoding/json"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	apimachineryRuntime "k8s.io/apimachinery/pkg/runtime"
)

const CertExtensionType = "shoot-cert-service"

type ExtensionProviderConfig struct {
	// APIVersion is gardener extension api version
	APIVersion string `json:"apiVersion"`
	// DnsProviderReplication indicates whether dnsProvider replication is on
	DNSProviderReplication *DNSProviderReplication `json:"dnsProviderReplication,omitempty"`
	// ShootIssuers indicates whether shoot Issuers are on
	ShootIssuers *ShootIssuers `json:"shootIssuers,omitempty"`
	// Kind is extension type
	Kind string `json:"kind"`
}

type ShootIssuers struct {
	// Enabled indicates whether shoot Issuers are on
	Enabled bool `json:"enabled"`
}

func NewCertConfig() *ExtensionProviderConfig {
	return &ExtensionProviderConfig{
		APIVersion:   "service.cert.extensions.gardener.cloud/v1alpha1",
		ShootIssuers: &ShootIssuers{Enabled: true},
		Kind:         "CertConfig",
	}
}

func NewCertExtension() (gardener.Extension, error) {
	certConfig := NewCertConfig()
	jsonCertConfig, encodingErr := json.Marshal(certConfig)
	if encodingErr != nil {
		return gardener.Extension{}, encodingErr
	}

	certServiceExtension := gardener.Extension{
		Type:           CertExtensionType,
		ProviderConfig: &apimachineryRuntime.RawExtension{Raw: jsonCertConfig},
	}

	return certServiceExtension, nil
}
