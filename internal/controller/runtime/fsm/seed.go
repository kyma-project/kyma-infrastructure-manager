package fsm

import (
	"context"
	gardener_types "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1helper "github.com/gardener/gardener/pkg/apis/core/v1beta1/helper"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"slices"
)

func seedForRegionAvailable(context context.Context, seedClient client.Client, providerType, region string) (bool, []string, error) {
	var seedList gardener_types.SeedList
	var regionsWithSeeds []string

	err := seedClient.List(context, &seedList)

	if err != nil {
		return false, nil, err
	}

	for _, seed := range seedList.Items {
		if seed.Spec.Provider.Type == providerType &&
			seedCanBeUsed(&seed) &&
			!slices.Contains(regionsWithSeeds, seed.Spec.Provider.Region) {
			regionsWithSeeds = append(regionsWithSeeds, seed.Spec.Provider.Region)
		}
	}

	return slices.Contains(regionsWithSeeds, region), regionsWithSeeds, nil
}

func seedCanBeUsed(seed *gardener_types.Seed) bool {
	return seed.DeletionTimestamp == nil && seed.Spec.Settings.Scheduling.Visible && verifySeedReadiness(seed)
}

func verifySeedReadiness(seed *gardener_types.Seed) bool {
	if seed.Status.LastOperation == nil {
		return false
	}

	if cond := v1beta1helper.GetCondition(seed.Status.Conditions, gardener_types.SeedGardenletReady); cond == nil || cond.Status != gardener_types.ConditionTrue {
		return false
	}

	if seed.Spec.Backup != nil {
		if cond := v1beta1helper.GetCondition(seed.Status.Conditions, gardener_types.SeedBackupBucketsReady); cond == nil || cond.Status != gardener_types.ConditionTrue {
			return false
		}
	}

	return true
}
