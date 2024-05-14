package hyperscaler

import (
	gardenerv1beta "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
)

func awsProviderExtender(imv1.RuntimeShoot, *gardenerv1beta.Shoot) error {
	return nil
}
