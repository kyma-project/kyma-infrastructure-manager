package shoot

import (
	gardenerv1beta "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
)

func dnsExtender(imv1.RuntimeShoot, *gardenerv1beta.Shoot) error {
	return nil
}
