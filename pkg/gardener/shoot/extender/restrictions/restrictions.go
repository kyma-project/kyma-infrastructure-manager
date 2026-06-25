package restrictions

import (
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
)

const (
	euAccessAddons = "support.gardener.cloud/eu-access-for-cluster-addons"
	euAccessNodes  = "support.gardener.cloud/eu-access-for-cluster-nodes"
)

var regions = map[string]bool{
	"cf-eu11": true,
	"cf-ch20": true,
	"cf-eu01": true,
	"cf-eu02": true,
	"cf-eu31": true}

func ExtendWithAccessRestriction() func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
	return func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
		if isEuAccess(runtime.Spec.Shoot.PlatformRegion) {
			extendWithEuAccess(shoot)
		}
		return nil
	}
}

func extendWithEuAccess(shoot *gardener.Shoot) {
	shoot.Spec.AccessRestrictions = append(shoot.Spec.AccessRestrictions, gardener.AccessRestrictionWithOptions{
		AccessRestriction: gardener.AccessRestriction{
			Name: "eu-access-only",
		},
		Options: map[string]string{
			euAccessAddons: "true",
			euAccessNodes:  "true",
		},
	})
}

func isEuAccess(platformRegion string) bool {
	return regions[platformRegion]
}
