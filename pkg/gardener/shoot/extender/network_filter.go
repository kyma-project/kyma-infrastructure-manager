package extender

import (
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
)

const NetworkFilterType = "shoot-networking-filter"

func ExtendWithNetworkFilter(runtime imv1.Runtime, shoot *gardener.Shoot) error { //nolint:revive
	disabled := !runtime.Spec.Security.Networking.Filter.Egress.Enabled

	networkingFilter := gardener.Extension{
		Type:     NetworkFilterType,
		Disabled: &disabled,
	}

	shoot.Spec.Extensions = append(shoot.Spec.Extensions, networkingFilter)

	return nil
}
