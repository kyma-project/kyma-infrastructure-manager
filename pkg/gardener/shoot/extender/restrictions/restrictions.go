package restrictions

import (
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
)

const (
	euAccessAddons = "support.gardener.cloud/eu-access-for-cluster-addons"
	euAccessNodes  = "support.gardener.cloud/eu-access-for-cluster-nodes"
)

func ExtendWithAccessRestriction() func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
	return func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
		if isEuAccess(runtime.Spec.Shoot.PlatformRegion) {
			shoot.Spec.AccessRestrictions = append(shoot.Spec.AccessRestrictions, extendWithEuAccess())
		}
		return nil
	}
}

func extendWithEuAccess() gardener.AccessRestrictionWithOptions {
	restrictions := make([]gardener.AccessRestrictionWithOptions, 1)
	restrictions[0].AccessRestriction = gardener.AccessRestriction{
		Name: "eu-access-only",
	}
	restrictions[0].Options = map[string]string{
		euAccessAddons: "true",
		euAccessNodes:  "true",
	}
	return restrictions[0]
}

func isEuAccess(platformRegion string) bool {
	switch platformRegion {
	case "cf-eu11":
		return true
	case "cf-ch20":
		return true
	}
	return false
}
