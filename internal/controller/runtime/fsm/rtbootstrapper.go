package fsm

import (
	"context"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/rtbootstrapper"
)

//mockery:generate: true
type RuntimeBootstrapperInstaller interface {
	Install(context context.Context, runtime imv1.Runtime) error
	Status(context context.Context, runtime imv1.Runtime) (rtbootstrapper.InstallationStatus, error)
}

//mockery:generate: true
type RuntimeBootstrapperConfigurator interface {
	Configure(context context.Context, runtime imv1.Runtime) error
}
