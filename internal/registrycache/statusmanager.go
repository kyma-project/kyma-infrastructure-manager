package registrycache

import (
	"context"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	registrycache "github.com/kyma-project/kim-snatch/api/v1beta1"
	"github.com/pkg/errors"
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

func (s StatusManager) SetStatusReady(ctx context.Context, instance imv1.Runtime, conditionReason registrycache.ConditionReason) error {
	for _, cache := range instance.Spec.Caching {
		var registryCache registrycache.RegistryCacheConfig

		err := s.RuntimeClient.Get(ctx, client.ObjectKey{
			Name:      cache.Name,
			Namespace: cache.Namespace,
		}, &registryCache)

		if err != nil {
			return err
		}

		registryCache.RegistryCacheConfiguredUpdateStatusReady(conditionReason)

		err = s.RuntimeClient.Status().Update(ctx, &registryCache)
		if err != nil {
			return errors.New("failed to update registry cache status: " + err.Error())
		}
	}

	return nil
}

func (s StatusManager) SetStatusFailed(ctx context.Context, instance imv1.Runtime, conditionReason registrycache.ConditionReason, errorMessage string) error {
	for _, cache := range instance.Spec.Caching {
		var registryCache registrycache.RegistryCacheConfig

		err := s.RuntimeClient.Get(ctx, client.ObjectKey{
			Name:      cache.Name,
			Namespace: cache.Namespace,
		}, &registryCache)

		if err != nil {
			return err
		}

		registryCache.RegistryCacheConfiguredUpdateStatusFailed(conditionReason, errorMessage)
		err = s.RuntimeClient.Status().Update(ctx, &registryCache)
		if err != nil {
			return errors.New("failed to update registry cache status: " + err.Error())
		}
	}

	return nil
}

func (s StatusManager) SetStatusPending(ctx context.Context, instance imv1.Runtime, conditionReason registrycache.ConditionReason) error {
	for _, cache := range instance.Spec.Caching {
		var registryCache registrycache.RegistryCacheConfig

		err := s.RuntimeClient.Get(ctx, client.ObjectKey{
			Name:      cache.Name,
			Namespace: cache.Namespace,
		}, &registryCache)

		if err != nil {
			return err
		}

		registryCache.RegistryCacheConfiguredUpdateStatusPendingUnknown(conditionReason)
		err = s.RuntimeClient.Status().Update(ctx, &registryCache)
		if err != nil {
			return errors.New("failed to update registry cache status: " + err.Error())
		}
	}

	return nil
}
