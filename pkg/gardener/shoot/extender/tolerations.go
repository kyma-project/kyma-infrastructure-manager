package extender

import (
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
)

func ExtendWithTolerations(runtime imv1.Runtime, shoot *gardener.Shoot) error {
	if runtime.Spec.Shoot.Region == "me-central2" {
		shoot.Spec.Tolerations = append(shoot.Spec.Tolerations, gardener.Toleration{
			Key: "ksa-assured-workload",
		})
	}
	return nil
}
