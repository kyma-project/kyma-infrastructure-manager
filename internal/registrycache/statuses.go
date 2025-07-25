package registrycache

import (
	"context"
	registrycache "github.com/kyma-project/kim-snatch/api/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func UpdateStatusPending(ctx context.Context, runtimeClient client.Client, cache registrycache.RegistryCacheConfig, condition string) error {
	//cache.Status.State = registrycache.PendingState
	//cond := getCondition(&cache, condition)
	//
	//runtimeClient.Update(ctx, &cache)
	return nil
}

func UpdateStatusFailed(ctx context.Context, runtimeClient client.Client, cache registrycache.RegistryCacheConfig, condition string, errorMessage string) error {

	return nil
}

func UpdateStatusReady(ctx context.Context, runtimeClient client.Client, cache registrycache.RegistryCacheConfig, condition string) error {

	return nil
}

func getCondition(cache *registrycache.RegistryCacheConfig, condition string) v1.Condition {
	for _, c := range cache.Status.Conditions {
		if c.Type == condition {
			return c
		}
	}

	return v1.Condition{
		Type:   condition,
		Status: "Unknown",
	}
}
