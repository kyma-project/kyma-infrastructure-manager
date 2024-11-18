package extender

import (
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ExtendWithSeedInSameRegion(runtime imv1.Runtime, shoot *gardener.Shoot) error {

	if runtime.Spec.Shoot.EnforceSeedLocation != nil && *runtime.Spec.Shoot.EnforceSeedLocation && runtime.Spec.Shoot.Region != "" {
		//add required label to the shoot

		shoot.Spec.SeedSelector = &gardener.SeedSelector{
			LabelSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"seed.gardener.cloud/region": runtime.Spec.Shoot.Region,
				},
			},
		}
	}

	return nil
}
