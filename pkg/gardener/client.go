package gardener

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"

	gardener_api "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

//go:generate mockery --name=ShootClient
type ShootClient interface {
	Create(ctx context.Context, shoot *gardener_api.Shoot, opts v1.CreateOptions) (*gardener_api.Shoot, error)
	Get(ctx context.Context, name string, opts v1.GetOptions) (*gardener_api.Shoot, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *gardener_api.Shoot, err error)
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

// TODO: Use this function in the Runtime Controller's FSM
func GetShootClient(secret corev1.Secret) (client.Client, error) {

	restConfig, err := clientcmd.RESTConfigFromKubeConfig(secret.Data[kubeconfigSecretKey])
	if err != nil {
		return nil, err
	}

	shootClientWithAdmin, err := client.New(restConfig, client.Options{})
	if err != nil {
		return nil, err
	}

	return shootClientWithAdmin, nil
}
