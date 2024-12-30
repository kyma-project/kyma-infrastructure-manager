package restore

import (
	"github.com/kyma-project/infrastructure-manager/hack/runtime-migrator-app/internal/initialisation"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/kubeconfig"
)

type Restorer struct {
	cfg                initialisation.Config
	isDryRun           bool
	kubeconfigProvider kubeconfig.Provider
}

func NewRestorer(isDryRun bool, kubeconfigProvider kubeconfig.Provider) Restorer {
	return Restorer{
		isDryRun:           isDryRun,
		kubeconfigProvider: kubeconfigProvider,
	}
}
