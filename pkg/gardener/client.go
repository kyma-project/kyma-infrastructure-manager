package gardener

import (
	"fmt"
	"os"

	gardeneroidc "github.com/gardener/oidc-webhook-authenticator/apis/authentication/v1alpha1"
	kyma "github.com/kyma-project/lifecycle-manager/api/v1beta2"
	registrycacheapi "github.com/kyma-project/registry-cache/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func NewRestConfigFromFile(kubeconfigFilePath string) (*restclient.Config, error) {
	rawKubeconfig, err := os.ReadFile(kubeconfigFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Gardener Kubeconfig from path %s: %s", kubeconfigFilePath, err.Error())
	}

	restConfig, err := clientcmd.RESTConfigFromKubeConfig(rawKubeconfig)
	if err != nil {
		return nil, err
	}

	return restConfig, err
}

const (
	kubeconfigSecretKey = "config"
)

// GetRuntimeClientWithScheme creates a controller-runtime client for the given kubeconfig secret
// using the provided scheme (which must already have the required types registered).
func GetRuntimeClientWithScheme(secret corev1.Secret, scheme *runtime.Scheme) (client.Client, error) {
	if secret.Data == nil {
		return nil, fmt.Errorf("kubeconfig secret `%s` does not contain kubeconfig data", secret.Name)
	}

	restConfig, err := clientcmd.RESTConfigFromKubeConfig(secret.Data[kubeconfigSecretKey])
	if err != nil {
		return nil, err
	}

	runtimeClient, err := client.New(restConfig, client.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}

	return runtimeClient, nil
}

// GetRuntimeClient creates a controller-runtime client for the given kubeconfig secret.
// This is a convenience wrapper that builds and registers a scheme locally for the client.
// Prefer using GetRuntimeClientWithScheme with a pre-built scheme to avoid repeated registrations.
func GetRuntimeClient(secret corev1.Secret) (client.Client, error) {
	if secret.Data == nil {
		return nil, fmt.Errorf("kubeconfig secret `%s` does not contain kubeconfig data", secret.Name)
	}

	restConfig, err := clientcmd.RESTConfigFromKubeConfig(secret.Data[kubeconfigSecretKey])
	if err != nil {
		return nil, err
	}

	// Build a fresh scheme for this client
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := registrycacheapi.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := kyma.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := gardeneroidc.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := apiextensions.AddToScheme(scheme); err != nil {
		return nil, err
	}

	runtimeClient, err := client.New(restConfig, client.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}

	return runtimeClient, nil
}

func GetDynamicRuntimeClient(secret corev1.Secret) (*dynamic.DynamicClient, *discovery.DiscoveryClient, error) {
	restConfig, err := clientcmd.RESTConfigFromKubeConfig(secret.Data[kubeconfigSecretKey])
	if err != nil {
		return nil, nil, err
	}

	dynClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("creating dynamic client: %w", err)
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(restConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("creating discovery client: %w", err)
	}

	return dynClient, discoveryClient, nil
}
