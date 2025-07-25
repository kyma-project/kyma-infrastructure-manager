package registrycache

import (
	"context"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	registrycache "github.com/kyma-project/kim-snatch/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type StatusManager struct {
	RuntimeClient client.Client
}

func NewStatusManager(runtimeClient client.Client) *StatusManager {
	return &StatusManager{
		RuntimeClient: runtimeClient,
	}
}

func (s StatusManager) SetStatusReady(ctx context.Context, instance imv1.Runtime, condition string) error {
	for _, cache := range instance.Spec.Caching {
		var registryCache registrycache.RegistryCacheConfig

		err := s.RuntimeClient.Get(ctx, client.ObjectKey{
			Name:      cache.Name,
			Namespace: cache.Namespace,
		}, &registryCache)

		if err != nil {
			return err
		}

		err = UpdateStatusReady(s.RuntimeClient, registryCache, condition)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s StatusManager) SetStatusFailed(ctx context.Context, instance imv1.Runtime, condition string, errorMessage string) error {
	for _, cache := range instance.Spec.Caching {
		var registryCache registrycache.RegistryCacheConfig

		err := s.RuntimeClient.Get(ctx, client.ObjectKey{
			Name:      cache.Name,
			Namespace: cache.Namespace,
		}, &registryCache)

		if err != nil {
			return err
		}

		err = UpdateStatusFailed(s.RuntimeClient, registryCache, condition, errorMessage)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s StatusManager) SetStatusPending(ctx context.Context, instance imv1.Runtime, condition string) error {
	for _, cache := range instance.Spec.Caching {
		var registryCache registrycache.RegistryCacheConfig

		err := s.RuntimeClient.Get(ctx, client.ObjectKey{
			Name:      cache.Name,
			Namespace: cache.Namespace,
		}, &registryCache)

		if err != nil {
			return err
		}

		err = UpdateStatusPending(s.RuntimeClient, registryCache, condition)
		if err != nil {
			return err
		}
	}

	return nil
}
