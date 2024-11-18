package extender

import (
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	seedRegionSelectorLabel = "seed.gardener.cloud/region"
)

// ExtendWithSeedInSameRegion creates a new extender function that can enforce shoot seed location to be the same as shoot region
// When EnforceSeedLocation flag in set on RuntimeCR to true it adds special seedSelector field with labelSelector to match seed region with shoot region
func ExtendWithSeedInSameRegion(runtime imv1.Runtime, shoot *gardener.Shoot) error {
	if runtime.Spec.Shoot.EnforceSeedLocation != nil && *runtime.Spec.Shoot.EnforceSeedLocation && runtime.Spec.Shoot.Region != "" {
		shoot.Spec.SeedSelector = &gardener.SeedSelector{
			LabelSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					seedRegionSelectorLabel: runtime.Spec.Shoot.Region,
				},
			},
		}
	}
	return nil
}
