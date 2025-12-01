package fsm

import (
	"context"
	"github.com/kyma-project/infrastructure-manager/internal/rtbootstrapper"
)

//go:generate mockery --name=RuntimeBootstrapperInstaller
type RuntimeBootstrapperInstaller interface {
	Install(context context.Context, runtimeID string) error
	Status(context context.Context, runtimeID string) (rtbootstrapper.InstallationStatus, error)
}
