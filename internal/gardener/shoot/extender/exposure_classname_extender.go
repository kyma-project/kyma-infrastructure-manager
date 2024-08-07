package extender

import (
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/gardener/shoot/hyperscaler"
	"k8s.io/utils/ptr"
)

// ExposureClassName is required only for OpenStack
func ExtendWithExposureClassName(runtime imv1.Runtime, shoot *gardener.Shoot) error {
	if runtime.Spec.Shoot.Provider.Type == hyperscaler.TypeOpenStack {
		shoot.Spec.ExposureClassName = ptr.To("converged-cloud-internet")
	}

	return nil
}
