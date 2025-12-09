package rtbootstrapper

import (
	"context"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Installer struct {
	config          Config
	kcpClient       client.Client
	manifestApplier *ManifestApplier
}

type InstallationStatus int

const (
	StatusNotStarted = iota
	StatusInProgress
	StatusReady
	StatusFailed
)

type Config struct {
	PullSecretName           string
	ClusterTrustBundleName   string
	ManifestsPath            string
	DeploymentNamespacedName string
	ConfigPath               string
}

//mockery:generate: true
type RuntimeClientGetter interface {
	Get(ctx context.Context, runtime imv1.Runtime) (client.Client, error)
}

// TODO: consider using one interface with two methods
//
//mockery:generate: true
type RuntimeDynamicClientGetter interface {
	Get(ctx context.Context, runtime imv1.Runtime) (dynamic.Interface, discovery.DiscoveryInterface, error)
}

func NewInstaller(config Config, kcpClient client.Client, runtimeClientGetter RuntimeClientGetter, runtimeDynamicClientGetter RuntimeDynamicClientGetter) (*Installer, error) {
	err := validate(config)
	if err != nil {
		return nil, err
	}

	return &Installer{
		config:          config,
		kcpClient:       kcpClient,
		manifestApplier: NewManifestApplier(config.ManifestsPath, types.NamespacedName{Name: config.DeploymentNamespacedName, Namespace: "default"}, runtimeClientGetter, runtimeDynamicClientGetter),
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
	return r.manifestApplier.ApplyManifests(ctx, runtime)
}

func (r *Installer) Status(ctx context.Context, runtime imv1.Runtime) (InstallationStatus, error) {
	return r.manifestApplier.Status(ctx, runtime)
}
