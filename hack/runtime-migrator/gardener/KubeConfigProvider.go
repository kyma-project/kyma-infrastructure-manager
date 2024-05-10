package gardener

import (
	"context"

	authenticationv1alpha1 "github.com/gardener/gardener/pkg/apis/authentication/v1alpha1"
	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/pkg/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gardenerClient "sigs.k8s.io/controller-runtime/pkg/client"
)

type KubeconfigProvider struct {
	shootNamespace       string
	shootClient          ShootClient
	dynamicKubeconfigAPI DynamicKubeconfigAPI
	expirationInSeconds  int64
}

type ShootClient interface {
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1beta1.Shoot, error)
}

type DynamicKubeconfigAPI interface {
	Create(ctx context.Context, obj gardenerClient.Object, subResource gardenerClient.Object, opts ...gardenerClient.SubResourceCreateOption) error
}

func NewKubeconfigProvider(
	shootClient ShootClient,
	dynamicKubeconfigAPI DynamicKubeconfigAPI,
	shootNamespace string,
	expirationInSeconds int64) KubeconfigProvider {
	return KubeconfigProvider{
		shootClient:          shootClient,
		dynamicKubeconfigAPI: dynamicKubeconfigAPI,
		shootNamespace:       shootNamespace,
		expirationInSeconds:  expirationInSeconds,
	}
}

func (kp KubeconfigProvider) Fetch(ctx context.Context, shootName string) (string, error) {
	shoot, err := kp.shootClient.Get(ctx, shootName, v1.GetOptions{})
	if err != nil {
		return "", errors.Wrap(err, "failed to get shoot")
	}

	adminKubeconfigRequest := authenticationv1alpha1.AdminKubeconfigRequest{
		Spec: authenticationv1alpha1.AdminKubeconfigRequestSpec{
			ExpirationSeconds: &kp.expirationInSeconds,
		},
	}

	err = kp.dynamicKubeconfigAPI.Create(context.Background(), shoot, &adminKubeconfigRequest)
	if err != nil {
		return "", errors.Wrap(err, "failed to create AdminKubeconfigRequest")
	}

	return string(adminKubeconfigRequest.Status.Kubeconfig), nil
}
