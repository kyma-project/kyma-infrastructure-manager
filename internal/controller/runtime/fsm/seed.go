package fsm

import (
	"context"
	gardener_types "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func seedForRegionAvailable(client client.Client, region string) (bool, []string, error) {
	var seedList gardener_types.SeedList
	var regionsWithSeeds []string

	err := client.List(context.TODO(), &seedList)
	if err != nil {
		return false, nil, err
	}

	for _, seed := range seedList.Items {
		regionsWithSeeds = append(regionsWithSeeds, seed.Spec.Provider.Region)
	}

	for _, seed := range seedList.Items {
		if seed.Spec.Provider.Region == region {
			return true, regionsWithSeeds, nil
		}
	}

	return false, regionsWithSeeds, nil
}
