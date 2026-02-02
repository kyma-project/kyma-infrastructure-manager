package fsm

import (
	"context"
	"fmt"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:generate mockery --name=RuntimeClientGetter
//mockery:generate: false
type RuntimeClientGetter interface {
	Get(ctx context.Context, runtime imv1.Runtime) (client.Client, error)
}

type DynamicRuntimeClientGetter interface {
	Get(ctx context.Context, runtime imv1.Runtime) (dynamic.Interface, discovery.DiscoveryInterface, error)
}

// runtimeClientGetterWithScheme will use a provided scheme when building runtime clients.
type runtimeClientGetterWithScheme struct {
	kcpClient client.Client
	scheme    *runtime.Scheme
}

type runtimeDynamicClientGetter struct {
	kcpClient client.Client
}

// NewRuntimeClientGetterWithScheme returns a RuntimeClientGetter that builds runtime clients
// using the provided prebuilt scheme. This avoids registering scheme types concurrently.
func NewRuntimeClientGetterWithScheme(kcpClient client.Client, scheme *runtime.Scheme) RuntimeClientGetter {
	return &runtimeClientGetterWithScheme{
		kcpClient: kcpClient,
		scheme:    scheme,
	}
}

func NewRuntimeDynamicClientGetter(kcpClient client.Client) DynamicRuntimeClientGetter {
	return &runtimeDynamicClientGetter{
		kcpClient: kcpClient,
	}
}

func (r *runtimeClientGetterWithScheme) Get(ctx context.Context, runtime imv1.Runtime) (client.Client, error) {
	secret, err := getKubeconfigSecret(ctx, r.kcpClient, runtime.Labels[imv1.LabelKymaRuntimeID], runtime.Namespace)
	if err != nil {
		return nil, err
	}

	return gardener.GetRuntimeClientWithScheme(secret, r.scheme)
}

func (r *runtimeDynamicClientGetter) Get(ctx context.Context, runtime imv1.Runtime) (dynamic.Interface, discovery.DiscoveryInterface, error) {
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
