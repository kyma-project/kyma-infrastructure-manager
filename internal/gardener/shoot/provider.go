package shoot

import (
	gardenerv1beta "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
)

func providerExtender(runtimeShoot imv1.RuntimeShoot, shoot *gardenerv1beta.Shoot) error {

	shoot.Spec.Provider.Type = runtimeShoot.Provider.Type
	shoot.Spec.Provider.Workers = runtimeShoot.Provider.Workers

	return nil
}
