package initialisation

import (
	"context"
	"fmt"
	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardener_types "github.com/gardener/gardener/pkg/client/core/clientset/versioned/typed/core/v1beta1"
	gardener_oidc "github.com/gardener/oidc-webhook-authenticator/apis/authentication/v1alpha1"
	v1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/kubeconfig"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	crdv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

const (
	kubeconfigSecretKey = "config"
	timeoutK8sOperation = 20 * time.Second
)

func addToScheme(s *runtime.Scheme) error {
	for _, add := range []func(s *runtime.Scheme) error{
		corev1.AddToScheme,
		v1.AddToScheme,
	} {
		if err := add(s); err != nil {
			return fmt.Errorf("unable to add scheme: %w", err)
		}
	}
	return nil
}

func CreateKcpClient(cfg *Config) (client.Client, error) {
	restCfg, err := clientcmd.BuildConfigFromFlags("", cfg.KcpKubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch rest config: %w", err)
	}

	scheme := runtime.NewScheme()
	if err := addToScheme(scheme); err != nil {
		return nil, err
	}

	return client.New(restCfg, client.Options{
		Scheme: scheme,
	})
}

func SetupKubernetesKubeconfigProvider(kubeconfigPath string, namespace string, expirationTime time.Duration) (kubeconfig.Provider, error) {
	restConfig, err := gardener.NewRestConfigFromFile(kubeconfigPath)
	if err != nil {
		return kubeconfig.Provider{}, err
	}

	gardenerClientSet, err := gardener_types.NewForConfig(restConfig)
	if err != nil {
		return kubeconfig.Provider{}, err
	}

	gardenerClient, err := client.New(restConfig, client.Options{})
	if err != nil {
		return kubeconfig.Provider{}, err
	}

	shootClient := gardenerClientSet.Shoots(namespace)
	dynamicKubeconfigAPI := gardenerClient.SubResource("adminkubeconfig")

	err = v1beta1.AddToScheme(gardenerClient.Scheme())
	if err != nil {
		return kubeconfig.Provider{}, errors.Wrap(err, "failed to register Gardener schema")
	}

	return kubeconfig.NewKubeconfigProvider(shootClient,
		dynamicKubeconfigAPI,
		namespace,
		int64(expirationTime.Seconds())), nil
}

func SetupGardenerShootClients(kubeconfigPath, gardenerNamespace string) (gardener_types.ShootInterface, client.Client, error) {
	restConfig, err := gardener.NewRestConfigFromFile(kubeconfigPath)
	if err != nil {
		return nil, nil, err
	}

	gardenerClientSet, err := gardener_types.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, err
	}

	shootClient := gardenerClientSet.Shoots(gardenerNamespace)

	scheme := runtime.NewScheme()
	if err := addToScheme(scheme); err != nil {
		return nil, nil, err
	}

	dynamicClient, err := client.New(restConfig, client.Options{
		Scheme: scheme,
	})

	err = v1beta1.AddToScheme(dynamicClient.Scheme())
	if err != nil {
		return nil, nil, err
	}

	return shootClient, dynamicClient, err
}

//nolint:gochecknoglobals
func GetRuntimeClient(ctx context.Context, kcpClient client.Client, runtimeID string) (client.Client, error) {
	secret, err := getKubeconfigSecret(ctx, kcpClient, runtimeID, "kcp-system")
	if err != nil {
		return nil, err
	}

	restConfig, err := clientcmd.RESTConfigFromKubeConfig(secret.Data[kubeconfigSecretKey])
	if err != nil {
		return nil, err
	}

	scheme := runtime.NewScheme()
	err = gardener_oidc.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}

	err = rbacv1.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}

	err = crdv1.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}

	shootClientWithAdmin, err := client.New(restConfig, client.Options{
		Scheme: scheme,
	})
	if err != nil {
		return nil, err
	}

	return shootClientWithAdmin, nil
}

func getKubeconfigSecret(ctx context.Context, cnt client.Client, runtimeID, namespace string) (corev1.Secret, error) {
	secretName := fmt.Sprintf("kubeconfig-%s", runtimeID)

	var kubeconfigSecret corev1.Secret
	secretKey := types.NamespacedName{Name: secretName, Namespace: namespace}
	getCtx, cancel := context.WithTimeout(ctx, timeoutK8sOperation)
	defer cancel()

	err := cnt.Get(getCtx, secretKey, &kubeconfigSecret)

	if err != nil {
		return corev1.Secret{}, err
	}

	if kubeconfigSecret.Data == nil {
		return corev1.Secret{}, fmt.Errorf("kubeconfig secret `%s` does not contain kubeconfig data", kubeconfigSecret.Name)
	}
	return kubeconfigSecret, nil
}
