package extensions

import (
	"encoding/json"
	registrycache "github.com/kyma-project/kim-snatch/api/v1beta1"
	"k8s.io/utils/ptr"
	"slices"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/config"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/auditlogs"
)

type CreateExtensionFunc func(runtime imv1.Runtime, shoot gardener.Shoot) (*gardener.Extension, error)

type Extension struct {
	Type   string
	Create CreateExtensionFunc
}

func NewExtensionsExtenderForCreate(config config.ConverterConfig, auditLogData auditlogs.AuditLogData, registryCache []registrycache.RegistryCache) func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
	return newExtensionsExtender([]Extension{
		{
			Type: NetworkFilterType,
			Create: func(runtime imv1.Runtime, _ gardener.Shoot) (*gardener.Extension, error) {
				return NewNetworkFilterExtension(runtime.Spec.Security.Networking.Filter)
			},
		},
		{
			Type: CertExtensionType,
			Create: func(_ imv1.Runtime, _ gardener.Shoot) (*gardener.Extension, error) {
				return NewCertExtension()
			},
		},
		{
			Type: DNSExtensionType,
			Create: func(_ imv1.Runtime, shoot gardener.Shoot) (*gardener.Extension, error) {
				if config.DNS.IsGardenerInternal() {
					return NewDNSExtensionInternal()
				}
				return NewDNSExtensionExternal(shoot.Name, config.DNS.SecretName, config.DNS.DomainPrefix, config.DNS.ProviderType)
			},
		},
		{
			Type: OidcExtensionType,
			Create: func(_ imv1.Runtime, _ gardener.Shoot) (*gardener.Extension, error) {
				return NewOIDCExtension()
			},
		},
		{
			Type: AuditlogExtensionType,
			Create: func(_ imv1.Runtime, _ gardener.Shoot) (*gardener.Extension, error) {
				if auditLogData == (auditlogs.AuditLogData{}) {
					return nil, nil
				}

				return NewAuditLogExtension(auditLogData)
			},
		},
		{
			Type: RegistryCacheExtensionType,
			Create: func(runtime imv1.Runtime, shoot gardener.Shoot) (*gardener.Extension, error) {
				if len(registryCache) == 0 {
					return nil, nil
				}

				return NewRegistryCacheExtension(registryCache, true)
			},
		},
	}, nil)
}

func NewExtensionsExtenderForPatch(auditLogData auditlogs.AuditLogData, registryCache []registrycache.RegistryCache, extensionsOnTheShoot []gardener.Extension) func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
	return newExtensionsExtender([]Extension{
		{
			AuditlogExtensionType,
			func(_ imv1.Runtime, shoot gardener.Shoot) (*gardener.Extension, error) {
				if auditLogData == (auditlogs.AuditLogData{}) {
					return nil, nil
				}

				newAuditLogExtension, err := NewAuditLogExtension(auditLogData)
				if err != nil {
					return nil, err
				}

				auditLogIndex := slices.IndexFunc(shoot.Spec.Extensions, func(e gardener.Extension) bool {
					return e.Type == AuditlogExtensionType
				})

				if auditLogIndex == -1 {
					return newAuditLogExtension, nil
				}
				var existingAuditLogConfig AuditlogExtensionConfig
				if err := json.Unmarshal(shoot.Spec.Extensions[auditLogIndex].ProviderConfig.Raw, &existingAuditLogConfig); err != nil {
					return nil, err
				}

				var newAuditLogConfig AuditlogExtensionConfig
				if err := json.Unmarshal(newAuditLogExtension.ProviderConfig.Raw, &newAuditLogConfig); err != nil {
					return nil, err
				}

				if newAuditLogConfig != existingAuditLogConfig {
					return newAuditLogExtension, nil
				}

				return nil, nil
			},
		},
		{
			Type: NetworkFilterType,
			Create: func(runtime imv1.Runtime, _ gardener.Shoot) (*gardener.Extension, error) {
				return NewNetworkFilterExtension(runtime.Spec.Security.Networking.Filter)
			},
		},
		{
			Type: RegistryCacheExtensionType,
			Create: func(runtime imv1.Runtime, shoot gardener.Shoot) (*gardener.Extension, error) {

				if runtime.Spec.Caching != nil && runtime.Spec.Caching.Enabled && len(registryCache) > 0 {
					return NewRegistryCacheExtension(registryCache, true)
				}

				for _, ext := range shoot.Spec.Extensions {
					if ext.Type == RegistryCacheExtensionType {
						ext.Disabled = ptr.To(true)
						return &ext, nil
					}
				}

				return nil, nil
			},
		},
	}, extensionsOnTheShoot)
}

func newExtensionsExtender(extensionsToApply []Extension, currentGardenerExtensions []gardener.Extension) func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
	return func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
		shoot.Spec.Extensions = currentGardenerExtensions

		for _, ext := range extensionsToApply {
			gardenerExtension, err := ext.Create(runtime, *shoot)
			if err != nil {
				return err
			}

			// If the extension should not be applied we skip it
			if gardenerExtension == nil {
				continue
			}

			index := slices.IndexFunc(currentGardenerExtensions, func(e gardener.Extension) bool {
				return e.Type == ext.Type
			})

			if index == -1 {
				shoot.Spec.Extensions = append(shoot.Spec.Extensions, *gardenerExtension)
			} else {
				shoot.Spec.Extensions[index] = *gardenerExtension
			}
		}

		return nil
	}
}
