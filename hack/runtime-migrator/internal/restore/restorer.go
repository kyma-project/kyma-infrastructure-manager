package restore

import (
	init2 "github.com/kyma-project/infrastructure-manager/hack/runtime-migrator-app/internal/init"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/kubeconfig"
)

type Restorer struct {
	cfg                init2.Config
	isDryRun           bool
	kubeconfigProvider kubeconfig.Provider
}

func NewRestorer(isDryRun bool, kubeconfigProvider kubeconfig.Provider) Restorer {
	return Restorer{
		isDryRun:           isDryRun,
		kubeconfigProvider: kubeconfigProvider,
	}
}
