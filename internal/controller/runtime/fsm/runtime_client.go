package fsm

import (
	"context"
	"fmt"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:generate mockery --name=RuntimeClientGetter
//mockery:generate: false
type RuntimeClientGetter interface {
	Get(ctx context.Context, runtime imv1.Runtime) (client.Client, error)
	GetDynamic(ctx context.Context, runtime imv1.Runtime) (*dynamic.DynamicClient, *discovery.DiscoveryClient, error)
}

type runtimeClientGetter struct {
	kcpClient client.Client
}

func NewRuntimeClientGetter(kcpClient client.Client) RuntimeClientGetter {
	return &runtimeClientGetter{
		kcpClient: kcpClient,
	}
}

func (r *runtimeClientGetter) Get(ctx context.Context, runtime imv1.Runtime) (client.Client, error) {
	secret, err := getKubeconfigSecret(ctx, r.kcpClient, runtime.Labels[imv1.LabelKymaRuntimeID], runtime.Namespace)
	if err != nil {
		return nil, err
	}

	return gardener.GetRuntimeClient(secret)
}

func (r *runtimeClientGetter) GetDynamic(ctx context.Context, runtime imv1.Runtime) (*dynamic.DynamicClient, *discovery.DiscoveryClient, error) {
	secret, err := getKubeconfigSecret(ctx, r.kcpClient, runtime.Labels[imv1.LabelKymaRuntimeID], runtime.Namespace)
	if err != nil {
		return nil, nil, err
	}

	return gardener.GetDynamicRuntimeClient(secret)
}

func getKubeconfigSecret(ctx context.Context, cnt client.Client, runtimeID, namespace string) (corev1.Secret, error) {
	secretName := fmt.Sprintf("kubeconfig-%s", runtimeID)

	var kubeconfigSecret corev1.Secret
	secretKey := types.NamespacedName{Name: secretName, Namespace: namespace}

	err := cnt.Get(ctx, secretKey, &kubeconfigSecret)

	if err != nil {
		return corev1.Secret{}, err
	}

	if kubeconfigSecret.Data == nil {
		return corev1.Secret{}, fmt.Errorf("kubeconfig secret `%s` does not contain kubeconfig data", kubeconfigSecret.Name)
	}
	return kubeconfigSecret, nil
}
