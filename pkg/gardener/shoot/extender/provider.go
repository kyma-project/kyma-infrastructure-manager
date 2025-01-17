package extender

import (
	"fmt"
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

func NewProviderExtenderForCreateOperation(enableIMDSv2 bool, defMachineImgName, defMachineImgVer string) func(rt imv1.Runtime, shoot *gardener.Shoot) error {
	return func(rt imv1.Runtime, shoot *gardener.Shoot) error {
		provider := &shoot.Spec.Provider
		provider.Type = rt.Spec.Shoot.Provider.Type
		provider.Workers = rt.Spec.Shoot.Provider.Workers

		if rt.Spec.Shoot.Provider.AdditionalWorkers != nil {
			provider.Workers = append(provider.Workers, *rt.Spec.Shoot.Provider.AdditionalWorkers...)
		}

		var err error
		var controlPlaneConf, infraConfig *runtime.RawExtension
		zones, err := getNetworkingZonesFromWorkers(provider.Workers)

		if err != nil {
			return err
		}

		infraConfig, controlPlaneConf, err = getConfig(rt.Spec.Shoot, zones)
		if err != nil {
			return err
		}

		if rt.Spec.Shoot.Provider.ControlPlaneConfig != nil {
			controlPlaneConf = rt.Spec.Shoot.Provider.ControlPlaneConfig
		}

		if rt.Spec.Shoot.Provider.InfrastructureConfig != nil {
			infraConfig = rt.Spec.Shoot.Provider.InfrastructureConfig
		}

		provider.ControlPlaneConfig = controlPlaneConf
		provider.InfrastructureConfig = infraConfig

		setMachineImage(provider, defMachineImgName, defMachineImgVer)
		err = setWorkerConfig(provider, provider.Type, enableIMDSv2)
		setWorkerSettings(provider)

		return err
	}
}

// Zones for patching workes are taken from existing shoot workers
//
// Zones for patching infrastructureConfig are processed as follows:
// For Azure and AWS zones are stored in InfrastructureConfig and can be possibly updated if new workers with new zones are added
// For GCP single zone (one from defined in workers) is stored in ControlPlaneConfig and its value is immutable.
// For Openstack no zone information is stored neither in InfrastructureConfig nor in ControlPlaneConfig
// So only for Azure and AWS zone setup can be changed in InfrastructureConfig scope.
// For other providers we use existing data for patching the shoot
func NewProviderExtenderPatchOperation(enableIMDSv2 bool, defMachineImgName, defMachineImgVer string, shootWorkers []gardener.Worker, existingInfraConfig *runtime.RawExtension, existingControlPlaneConfig *runtime.RawExtension) func(rt imv1.Runtime, shoot *gardener.Shoot) error {
	return func(rt imv1.Runtime, shoot *gardener.Shoot) error {
		provider := &shoot.Spec.Provider
		provider.Type = rt.Spec.Shoot.Provider.Type
		provider.Workers = rt.Spec.Shoot.Provider.Workers

		if rt.Spec.Shoot.Provider.AdditionalWorkers != nil {
			provider.Workers = append(provider.Workers, *rt.Spec.Shoot.Provider.AdditionalWorkers...)
		}

		workerZones, err := getNetworkingZonesFromWorkers(provider.Workers)
		if err != nil {
			return err
		}

		controlPlaneConf, infraConfig, err := getProviderConfigsForPatch(rt, workerZones, existingInfraConfig, existingControlPlaneConfig)
		if err != nil {
			return err
		}

		provider.ControlPlaneConfig = controlPlaneConf
		provider.InfrastructureConfig = infraConfig

		setMachineImage(provider, defMachineImgName, defMachineImgVer)

		if err := setWorkerConfig(provider, provider.Type, enableIMDSv2); err != nil {
			return err
		}

		setWorkerSettings(provider)
		alignWorkersWithShoot(provider, shootWorkers)

		return nil
	}
}

func getProviderConfigsForPatch(rt imv1.Runtime, workerZones []string, existingInfraConfig, existingControlPlaneConfig *runtime.RawExtension) (*runtime.RawExtension, *runtime.RawExtension, error) {
	var controlPlaneConf, infraConfig *runtime.RawExtension

	if rt.Spec.Shoot.Provider.Type == hyperscaler.TypeAzure || rt.Spec.Shoot.Provider.Type == hyperscaler.TypeAWS {
		infraConfigZones, err := getZonesFromInfrastructureConfig(rt.Spec.Shoot.Provider.Type, existingInfraConfig)
		if err != nil {
			return nil, nil, err
		}
		// extend infrastructure zones collection if new workers are added with new zones
		for _, zone := range workerZones {
			if !slices.Contains(infraConfigZones, zone) {
				infraConfigZones = append(infraConfigZones, zone)
			}
		}

		infraConfig, controlPlaneConf, err = getConfig(rt.Spec.Shoot, infraConfigZones)
		if err != nil {
			return nil, nil, err
		}
	} else {
		infraConfig = existingInfraConfig
		controlPlaneConf = existingControlPlaneConfig
	}

	if rt.Spec.Shoot.Provider.ControlPlaneConfig != nil {
		controlPlaneConf = rt.Spec.Shoot.Provider.ControlPlaneConfig
	}

	if rt.Spec.Shoot.Provider.InfrastructureConfig != nil {
		infraConfig = rt.Spec.Shoot.Provider.InfrastructureConfig
	}

	return controlPlaneConf, infraConfig, nil
}

// parse infrastructure config to get current set of networking zones
func getZonesFromInfrastructureConfig(providerType string, infraConfig *runtime.RawExtension) ([]string, error) {
	if infraConfig == nil {
		return nil, errors.New("infrastructureConfig is nil")
	}

	var zones []string

	switch providerType {
	case hyperscaler.TypeAWS:
		infraConfig, err := aws.DecodeInfrastructureConfig(infraConfig.Raw)
		if err != nil {
			return nil, err
		}
		for _, zone := range infraConfig.Networks.Zones {
			zones = append(zones, zone.Name)
		}
	case hyperscaler.TypeAzure:
		infraConfig, err := azure.DecodeInfrastructureConfig(infraConfig.Raw)
		if err != nil {
			return nil, err
		}
		for _, zone := range infraConfig.Networks.Zones {
			zones = append(zones, fmt.Sprint(zone.Name))
		}
	default:
		return nil, errors.New("read zones from infrastructureConfig - provider not supported")
	}
	return zones, nil
}

type InfrastructureProviderFunc func(workersCidr string, zones []string) ([]byte, error)
type ControlPlaneProviderFunc func(zones []string) ([]byte, error)

func getConfig(runtimeShoot imv1.RuntimeShoot, zones []string) (infrastructureConfig *runtime.RawExtension, controlPlaneConfig *runtime.RawExtension, err error) {
	getConfigForProvider := func(runtimeShoot imv1.RuntimeShoot, infrastructureConfigFunc InfrastructureProviderFunc, controlPlaneConfigFunc ControlPlaneProviderFunc) (*runtime.RawExtension, *runtime.RawExtension, error) {
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

// Get set of zones from first worker.
// All other workers should have the same zones provided in the same order.
// Otherwise fail
func getNetworkingZonesFromWorkers(workers []gardener.Worker) ([]string, error) {
	var zones []string

	if len(workers) == 0 {
		return nil, errors.New("no workers provided")
	}

	for _, zone := range workers[0].Zones {
		if !slices.Contains(zones, zone) {
			zones = append(zones, zone)
		} else {
			return nil, fmt.Errorf("duplicate zone name detected for worker %s", workers[0].Name)
		}
	}

	if len(zones) == 0 {
		return nil, fmt.Errorf("no networking zones provided for worker %s", workers[0].Name)
	}

	if len(workers) == 1 {
		return zones, nil
	}

	for _, worker := range workers {
		if !slices.Equal(worker.Zones, zones) {
			return nil, errors.New("workers have specified different zones set, or zones are in different order")
		}
	}

	return zones, nil
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

// It sets the machine image name and version to the values specified in the Runtime worker configuration.
// If any value is not specified in the Runtime, it sets it as `machineImage.defaultVersion` or `machineImage.defaultName`, set in `converter_config.json`.
func setMachineImage(provider *gardener.Provider, defMachineImgName, defMachineImgVer string) {
	for i := 0; i < len(provider.Workers); i++ {
		worker := &provider.Workers[i]

		if worker.Machine.Image == nil {
			worker.Machine.Image = &gardener.ShootMachineImage{
				Name:    defMachineImgName,
				Version: &defMachineImgVer,
			}
		}
		if worker.Machine.Image.Version == nil || *worker.Machine.Image.Version == "" {
			worker.Machine.Image.Version = &defMachineImgVer
		}

		if worker.Machine.Image.Name == "" {
			worker.Machine.Image.Name = defMachineImgName
		}
	}
}

// We can't predict what will be the order of zones stored by Gardener.
// Without this patch, gardener's admission webhook might reject the request if the zones order does not match.
// If the current image version with the same name on Shoot is greater than the version, it sets the version to the current machine image version.
func alignWorkersWithShoot(provider *gardener.Provider, existingWorkers []gardener.Worker) {
	existingWorkersMap := make(map[string]gardener.Worker)
	for _, existing := range existingWorkers {
		existingWorkersMap[existing.Name] = existing
	}

	for i := range provider.Workers {
		alignedWorker := &provider.Workers[i]

		if existing, found := existingWorkersMap[alignedWorker.Name]; found {
			alignedWorker.Zones = existing.Zones
			alignWorkerMachineImageVersion(alignedWorker.Machine.Image, existing.Machine.Image)
		}
	}
}

func alignWorkerMachineImageVersion(workerImage *gardener.ShootMachineImage, shootWorkerImage *gardener.ShootMachineImage) {
	if shootWorkerImage == nil || workerImage == nil || workerImage.Name != shootWorkerImage.Name {
		return
	}

	if shootWorkerImage.Version == nil || *shootWorkerImage.Version == *workerImage.Version {
		return
	}

	if result, err := compareVersions(*workerImage.Version, *shootWorkerImage.Version); err == nil && result < 0 {
		workerImage.Version = shootWorkerImage.Version
	}
}
