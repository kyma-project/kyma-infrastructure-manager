package registrycache

import (
	"context"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ConfigExplorer struct {
	shootClient client.Client
}

func NewConfigExplorer(ctx context.Context, kcpClient client.Client, runtime imv1.Runtime) (ConfigExplorer, error) {
	shootClient, err := gardener.GetShootClient(ctx, kcpClient, runtime)
	if err != nil {
		return ConfigExplorer{}, err
	}

	return ConfigExplorer{
		shootClient: shootClient,
	}, nil
}

func (c *ConfigExplorer) RegistryCacheConfigExists() (bool, error) {
	return true, nil
}
