package extender

import (
	"encoding/json"
	"fmt"
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	v12 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"os"
)

const (
	auditLogConditionType   = "AuditlogServiceAvailability"
	auditlogSecretReference = "auditlog-credentials"
	auditlogExtensionType   = "shoot-auditlog-service"
)

type AuditLogConfig struct {
	TenantID   string `json:"tenantID"`
	ServiceURL string `json:"serviceURL"`
	SecretName string `json:"secretName"`
}

// AuditlogConfig configuration resource
type AuditlogExtensionConfig struct {
	metav1.TypeMeta `json:",inline"`

	// Type is the type of auditlog service provider.
	Type string `json:"type"`
	// TenantID is the id of the tenant.
	TenantID string `json:"tenantID"`
	// ServiceURL is the URL of the auditlog service.
	ServiceURL string `json:"serviceURL"`
	// SecretReferenceName is the name of the reference for the secret containing the auditlog service credentials.
	SecretReferenceName string `json:"secretReferenceName"`
}

func NewAuditLogExtender(policyConfigMapName, tenantConfigPath string) func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
	return func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
		if canEnableAuditLogsForShoot(getSeedName(*shoot), tenantConfigPath) {
			providerType := runtime.Spec.Shoot.Provider.Type

			configureAuditPolicy(shoot, policyConfigMapName)
			annotated, err := configureAuditLogs(providerType, tenantConfigPath, shoot)
			if err != nil {
				//logger.Warnf("Cannot enable audit logs: %s", err.Error())
				return nil
			}
			if !annotated {
				//logger.Debug("Audit Log Tenant did not change, skipping update of cluster")
				return nil
			}

			//logger.Debug("Modifying Audit Log config")
		}

		// reque dopoki nie wlaczymy audit logow na shoocie?

		return nil
	}
}

func configureAuditPolicy(shoot *gardener.Shoot, policyConfigMapName string) {
	if shoot.Spec.Kubernetes.KubeAPIServer == nil {
		shoot.Spec.Kubernetes.KubeAPIServer = &gardener.KubeAPIServerConfig{}
	}

	shoot.Spec.Kubernetes.KubeAPIServer.AuditConfig = newAuditPolicyConfig(policyConfigMapName)
}

func newAuditPolicyConfig(policyConfigMapName string) *gardener.AuditConfig {
	return &gardener.AuditConfig{
		AuditPolicy: &gardener.AuditPolicy{
			ConfigMapRef: &v12.ObjectReference{Name: policyConfigMapName},
		},
	}
}

func configureAuditLogs(providerType, tenantConfigPath string, shoot *gardener.Shoot) (bool, error) {
	auditLogConfig, err := getConfigFromFile(tenantConfigPath)
	if err != nil {
		return false, err
	}

	providerConfig := auditLogConfig[providerType]
	if providerConfig == nil {
		return false, fmt.Errorf("cannot find config for provider %s", providerType)
	}

	// ctxLogger := logger.WithField("provider", providerType)
	auditID := shoot.Spec.Region
	if auditID == "" {
		return false, fmt.Errorf("shoot has no region set")
	}

	// ctxLogger = ctxLogger.WithField("auditID", auditID)

	tenant, ok := providerConfig[auditID]
	if !ok {
		return false, fmt.Errorf("auditlog config for region %s, provider %s is empty", auditID, providerType)
	}

	changedExt, err := configureExtension(shoot, tenant)
	changedSec := configureSecret(shoot, tenant)

	return changedExt || changedSec, err
}

func configureExtension(shoot *gardener.Shoot, config AuditLogConfig) (changed bool, err error) {
	changed = false
	const (
		extensionKind    = "AuditlogConfig"
		extensionVersion = "service.auditlog.extensions.gardener.cloud/v1alpha1"
		extensionType    = "standard"
	)

	ext := findExtension(shoot)
	if ext != nil {
		cfg := AuditlogExtensionConfig{}
		err = json.Unmarshal(ext.ProviderConfig.Raw, &cfg)
		if err != nil {
			return
		}

		if cfg.Kind == extensionKind &&
			cfg.Type == extensionType &&
			cfg.TenantID == config.TenantID &&
			cfg.ServiceURL == config.ServiceURL &&
			cfg.SecretReferenceName == auditlogSecretReference {
			return false, nil
		}
	} else {
		shoot.Spec.Extensions = append(shoot.Spec.Extensions, gardener.Extension{
			Type: auditlogExtensionType,
		})
		ext = &shoot.Spec.Extensions[len(shoot.Spec.Extensions)-1]
	}

	changed = true

	cfg := AuditlogExtensionConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       extensionKind,
			APIVersion: extensionVersion,
		},
		Type:                extensionType,
		TenantID:            config.TenantID,
		ServiceURL:          config.ServiceURL,
		SecretReferenceName: auditlogSecretReference,
	}

	ext.ProviderConfig = &runtime.RawExtension{}
	ext.ProviderConfig.Raw, err = json.Marshal(cfg)

	return
}

func configureSecret(shoot *gardener.Shoot, config AuditLogConfig) (changed bool) {
	changed = false

	sec := findSecret(shoot)
	if sec != nil {
		if sec.Name == auditlogSecretReference &&
			sec.ResourceRef.APIVersion == "v1" &&
			sec.ResourceRef.Kind == "Secret" &&
			sec.ResourceRef.Name == config.SecretName {
			return false
		}
	} else {
		shoot.Spec.Resources = append(shoot.Spec.Resources, gardener.NamedResourceReference{})
		sec = &shoot.Spec.Resources[len(shoot.Spec.Resources)-1]
	}

	changed = true

	sec.Name = auditlogSecretReference
	sec.ResourceRef.APIVersion = "v1"
	sec.ResourceRef.Kind = "Secret"
	sec.ResourceRef.Name = config.SecretName

	return
}

func canEnableAuditLogsForShoot(seedName, tenantConfigPath string) bool {
	return seedName != "" && tenantConfigPath != ""
}

func getSeedName(shoot gardener.Shoot) string {
	if shoot.Spec.SeedName != nil {
		return *shoot.Spec.SeedName
	}
	return ""
}

func getConfigFromFile(tenantConfigPath string) (data map[string]map[string]AuditLogConfig, err error) {
	file, err := os.Open(tenantConfigPath)

	if err != nil {
		return nil, err
	}

	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	if err := json.NewDecoder(file).Decode(&data); err != nil {
		return nil, err
	}
	return data, nil
}

func findExtension(shoot *gardener.Shoot) *gardener.Extension {
	for i, e := range shoot.Spec.Extensions {
		if e.Type == auditlogExtensionType {
			return &shoot.Spec.Extensions[i]
		}
	}

	return nil
}

func findSecret(shoot *gardener.Shoot) *gardener.NamedResourceReference {
	for i, e := range shoot.Spec.Resources {
		if e.Name == auditlogSecretReference {
			return &shoot.Spec.Resources[i]
		}
	}

	return nil
}
