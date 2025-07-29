package registrycache

import (
	"context"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	registrycache "github.com/kyma-project/kim-snatch/api/v1beta1"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func (s StatusManager) SetStatusReady(ctx context.Context, instance imv1.Runtime, conditionType registrycache.ConditionType, conditionReason registrycache.ConditionReason) error {
	for _, cache := range instance.Spec.Caching {
		var registryCache registrycache.RegistryCacheConfig

		err := s.RuntimeClient.Get(ctx, client.ObjectKey{
			Name:      cache.Name,
			Namespace: cache.Namespace,
		}, &registryCache)

		if err != nil {
			return err
		}

		registryCache.UpdateStatusReady(conditionType, conditionReason, metav1.ConditionTrue)

		err = s.RuntimeClient.Status().Update(ctx, &registryCache)
		if err != nil {
			return errors.New("failed to update registry cache status: " + err.Error())
		}
	}

	return nil
}

func (s StatusManager) SetStatusFailed(ctx context.Context, instance imv1.Runtime, conditionType registrycache.ConditionType, conditionReason registrycache.ConditionReason, errorMessage string) error {
	for _, cache := range instance.Spec.Caching {
		var registryCache registrycache.RegistryCacheConfig

		err := s.RuntimeClient.Get(ctx, client.ObjectKey{
			Name:      cache.Name,
			Namespace: cache.Namespace,
		}, &registryCache)

		if err != nil {
			return err
		}

		registryCache.UpdateStatusFailed(conditionType, conditionReason, metav1.ConditionFalse, errorMessage)
		err = s.RuntimeClient.Status().Update(ctx, &registryCache)
		if err != nil {
			return errors.New("failed to update registry cache status: " + err.Error())
		}
	}

	return nil
}

func (s StatusManager) SetStatusPending(ctx context.Context, instance imv1.Runtime, conditionType registrycache.ConditionType, conditionReason registrycache.ConditionReason) error {
	for _, cache := range instance.Spec.Caching {
		var registryCache registrycache.RegistryCacheConfig

		err := s.RuntimeClient.Get(ctx, client.ObjectKey{
			Name:      cache.Name,
			Namespace: cache.Namespace,
		}, &registryCache)

		if err != nil {
			return err
		}

		registryCache.UpdateStatusPending(conditionType, conditionReason, metav1.ConditionUnknown)
		err = s.RuntimeClient.Status().Update(ctx, &registryCache)
		if err != nil {
			return errors.New("failed to update registry cache status: " + err.Error())
		}
	}

	return nil
}
