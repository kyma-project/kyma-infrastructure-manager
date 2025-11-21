package extender

import (
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
)

func ExtendWithCredentialsBinding(runtime imv1.Runtime, shoot *gardener.Shoot) error {
	labels := map[string]string{
		ShootGlobalAccountLabel: runtime.Labels[RuntimeGlobalAccountLabel],
		ShootSubAccountLabel:    runtime.Labels[RuntimeSubaccountLabel],
	}

	shoot.Labels = labels

	return nil
}
