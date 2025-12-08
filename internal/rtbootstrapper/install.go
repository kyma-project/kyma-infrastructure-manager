package rtbootstrapper

import (
	"context"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/pkg/errors"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Installer struct {
	config                     Config
	kcpClient                  client.Client
	runtimeClientGetter        RuntimeClientGetter
	runtimeDynamicClientGetter RuntimeDynamicClientGetter
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

// TODO: consider using one interface with two methods
type RuntimeDynamicClientGetter interface {
	Get(ctx context.Context, runtime imv1.Runtime) (*dynamic.DynamicClient, *discovery.DiscoveryClient, error)
}

func NewInstaller(config Config, kcpClient client.Client, runtimeClientGetter RuntimeClientGetter, runtimeDynamicClientGetter RuntimeDynamicClientGetter) (*Installer, error) {
	err := validate(config)
	if err != nil {
		return nil, err
	}

	return &Installer{
		config:                     config,
		kcpClient:                  kcpClient,
		runtimeClientGetter:        runtimeClientGetter,
		runtimeDynamicClientGetter: runtimeDynamicClientGetter,
	}, nil
}

func validate(config Config) error {
	if config.ManifestsPath == "" {
		return errors.New("manifests path is required")
	}

	if _, err := os.Stat(config.ManifestsPath); err != nil {
		return errors.Wrapf(err, "manifests path %s is invalid", config.ManifestsPath)
	}

	// TODO: consider validating is file contains valid yaml
	// TODO: add validations for pull secret and configuration

	return nil
}

func (r *Installer) Install(ctx context.Context, runtime imv1.Runtime) error {
	manifestApplier, err := NewManifestApplier(r.config.ManifestsPath, r.runtimeDynamicClientGetter)
	if err != nil {
		return err
	}

	return manifestApplier.ApplyManifests(ctx, runtime)
}

func (r *Installer) Status(ctx context.Context, runtime imv1.Runtime) (InstallationStatus, error) {
	return StatusReady, nil
}
