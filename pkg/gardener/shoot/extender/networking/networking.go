package networking

import (
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	hyperscaler2 "github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/hyperscaler"
)

func ExtendWithNetworking() func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
	return func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
		if isDualStackIPsEnabled(runtime.Spec.Shoot.Networking.DualStack, runtime.Spec.Shoot.Provider.Type) {
			extendWithDualIPs(shoot)
		}
		return nil
	}
}

func isDualStackIPsEnabled(dualStack *bool, providerType string) bool {
	if dualStack == nil {
		return false
	}
	return *dualStack == true && (providerType == hyperscaler2.TypeGCP || providerType == hyperscaler2.TypeAWS)
}

func extendWithDualIPs(shoot *gardener.Shoot) {
	shoot.Spec.Networking.IPFamilies = shoot.Spec.Networking.IPFamilies[:0] // reset existing IPFamilies
	shoot.Spec.Networking.IPFamilies = append(shoot.Spec.Networking.IPFamilies, gardener.IPFamilyIPv4, gardener.IPFamilyIPv6)
}
