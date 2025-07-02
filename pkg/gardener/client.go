package gardener

import (
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"

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

func GetRuntimeClient(secret corev1.Secret) (client.Client, error) {

	restConfig, err := clientcmd.RESTConfigFromKubeConfig(secret.Data[kubeconfigSecretKey])
	if err != nil {
		return nil, err
	}

	runtimeClientWithAdmin, err := client.New(restConfig, client.Options{})
	if err != nil {
		return nil, err
	}

	return runtimeClientWithAdmin, nil
}
