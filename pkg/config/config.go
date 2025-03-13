package config

import (
	"encoding/json"
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
	SecretName   string `json:"secretName" validate:"required"`
	DomainPrefix string `json:"domainPrefix" validate:"required"`
	ProviderType string `json:"providerType" validate:"required"`
}

type KubernetesConfig struct {
	DefaultVersion                      string       `json:"defaultVersion" validate:"required"`
	EnableKubernetesVersionAutoUpdate   bool         `json:"enableKubernetesVersionAutoUpdate"`
	EnableMachineImageVersionAutoUpdate bool         `json:"enableMachineImageVersionVersionAutoUpdate"`
	DefaultOperatorOidc                 OidcProvider `json:"defaultOperatorOidc" validate:"required"`
}

type OidcProvider struct {
	ClientID       string   `json:"clientID" validate:"required"`
	GroupsClaim    string   `json:"groupsClaim" validate:"required"`
	IssuerURL      string   `json:"issuerURL" validate:"required"`
	SigningAlgs    []string `json:"signingAlgs" validate:"required"`
	UsernameClaim  string   `json:"usernameClaim" validate:"required"`
	UsernamePrefix string   `json:"usernamePrefix" validate:"required"`
}

type AuditLogConfig struct {
	PolicyConfigMapName string `json:"policyConfigMapName" validate:"required"`
	TenantConfigPath    string `json:"tenantConfigPath" validate:"required"`
}

type MaintenanceWindowConfig struct {
	WindowMapPath string `json:"windowMapPath"`
}

type GardenerConfig struct {
	ProjectName string `json:"projectName" validate:"required"`
}

type MachineImageConfig struct {
	DefaultName    string `json:"defaultName" validate:"required"`
	DefaultVersion string `json:"defaultVersion" validate:"required"`
}

type ConverterConfig struct {
	Kubernetes        KubernetesConfig        `json:"kubernetes" validate:"required"`
	DNS               DNSConfig               `json:"dns" validate:"required"`
	Provider          ProviderConfig          `json:"provider"`
	MachineImage      MachineImageConfig      `json:"machineImage" validate:"required"`
	Gardener          GardenerConfig          `json:"gardener" validate:"required"`
	AuditLog          AuditLogConfig          `json:"auditLogging" validate:"required"`
	MaintenanceWindow MaintenanceWindowConfig `json:"maintenanceWindow"`
}

type ReaderGetter = func() (io.Reader, error)

func (c *Config) Load(f ReaderGetter) error {
	r, err := f()
	if err != nil {
		return err
	}
	return json.NewDecoder(r).Decode(c)
}
