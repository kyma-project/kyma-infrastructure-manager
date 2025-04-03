package testing

import (
	"context"
	"errors"
	gardener_api "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetFakePatchInterceptorFn(incGeneration bool) func(ctx context.Context, client client.WithWatch, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
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
		if incGeneration {
			shoot.Generation++
		}
		return nil
	}
}

func GetFakePatchInterceptorFnError(returnedError *k8s_errors.StatusError) func(ctx context.Context, client client.WithWatch, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	return func(ctx context.Context, client client.WithWatch, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
		return returnedError
	}
}

func GetFakeUpdateInterceptorFnError(returnedError *k8s_errors.StatusError) func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
	return func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
		return returnedError
	}
}

func GetFakeUpdateInterceptorFn(incGeneration bool) func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
	return func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
		shoot, ok := obj.(*gardener_api.Shoot)
		if !ok {
			return client.Update(ctx, obj, opts...)
		}
		// Update the generation to simulate shoot object being updated using interceptor function.
		if incGeneration {
			shoot.Generation++
		}
		return nil
	}
}
