package networking

import (
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	hyperscaler2 "github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/hyperscaler"
)

func ExtendWithNetworking() func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
	return func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
		if canEnableDualStackIPs(runtime.Spec.Shoot.Provider.Type) {
			extendWithDualIPs(shoot)
		}
		// if other provider is used, Gardener by default configures IPv4 only, so no action is needed
		return nil
	}
}

func canEnableDualStackIPs(providerType string) bool {
	return providerType == hyperscaler2.TypeGCP || providerType == hyperscaler2.TypeAWS
}

func extendWithDualIPs(shoot *gardener.Shoot) {

	if shoot.Spec.Networking == nil {
		shoot.Spec.Networking = &gardener.Networking{
			IPFamilies: []gardener.IPFamily{gardener.IPFamilyIPv4, gardener.IPFamilyIPv6},
		}
	} else {
		shoot.Spec.Networking.IPFamilies = []gardener.IPFamily{gardener.IPFamilyIPv4, gardener.IPFamilyIPv6}
	}
}
