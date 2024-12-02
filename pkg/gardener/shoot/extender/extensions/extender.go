package extensions

import (
	"slices"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/config"
)

type CreateExtension func(shoot *gardener.Shoot) (gardener.Extension, error)

type Extension struct {
	Type    string
	Factory CreateExtension
}

func NewExtensionsExtenderForCreate(config config.ConverterConfig, auditLogData AuditLogData) func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
	return newExtensionsExtender([]Extension{
		{
			Type: CertExtensionType,
			Factory: func(_ *gardener.Shoot) (gardener.Extension, error) {
				return NewCertExtension()
			},
		},
		{
			Type: DNSExtensionType,
			Factory: func(shoot *gardener.Shoot) (gardener.Extension, error) {
				return NewDNSExtension(shoot.Name, config.DNS.SecretName, config.DNS.DomainPrefix, config.DNS.ProviderType)
			},
		},
		{
			Type: OidcExtensionType,
			Factory: func(_ *gardener.Shoot) (gardener.Extension, error) {
				return NewOIDCExtension()
			},
		},
		{
			Type: auditlogExtensionType,
			Factory: func(_ *gardener.Shoot) (gardener.Extension, error) {
				return NewAuditLogExtension(auditLogData)
			},
		},
	}, nil)
}

func NewExtensionsExtenderForPatch(auditLogData AuditLogData, extensionsOnTheShoot []gardener.Extension) func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
	return newExtensionsExtender([]Extension{
		{
			Type: OidcExtensionType,
			Factory: func(_ *gardener.Shoot) (gardener.Extension, error) {
				return NewOIDCExtension()
			},
		},
		{
			Type: auditlogExtensionType,
			Factory: func(_ *gardener.Shoot) (gardener.Extension, error) {
				return NewAuditLogExtension(auditLogData)
			},
		},
	}, extensionsOnTheShoot)
}

func newExtensionsExtender(extensionsToApply []Extension, currentGardenerExtensions []gardener.Extension) func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
	return func(_ imv1.Runtime, shoot *gardener.Shoot) error {
		for _, ext := range extensionsToApply {
			gardenerExtension, err := ext.Factory(shoot)
			if err != nil {
				return err
			}

			index := slices.IndexFunc(currentGardenerExtensions, func(e gardener.Extension) bool {
				return e.Type == ext.Type
			})

			if index == -1 {
				shoot.Spec.Extensions = append(shoot.Spec.Extensions, gardenerExtension)
			} else {
				shoot.Spec.Extensions[index] = gardenerExtension
			}
		}

		return nil
	}
}
