package fsm

import (
	"context"
	"github.com/kyma-project/infrastructure-manager/internal/rtbootstrapper"
)

//mockery:generate: true
type RuntimeBootstrapperInstaller interface {
	Install(context context.Context, runtimeID string) error
	Status(context context.Context, runtimeID string) (rtbootstrapper.InstallationStatus, error)
}
