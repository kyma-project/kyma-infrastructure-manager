package extensions

import (
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/config"
	"slices"
)

type CreateExtension func(runtime imv1.Runtime, shoot *gardener.Shoot) (gardener.Extension, error)

type Extension struct {
	Type    string
	Factory CreateExtension
}

func NewExtensionsExtenderForCreate(config config.ConverterConfig) func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
	return newExtensionsExtender([]Extension{
		{
			Type: DNSExtensionType,
			Factory: func(runtime imv1.Runtime, shoot *gardener.Shoot) (gardener.Extension, error) {
				return NewDNSExtension(runtime.Spec.Shoot.Name, config.DNS.SecretName, config.DNS.DomainPrefix, config.DNS.ProviderType)
			},
		},
		{
			Type: OidcExtensionType,
			Factory: func(runtime imv1.Runtime, shoot *gardener.Shoot) (gardener.Extension, error) {
				return NewOIDCExtension()
			},
		},
	}, nil)
}

func newExtensionsExtender(extensionsToApply []Extension, currentGardenerExtensions []gardener.Extension) func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
	return func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
		for _, ext := range extensionsToApply {
			gardenerExtension, err := ext.Factory(runtime, shoot)
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
