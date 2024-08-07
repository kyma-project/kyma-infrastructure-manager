package extender

import (
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"k8s.io/utils/ptr"
)

const NetworkFilterType = "shoot-networking-filter"

func ExtendWithNetworkFilter(runtime imv1.Runtime, shoot *gardener.Shoot) error { //nolint:revive
	networkingFilter := gardener.Extension{
		Type:     NetworkFilterType,
		Disabled: ptr.To(false),
	}

	shoot.Spec.Extensions = append(shoot.Spec.Extensions, networkingFilter)

	return nil
}
