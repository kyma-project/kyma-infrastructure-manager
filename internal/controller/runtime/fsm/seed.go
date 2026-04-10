package fsm

import (
	"context"
	v1beta1helper "github.com/gardener/gardener/pkg/api/core/v1beta1/helper"
	gardener_types "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"slices"
)

func seedForRegionAvailable(context context.Context, gardenClient client.Client, providerType, region string) (bool, []string, error) {
	var seedList gardener_types.SeedList
	var regionsWithSeeds []string

	err := gardenClient.List(context, &seedList)

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

	// Check that extensions are ready (replaces SeedGardenletReady from earlier versions)
	if cond := v1beta1helper.GetCondition(seed.Status.Conditions, gardener_types.SeedExtensionsReady); cond == nil || cond.Status != gardener_types.ConditionTrue {
		return false
	}

	if seed.Spec.Backup != nil {
		if cond := v1beta1helper.GetCondition(seed.Status.Conditions, gardener_types.SeedBackupBucketsReady); cond == nil || cond.Status != gardener_types.ConditionTrue {
			return false
		}
	}

	return true
}
