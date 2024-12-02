package extensions

import (
	"slices"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/config"
)

type CreateExtensionFunc func(runtime imv1.Runtime, shoot gardener.Shoot) (*gardener.Extension, error)

type Extension struct {
	Type   string
	Create CreateExtensionFunc
}

func NewExtensionsExtenderForCreate(config config.ConverterConfig, auditLogData AuditLogData) func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
	return newExtensionsExtender([]Extension{
		{
			Type: NetworkFilterType,
			Create: func(runtime imv1.Runtime, _ gardener.Shoot) (*gardener.Extension, error) {
				return NewNetworkFilterExtension(!runtime.Spec.Security.Networking.Filter.Egress.Enabled)
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
				return NewDNSExtension(shoot.Name, config.DNS.SecretName, config.DNS.DomainPrefix, config.DNS.ProviderType)
			},
		},
		{
			Type: OidcExtensionType,
			Create: func(_ imv1.Runtime, shoot gardener.Shoot) (*gardener.Extension, error) {
				return NewOIDCExtension()
			},
		},
		{
			Type: auditlogExtensionType,
			Create: func(runtime imv1.Runtime, shoot gardener.Shoot) (*gardener.Extension, error) {
				return NewAuditLogExtension(auditLogData)
			},
		},
	}, nil)
}

func NewExtensionsExtenderForPatch(auditLogData AuditLogData, extensionsOnTheShoot []gardener.Extension) func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
	return newExtensionsExtender([]Extension{
		{
			Type: OidcExtensionType,
			Create: func(_ imv1.Runtime, shoot gardener.Shoot) (*gardener.Extension, error) {
				// If oidc is not set on the shoot we skip it
				oidcIndex := slices.IndexFunc(shoot.Spec.Extensions, func(e gardener.Extension) bool {
					return e.Type == OidcExtensionType
				})

				if oidcIndex == -1 {
					return nil, nil
				}
				return NewOIDCExtension()
			},
		},
		{
			Type: auditlogExtensionType,
			Create: func(_ imv1.Runtime, _ gardener.Shoot) (*gardener.Extension, error) {
				return NewAuditLogExtension(auditLogData)
			},
		},
	}, extensionsOnTheShoot)
}

func newExtensionsExtender(extensionsToApply []Extension, currentGardenerExtensions []gardener.Extension) func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
	return func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
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
