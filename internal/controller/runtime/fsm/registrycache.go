package fsm

import (
	"context"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/registrycache"
	"github.com/kyma-project/kim-snatch/api/v1beta1"
	v1 "k8s.io/api/core/v1"
)

func getRegistryCache(ctx context.Context, s *systemState, m *fsm) ([]v1beta1.RegistryCache, error) {
	getSecretFunc := func() (v1.Secret, error) {
		return getKubeconfigSecret(ctx, m.Client, s.instance.Labels[imv1.LabelKymaRuntimeID], s.instance.Namespace)
	}
	configExplorer, err := registrycache.NewConfigExplorer(ctx, getSecretFunc)
	if err != nil {
		return nil, err
	}

	return configExplorer.GetRegistryCacheConfig()
}
