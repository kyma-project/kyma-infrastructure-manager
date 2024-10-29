package extender

import (
	"slices"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/hyperscaler"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/hyperscaler/aws"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/hyperscaler/azure"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/hyperscaler/gcp"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/hyperscaler/openstack"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
)

func NewProviderExtender(enableIMDSv2 bool, defaultMachineImageName, defaultMachineImageVersion string, currentShootState *gardener.Shoot) func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
	return func(rt imv1.Runtime, shoot *gardener.Shoot) error {
		provider := &shoot.Spec.Provider
		provider.Type = rt.Spec.Shoot.Provider.Type
		provider.Workers = rt.Spec.Shoot.Provider.Workers

		var err error
		var controlPlaneConf *runtime.RawExtension
		provider.InfrastructureConfig, controlPlaneConf, err = getConfig(rt.Spec.Shoot, currentShootState)

		if err != nil {
			return err
		}

		if rt.Spec.Shoot.Provider.ControlPlaneConfig != nil {
			controlPlaneConf = rt.Spec.Shoot.Provider.ControlPlaneConfig
		}

		provider.ControlPlaneConfig = controlPlaneConf

		setDefaultMachineImage(provider, defaultMachineImageName, defaultMachineImageVersion)
		err = setWorkerConfig(provider, provider.Type, enableIMDSv2)
		setWorkerSettings(provider)
		alignWithExistingShoot(provider, currentShootState)

		return err
	}
}

// alignWithExistingShoot replaces the `Runtime CR` for
// - `Provider.workers.zones`,
// - `Provider.InfrastructureConfig`
// as we can't predict what will be the order of zones stored by Gardener.
// Without this patch, gardener's admission webhook might reject the request if the zones order does not match.
func alignWithExistingShoot(provider *gardener.Provider, currentShootState *gardener.Shoot) {
	if currentShootState != nil {
		provider.Workers = currentShootState.Spec.Provider.Workers
		for i, worker := range currentShootState.Spec.Provider.Workers {
			provider.Workers[i].Zones = worker.Zones
		}
	}
}

type InfrastructureProviderFunc func(workersCidr string, zones []string) ([]byte, error)
type ControlPlaneProviderFunc func(zones []string) ([]byte, error)

func getConfig(runtimeShoot imv1.RuntimeShoot, currentShootState *gardener.Shoot) (infrastructureConfig *runtime.RawExtension, controlPlaneConfig *runtime.RawExtension, err error) {
	getConfigForProvider := func(runtimeShoot imv1.RuntimeShoot, infrastructureConfigFunc InfrastructureProviderFunc, controlPlaneConfigFunc ControlPlaneProviderFunc) (*runtime.RawExtension, *runtime.RawExtension, error) {
		zones := getZones(runtimeShoot, currentShootState)

		infrastructureConfigBytes, err := infrastructureConfigFunc(runtimeShoot.Networking.Nodes, zones)
		if err != nil {
			return nil, nil, err
		}

		controlPlaneConfigBytes, err := controlPlaneConfigFunc(zones)
		if err != nil {
			return nil, nil, err
		}

		return &runtime.RawExtension{Raw: infrastructureConfigBytes}, &runtime.RawExtension{Raw: controlPlaneConfigBytes}, nil
	}

	switch runtimeShoot.Provider.Type {
	case hyperscaler.TypeAWS:
		{
			return getConfigForProvider(runtimeShoot, aws.GetInfrastructureConfig, aws.GetControlPlaneConfig)
		}
	case hyperscaler.TypeAzure:
		{
			// Azure shoots are all zoned, put probably it not be validated here.
			return getConfigForProvider(runtimeShoot, azure.GetInfrastructureConfig, azure.GetControlPlaneConfig)
		}
	case hyperscaler.TypeGCP:
		{
			return getConfigForProvider(runtimeShoot, gcp.GetInfrastructureConfig, gcp.GetControlPlaneConfig)
		}
	case hyperscaler.TypeOpenStack:
		{
			return getConfigForProvider(runtimeShoot, openstack.GetInfrastructureConfig, openstack.GetControlPlaneConfig)
		}
	default:
		return nil, nil, errors.New("provider not supported")
	}
}

func getAWSWorkerConfig() (*runtime.RawExtension, error) {
	workerConfigBytes, err := aws.GetWorkerConfig()
	if err != nil {
		return nil, err
	}

	return &runtime.RawExtension{Raw: workerConfigBytes}, nil
}

func getZones(runtime imv1.RuntimeShoot, currentShootState *gardener.Shoot) []string {
	var workers []gardener.Worker
	// new cluster
	if currentShootState == nil {
		workers = runtime.Provider.Workers
	} else {
		workers = currentShootState.Spec.Provider.Workers
	}

	var zones []string

	for _, worker := range workers {
		for _, zone := range worker.Zones {
			if !slices.Contains(zones, zone) {
				zones = append(zones, zone)
			}
		}
	}

	return zones
}

func setWorkerConfig(provider *gardener.Provider, providerType string, enableIMDSv2 bool) error {
	if providerType != hyperscaler.TypeAWS || !enableIMDSv2 {
		return nil
	}

	for i := 0; i < len(provider.Workers); i++ {
		var err error
		provider.Workers[i].ProviderConfig, err = getAWSWorkerConfig()

		if err != nil {
			return err
		}
	}

	return nil
}

func setWorkerSettings(provider *gardener.Provider) {
	provider.WorkersSettings = &gardener.WorkersSettings{
		SSHAccess: &gardener.SSHAccess{
			Enabled: false,
		},
	}
}

func setDefaultMachineImage(provider *gardener.Provider, defaultMachineImageName, defaultMachineImageVersion string) {
	for i := 0; i < len(provider.Workers); i++ {
		worker := &provider.Workers[i]

		if worker.Machine.Image == nil {
			worker.Machine.Image = &gardener.ShootMachineImage{
				Name:    defaultMachineImageName,
				Version: &defaultMachineImageVersion,
			}

			continue
		}
		machineImageVersion := worker.Machine.Image.Version
		if machineImageVersion == nil || *machineImageVersion == "" {
			machineImageVersion = &defaultMachineImageVersion
		}

		if worker.Machine.Image.Name == "" {
			worker.Machine.Image.Name = defaultMachineImageName
		}

		worker.Machine.Image.Version = machineImageVersion
	}
}
