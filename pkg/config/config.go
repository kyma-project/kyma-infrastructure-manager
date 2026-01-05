package config

import (
	"encoding/json"
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"io"
)

type Config struct {
	ConverterConfig ConverterConfig `json:"converter" validate:"required"`
	ClusterConfig   ClusterConfig   `json:"cluster" validate:"required"`
}

type ClusterConfig struct {
	DefaultSharedIASTenant OidcProvider `json:"defaultSharedIASTenant" validate:"required"`
}

type ProviderConfig struct {
	AWS AWSConfig `json:"aws"`
}

type AWSConfig struct {
	EnableIMDSv2 bool `json:"enableIMDSv2"`
}

type DNSConfig struct {
	SecretName   string `json:"secretName"`
	DomainPrefix string `json:"domainPrefix"`
	ProviderType string `json:"providerType"`
}

type KubernetesConfig struct {
	KubeApiServer                       KubeApiServer `json:"kubeApiServer"`
	DefaultVersion                      string        `json:"defaultVersion" validate:"required"`
	EnableKubernetesVersionAutoUpdate   bool          `json:"enableKubernetesVersionAutoUpdate"`
	EnableMachineImageVersionAutoUpdate bool          `json:"enableMachineImageVersionAutoUpdate"`
	DefaultOperatorOidc                 OidcProvider  `json:"defaultOperatorOidc" validate:"required"`
}

type OidcProvider struct {
	ClientID       string   `json:"clientID" validate:"required"`
	GroupsClaim    string   `json:"groupsClaim" validate:"required"`
	IssuerURL      string   `json:"issuerURL" validate:"required"`
	SigningAlgs    []string `json:"signingAlgs" validate:"required"`
	UsernameClaim  string   `json:"usernameClaim" validate:"required"`
	UsernamePrefix string   `json:"usernamePrefix" validate:"required"`
}

func (p OidcProvider) ToOIDCConfig() gardener.OIDCConfig {
	return gardener.OIDCConfig{
		ClientID:       &p.ClientID,
		GroupsClaim:    &p.GroupsClaim,
		IssuerURL:      &p.IssuerURL,
		SigningAlgs:    p.SigningAlgs,
		UsernameClaim:  &p.UsernameClaim,
		UsernamePrefix: &p.UsernamePrefix,
	}
}

type AuditLogConfig struct {
	PolicyConfigMapName string `json:"policyConfigMapName" validate:"required"`
	TenantConfigPath    string `json:"tenantConfigPath" validate:"required"`
}

type MaintenanceWindowConfig struct {
	WindowMapPath string `json:"windowMapPath"`
}

type GardenerConfig struct {
	ProjectName             string `json:"projectName" validate:"required"`
	EnableCredentialBinding bool   `json:"enableCredentialBinding"`
}

type MachineImageConfig struct {
	DefaultName    string `json:"defaultName" validate:"required"`
	DefaultVersion string `json:"defaultVersion" validate:"required"`
}

type KubeApiServer struct {
	MaxTokenExpiration string          `json:"maxTokenExpiration"`
	RuntimeConfig      map[string]bool `json:"runtimeConfig"`
	FeatureGates       map[string]bool `json:"featureGates"`
}

type TolerationsConfig map[string][]gardener.Toleration

type Networking struct {
	EnableDualStackIP bool `json:"enableDualStackIP"`
}

type ConverterConfig struct {
	Kubernetes        KubernetesConfig        `json:"kubernetes" validate:"required"`
	DNS               DNSConfig               `json:"dns"`
	Provider          ProviderConfig          `json:"provider"`
	MachineImage      MachineImageConfig      `json:"machineImage" validate:"required"`
	Gardener          GardenerConfig          `json:"gardener" validate:"required"`
	AuditLog          AuditLogConfig          `json:"auditLogging" validate:"required"`
	MaintenanceWindow MaintenanceWindowConfig `json:"maintenanceWindow"`
	Tolerations       TolerationsConfig       `json:"tolerations"`
	Networking        Networking              `json:"networking"`
}

// special case for own Gardener's DNS solution
func (c DNSConfig) IsGardenerInternal() bool {
	return c.ProviderType == "" && c.SecretName == "" && c.DomainPrefix == ""
}

type ReaderGetter = func() (io.Reader, error)

func (c *Config) Load(f ReaderGetter) error {
	r, err := f()
	if err != nil {
		return err
	}
	return json.NewDecoder(r).Decode(c)
}
