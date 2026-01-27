package gardener

import (
	"fmt"
	"os"
	"sync"

	gardeneroidc "github.com/gardener/oidc-webhook-authenticator/apis/authentication/v1alpha1"
	kyma "github.com/kyma-project/lifecycle-manager/api/v1beta2"
	registrycacheapi "github.com/kyma-project/registry-cache/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/discovery"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	// runtimeScheme is used for runtime cluster clients; initialize once to avoid concurrent AddToScheme calls
	runtimeScheme     = runtime.NewScheme()
	runtimeSchemeOnce sync.Once
)

func initRuntimeScheme() {
	// register core Kubernetes types and CRDs used by runtime clients
	utilruntime.Must(clientgoscheme.AddToScheme(runtimeScheme))
	utilruntime.Must(registrycacheapi.AddToScheme(runtimeScheme))
	utilruntime.Must(kyma.AddToScheme(runtimeScheme))
	utilruntime.Must(gardeneroidc.AddToScheme(runtimeScheme))
	utilruntime.Must(apiextensions.AddToScheme(runtimeScheme))
}

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

func GetRuntimeClient(secret corev1.Secret) (client.Client, error) {

	restConfig, err := clientcmd.RESTConfigFromKubeConfig(secret.Data[kubeconfigSecretKey])
	if err != nil {
		return nil, err
	}

	// ensure the shared runtime scheme is initialized exactly once
	runtimeSchemeOnce.Do(initRuntimeScheme)

	runtimeClient, err := client.New(restConfig, client.Options{Scheme: runtimeScheme})
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
