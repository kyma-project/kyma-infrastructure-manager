package rtbootstrapper

import (
	"context"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Installer struct {
	config              Config
	kcpClient           client.Client
	runtimeClientGetter RuntimeClientGetter
}

type InstallationStatus int

const (
	StatusNotStarted InstallationStatus = iota
	StatusInProgress
	StatusReady
	StatusFailed
)

type Config struct {
	PullSecretName         string
	ClusterTrustBundleName string
	ManifestsPath          string
	ConfigPath             string
}

//go:generate mockery --name=RuntimeClientGetter
type RuntimeClientGetter interface {
	Get(ctx context.Context, runtime imv1.Runtime) (client.Client, error)
}

func NewInstaller(config Config, kcpClient client.Client, runtimeClientGetter RuntimeClientGetter) (Installer, error) {
	err := validate(config)
	if err != nil {
		return Installer{}, err
	}

	return Installer{
		config:              config,
		kcpClient:           kcpClient,
		runtimeClientGetter: runtimeClientGetter,
	}, nil
}

func validate(config Config) error {
	return nil
}

func (r *Installer) Install(context context.Context, runtimeID string) error {
	return nil
}

func (r *Installer) Status(context context.Context, runtimeID string) (InstallationStatus, error) {
	return StatusReady, nil
}
