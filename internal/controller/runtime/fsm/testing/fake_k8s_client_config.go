package testing

import (
	"context"
	"errors"
	gardener_api "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	core_v1 "k8s.io/api/core/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetFakePatchInterceptorFn(incShootGeneration bool) func(ctx context.Context, client client.WithWatch, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	return func(ctx context.Context, client client.WithWatch, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
		// Apply patches are supposed to upsert, but fake client fails if the object doesn't exist,
		// Update the generation to simulate the object being updated using interceptor function.
		if patch.Type() != types.ApplyPatchType {
			return client.Patch(ctx, obj, patch, opts...)
		}
		shoot, ok := obj.(*gardener_api.Shoot)
		if !ok {
			return errors.New("failed to cast object to shoot")
		}
		if incShootGeneration {
			shoot.Generation++
		}
		return nil
	}
}

func GetFakePatchInterceptorForShootsAndConfigMaps(incShootGeneration bool) func(ctx context.Context, client client.WithWatch, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	return func(ctx context.Context, client client.WithWatch, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
		// Apply patches are supposed to upsert, but fake client fails if the object doesn't exist,
		// Update the generation to simulate the object being updated using interceptor function.
		if patch.Type() != types.ApplyPatchType {
			return client.Patch(ctx, obj, patch, opts...)
		}

		// workaround for shoot
		shoot, isShoot := obj.(*gardener_api.Shoot)
		if isShoot {
			if incShootGeneration {
				shoot.Generation++
			}
			return nil
		}

		// workaround for configmaps
		cm, isConfigMap := obj.(*core_v1.ConfigMap)
		if isConfigMap {
			// workaround for https://github.com/kubernetes-sigs/controller-runtime/issues/2341
			// As the Patch with Apply type does not work with fake client, we're first attempting to
			// use Create and then Update if the object already exists.
			err := client.Create(ctx, cm)
			if err != nil && k8s_errors.IsAlreadyExists(err) {
				err := client.Update(ctx, cm)
				if err != nil {
					return err
				}
			}
			return nil
		}

		return client.Patch(ctx, obj, patch, opts...) // If not shoot or configmap, use default patch
	}
}

func GetFakeInterceptorThatThrowsErrorOnCMPatch() func(ctx context.Context, client client.WithWatch, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	return func(ctx context.Context, client client.WithWatch, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
		// Apply patches are supposed to upsert, but fake client fails if the object doesn't exist,
		// Update the generation to simulate the object being updated using interceptor function.
		if patch.Type() != types.ApplyPatchType {
			return client.Patch(ctx, obj, patch, opts...)
		}

		// workaround for shoot
		shoot, isShoot := obj.(*gardener_api.Shoot)
		if isShoot {
			shoot.Generation++
			return nil
		}

		// workaround for configmaps
		_, isConfigMap := obj.(*core_v1.ConfigMap)
		if isConfigMap {
			return errors.New("simulated error to for tests that expect an error when applying a configmap")
		}

		return client.Patch(ctx, obj, patch, opts...) // If not shoot or configmap, use default patch
	}
}

func GetFakeInterceptorThatThrowsErrorOnNSCreation() func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
	return func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
		_, isNamespace := obj.(*core_v1.Namespace)
		if isNamespace {
			return errors.New("simulated error to for tests that expect an error when creating a namespace")
		}

		return client.Create(ctx, obj, opts...)
	}
}
