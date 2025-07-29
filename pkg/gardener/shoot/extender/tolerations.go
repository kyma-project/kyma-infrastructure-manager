package extender

import (
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/config"
)

func NewTolerationsExtender(config config.TolerationsConfig) func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
	return func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
		for region, tolerations := range config {
			if runtime.Spec.Shoot.Region == region {
				shoot.Spec.Tolerations = make([]gardener.Toleration, len(tolerations))
				for i := range tolerations {
					shoot.Spec.Tolerations[i] = tolerations[i]
				}
			}
		}
		return nil
	}
}
