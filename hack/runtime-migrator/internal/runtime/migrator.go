package runtime

import (
	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/config"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/kubeconfig"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Migrator struct {
	converterConfig    config.ConverterConfig
	kubeconfigProvider kubeconfig.Provider
	kcpClient          client.Client
}

func NewMigrator(converterConfig config.ConverterConfig, kubeconfigProvider kubeconfig.Provider, kcpClient client.Client) Migrator {
	return Migrator{
		converterConfig:    converterConfig,
		kubeconfigProvider: kubeconfigProvider,
		kcpClient:          kcpClient,
	}
}

func (m Migrator) Do(shoot v1beta1.Shoot) (v1.Runtime, error) {
	return v1.Runtime{}, nil
}
