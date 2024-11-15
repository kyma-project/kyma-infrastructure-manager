package extender

import (
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
)

func ExtendWithSeedInSameRegion(runtime imv1.Runtime, shoot *gardener.Shoot) error {

	if runtime.Spec.Shoot.EnforceSeedLocation != nil && *runtime.Spec.Shoot.EnforceSeedLocation {
		//add required label to the shoot
	}

	return nil
}
