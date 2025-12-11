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
	"strings"
)

type Installer struct {
	config          Config
	kcpClient       client.Client
	manifestApplier *ManifestApplier
	configurator    *Configurator
}

type InstallationStatus string

const (
	StatusNotStarted InstallationStatus = "NotStarted"
	StatusInProgress InstallationStatus = "InProgress"
	StatusReady      InstallationStatus = "Ready"
	StatusFailed     InstallationStatus = "Failed"
)

type Config struct {
	PullSecretName           string
	ClusterTrustBundleName   string
	ManifestsPath            string
	DeploymentNamespacedName string
	ConfigName               string
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
	configurator := NewConfigurator(kcpClient, runtimeClientGetter, config)

	err := validate(config, configurator)
	if err != nil {
		return nil, errors.Wrap(err, "installer config is invalid")
	}

	return &Installer{
		config:          config,
		kcpClient:       kcpClient,
		manifestApplier: NewManifestApplier(config.ManifestsPath, toNamespacedName(config.DeploymentNamespacedName), runtimeClientGetter, runtimeDynamicClientGetter),
		configurator:    configurator,
	}, nil
}

func validate(config Config, configurator *Configurator) error {
	if config.ManifestsPath == "" {
		return errors.New("manifests path is required")
	}

	if _, err := os.Stat(config.ManifestsPath); err != nil {
		return errors.Wrapf(err, "manifests path %s is invalid", config.ManifestsPath)
	}

	deploymentNameParts := strings.Split(config.DeploymentNamespacedName, string(types.Separator))

	if len(deploymentNameParts) != 2 || deploymentNameParts[0] == "" || deploymentNameParts[1] == "" {
		return errors.New("deployment namespaced name is invalid")
	}
	// TODO: consider validating is file contains valid yaml

	if config.ConfigName == "" {
		return errors.New("config name is required")
	}

	ctx := context.Background()
	if !configurator.ValidateConfigMap(ctx) {
		return errors.New("unable to find Runtime Bootstrapper ConfigMap in KCP cluster")
	}

	if !configurator.ValidatePullSecretConfig(ctx, config) {
		return errors.New("unable to find Runtime Bootstrapper PullSecret in KCP cluster")
	}

	return nil
}

func (r *Installer) Install(ctx context.Context, runtime imv1.Runtime) error {
	err := r.configurator.Configure(context.Background(), runtime)
	if err != nil {
		return errors.Wrap(err, "failed to prepare for installation Runtime Bootstrapper installation")
	}

	return r.manifestApplier.ApplyManifests(ctx, runtime)
}

func (r *Installer) Status(ctx context.Context, runtime imv1.Runtime) (InstallationStatus, error) {
	return r.manifestApplier.Status(ctx, runtime)
}

func toNamespacedName(namespacedName string) types.NamespacedName {
	nameAndNamespace := strings.Split(namespacedName, string(types.Separator))
	return types.NamespacedName{
		Name:      nameAndNamespace[1],
		Namespace: nameAndNamespace[0],
	}
}
