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
	err := validate(config)
	if err != nil {
		return nil, err
	}

	return &Installer{
		config:          config,
		kcpClient:       kcpClient,
		manifestApplier: NewManifestApplier(config.ManifestsPath, toNamespacedName(config.DeploymentNamespacedName), runtimeClientGetter, runtimeDynamicClientGetter),
	}, nil
}

func validate(config Config) error {
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
	// TODO: add validations for pull secret and configuration

	return nil
}

func (r *Installer) Install(ctx context.Context, runtime imv1.Runtime) error {
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
