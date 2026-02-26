package rtbootstrapper

import (
	"context"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
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
	StatusNotStarted    InstallationStatus = "NotStarted"
	StatusInProgress    InstallationStatus = "InProgress"
	StatusReady         InstallationStatus = "Ready"
	StatusFailed        InstallationStatus = "Failed"
	StatusUpgradeNeeded InstallationStatus = "UpgradeNeeded"
)

type Config struct {
	KCPConfig KCPConfig
	SKRConfig SKRConfig
}

type KCPConfig struct {
	PullSecretName         string
	ClusterTrustBundleName string
	ConfigName             string
	ManifestsConfigMapName string
}

type SKRConfig struct {
	Namespace              string
	PullSecretName         string
	ClusterTrustBundleName string
	ConfigName             string
	DeploymentName         string
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

func NewInstaller(config Config, kcpClient client.Client, runtimeClientGetter RuntimeClientGetter, runtimeDynamicClientGetter RuntimeDynamicClientGetter) *Installer {

	return &Installer{
		config:    config,
		kcpClient: kcpClient,
		manifestApplier: NewManifestApplier(config.KCPConfig.ManifestsConfigMapName,
			types.NamespacedName{
				Name:      config.SKRConfig.DeploymentName,
				Namespace: config.SKRConfig.Namespace,
			},
			runtimeClientGetter,
			runtimeDynamicClientGetter,
			kcpClient),
		configurator: NewConfigurator(kcpClient, runtimeClientGetter, config),
	}
}

func (r *Installer) Install(ctx context.Context, runtime imv1.Runtime, manifests string) error {
	return r.manifestApplier.ApplyManifests(ctx, runtime, manifests)
}

func (r *Installer) InstallationInfo(ctx context.Context, runtime imv1.Runtime) (InstallationStatus, string, error) {
	return r.manifestApplier.InstallationInfo(ctx, runtime)
}

func (r *Installer) Configure(ctx context.Context, runtime imv1.Runtime) error {
	return r.configurator.Configure(ctx, runtime)
}

// This method is supposed to be called after upgrade is finished. It can be used to clean up old resources that are no longer available in the new runtime manifests.
func (r *Installer) Cleanup(ctx context.Context, runtime imv1.Runtime) error {
	// No cleanup needed for now. Implement when needed.
	return nil
}

func toNamespacedName(namespacedName string) types.NamespacedName {
	nameAndNamespace := strings.Split(namespacedName, string(types.Separator))
	return types.NamespacedName{
		Name:      nameAndNamespace[1],
		Namespace: nameAndNamespace[0],
	}
}
