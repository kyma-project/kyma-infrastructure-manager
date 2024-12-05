package extender

import (
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
)

func NewResourcesExtenderForPatch(resources []gardener.NamedResourceReference) func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
	return func(_ imv1.Runtime, shoot *gardener.Shoot) error {
		if resources != nil {
			shoot.Spec.Resources = resources
		}

		return nil
	}
}
