package extender

import (
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
)

// Provisioner was setting the following labels:
//- account
//- subaccount

const (
	RuntimeGlobalAccountLabel = "kyma-project.io/global-account-id"
	RuntimeSubaccountLabel    = "kyma-project.io/subaccount-id"
	ShootGlobalAccountLabel   = "account"
	ShootSubAccountLabel      = "subaccount"
)

func ExtendWithLabels(runtime imv1.Runtime, shoot *gardener.Shoot) error {
	labels := map[string]string{
		ShootGlobalAccountLabel: runtime.Labels[RuntimeGlobalAccountLabel],
		ShootSubAccountLabel:    runtime.Labels[RuntimeSubaccountLabel],
		"disper-from-code":      "value",
	}

	shoot.Labels = labels

	return nil
}
