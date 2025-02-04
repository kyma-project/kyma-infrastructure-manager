package fsm

import (
	"context"
	gardener_types "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func seedForRegionAvailable(context context.Context, client client.Client, providerType, region string) (bool, []string, error) {
	var seedList gardener_types.SeedList
	var regionsWithSeeds []string

	err := client.List(context, &seedList)
	if err != nil {
		return false, nil, err
	}

	for _, seed := range seedList.Items {
		if seed.Spec.Provider.Type == providerType {
			regionsWithSeeds = append(regionsWithSeeds, seed.Spec.Provider.Region)
		}
	}

	for _, seed := range seedList.Items {
		if seed.Spec.Provider.Region == region && seed.Spec.Provider.Type == providerType {
			return true, regionsWithSeeds, nil
		}
	}

	return false, regionsWithSeeds, nil
}
