package registrycache

import (
	"context"
	registrycache "github.com/kyma-project/kim-snatch/api/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type RegistryCacheConditionType string

const (
	ConditionTypeCacheConfigured RegistryCacheConditionType = "CacheConfigured"
)

func UpdateStatusPending(ctx context.Context, runtimeClient client.Client, cache registrycache.RegistryCacheConfig, condition string) error {
	cache.Status.State = registrycache.PendingState

	cond := getCondition(&cache, condition)

	if cond == nil {
		cond = &v1.Condition{
			Type:   condition,
			Status: v1.ConditionFalse,
		}
		cache.Status.Conditions = append(cache.Status.Conditions, *cond)
	}

	return runtimeClient.Update(ctx, &cache)
}

func UpdateStatusFailed(ctx context.Context, runtimeClient client.Client, cache registrycache.RegistryCacheConfig, condition string, errorMessage string) error {
	return nil
}

func UpdateStatusReady(ctx context.Context, runtimeClient client.Client, cache registrycache.RegistryCacheConfig, condition string) error {

	return nil
}

func getCondition(cache *registrycache.RegistryCacheConfig, condition string) *v1.Condition {
	for _, c := range cache.Status.Conditions {
		if c.Type == condition {
			return &c
		}
	}

	return nil
}
