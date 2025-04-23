package provider

import (
	"fmt"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender"
	"slices"
	"sort"

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

// InfrastructureConfig and ControlPlaneConfig are generated unless they are specified in the RuntimeCR
func NewProviderExtenderForCreateOperation(enableIMDSv2 bool, defMachineImgName, defMachineImgVer string) func(rt imv1.Runtime, shoot *gardener.Shoot) error {
	return func(rt imv1.Runtime, shoot *gardener.Shoot) error {
		provider := &shoot.Spec.Provider
		provider.Type = rt.Spec.Shoot.Provider.Type
		provider.Workers = rt.Spec.Shoot.Provider.Workers

		//NOTE: we can have this code moved to validation webhook later on
		if len(rt.Spec.Shoot.Provider.Workers) != 1 {
			return errors.New("single main worker is required")
		}

		if rt.Spec.Shoot.Provider.AdditionalWorkers != nil {
			provider.Workers = append(provider.Workers, *rt.Spec.Shoot.Provider.AdditionalWorkers...)
		}

		workerZones, err := getNetworkingZonesFromWorkers(provider.Workers)
		if err != nil {
			return err
		}

		infraConfig, controlPlaneConf, err := getConfig(rt.Spec.Shoot, workerZones, nil)
		if err != nil {
			return err
		}

		controlPlaneConf, infraConfig = overrideConfigIfProvided(rt, infraConfig, controlPlaneConf)

		provider.ControlPlaneConfig = controlPlaneConf
		provider.InfrastructureConfig = infraConfig

		setMachineImage(provider, defMachineImgName, defMachineImgVer)
		if err = setWorkerConfig(provider, provider.Type, enableIMDSv2); err != nil {
			return err
		}
		setWorkerSettings(provider)

		return err
	}
}

// Zones for patching workes are taken from existing shoot workers
// InfrastructureConfig and ControlPlaneConfig are treated as immutable unless they are specified in the RuntimeCR
func NewProviderExtenderPatchOperation(enableIMDSv2 bool, defMachineImgName, defMachineImgVer string, shootWorkers []gardener.Worker, existingInfraConfig, existingControlPlaneConfig *runtime.RawExtension) func(rt imv1.Runtime, shoot *gardener.Shoot) error {
	return func(rt imv1.Runtime, shoot *gardener.Shoot) error {
		provider := &shoot.Spec.Provider
		provider.Type = rt.Spec.Shoot.Provider.Type
		provider.Workers = rt.Spec.Shoot.Provider.Workers

		if len(rt.Spec.Shoot.Provider.Workers) != 1 {
			return errors.New("single main worker is required")
		}

		if rt.Spec.Shoot.Provider.AdditionalWorkers != nil {
			provider.Workers = append(provider.Workers, *rt.Spec.Shoot.Provider.AdditionalWorkers...)
		}

		provider.Workers = sortWorkersToShootOrder(provider.Workers, shootWorkers)

		workerZonesFromRuntime, err := getNetworkingZonesFromWorkers(provider.Workers)
		if err != nil {
			return err
		}

		workerZonesFromShoot, err := getNetworkingZonesFromWorkers(shootWorkers)
		if err != nil {
			return err
		}

		if zonesEqual(workerZonesFromShoot, workerZonesFromRuntime) {
			provider.ControlPlaneConfig = existingControlPlaneConfig
			provider.InfrastructureConfig = existingInfraConfig
		} else {
			infraConfig, controlPlaneConfig, err := getConfig(rt.Spec.Shoot, workerZonesFromRuntime, existingInfraConfig.Raw)
			if err != nil {
				return err
			}

			provider.ControlPlaneConfig = controlPlaneConfig
			provider.InfrastructureConfig = infraConfig

			if err = checkWorkerZonesMatchProviderConfig(rt.Spec.Shoot.Provider.Type, workerZonesFromShoot, controlPlaneConfig, infraConfig, true); err != nil {
				return err
			}
		}

		// final validation

		setMachineImage(provider, defMachineImgName, defMachineImgVer)

		if err := setWorkerConfig(provider, provider.Type, enableIMDSv2); err != nil {
			return err
		}

		setWorkerSettings(provider)
		alignWorkersWithGardener(provider, shootWorkers)

		return nil
	}
}

func zonesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for _, zone := range a {
		if !slices.Contains(b, zone) {
			return false
		}
	}

	return true
}

func sortWorkersToShootOrder(runtimeWorkers []gardener.Worker, shootWorkers []gardener.Worker) []gardener.Worker {
	sortedWorkers := make([]gardener.Worker, len(runtimeWorkers))
	copy(sortedWorkers, runtimeWorkers)

	sort.Slice(sortedWorkers, func(i, j int) bool {
		index1 := slices.IndexFunc(shootWorkers, func(worker gardener.Worker) bool {
			return worker.Name == runtimeWorkers[i].Name
		})

		if index1 == -1 {
			index1 = len(runtimeWorkers) + 1
		}

		index2 := slices.IndexFunc(shootWorkers, func(worker gardener.Worker) bool {
			return worker.Name == runtimeWorkers[j].Name
		})

		if index2 == -1 {
			index1 = len(runtimeWorkers) + 1
		}

		return index1 < index2
	})

	return sortedWorkers
}
func checkWorkerZonesMatchProviderConfig(providerType string, shootZones []string, ctrlPlaneConfig *runtime.RawExtension, infraConfig *runtime.RawExtension, patchValidation bool) error {
	if providerType == hyperscaler.TypeAzure || providerType == hyperscaler.TypeAWS {
		infraConfigZones, err := getZonesFromProviderConfig(providerType, infraConfig)
		if err != nil {
			return err
		}

		// workaround for legacy azure-lite shoots where networking zones are not specified in the infrastructureConfig
		// such shoots are treated as correct, and we can skipp the validation of worker zones with infrastructureConfig zones
		if patchValidation && providerType == hyperscaler.TypeAzure && len(infraConfigZones) == 0 {
			return nil
		}

		for _, zone := range shootZones {
			if !slices.Contains(infraConfigZones, zone) {
				return fmt.Errorf("one of workers is using networking zone not specified in the %s infrastructureConfig: %s", providerType, zone)
			}
		}
	}
	if providerType == hyperscaler.TypeGCP {
		ctrlPlaneZones, err := getZonesFromProviderConfig(providerType, ctrlPlaneConfig)
		if err != nil {
			return err
		}

		if len(ctrlPlaneZones) == 0 {
			return fmt.Errorf("cannot validate workers zones against GCP controlPlaneConfig, cannot read current networking zone")
		}

		if !slices.Contains(shootZones, ctrlPlaneZones[0]) {
			return fmt.Errorf("none of workers is using networking zone specified in the controlPlaneConfig: %s", ctrlPlaneZones[0])
		}
	}

	return nil
}

func overrideConfigIfProvided(rt imv1.Runtime, existingInfraConfig, existingControlPlaneConfig *runtime.RawExtension) (*runtime.RawExtension, *runtime.RawExtension) {
	controlPlaneConf := getConfigOrDefault(rt.Spec.Shoot.Provider.ControlPlaneConfig, existingControlPlaneConfig)
	infraConfig := getConfigOrDefault(rt.Spec.Shoot.Provider.InfrastructureConfig, existingInfraConfig)
	return controlPlaneConf, infraConfig
}

func getConfigOrDefault(config, defaultConfig *runtime.RawExtension) *runtime.RawExtension {
	if config != nil {
		return config
	}
	return defaultConfig
}

// parse infrastructureConfig or controlPlaneConfig to get current set of networking zones
func getZonesFromProviderConfig(providerType string, extensionConfig *runtime.RawExtension) ([]string, error) {
	if extensionConfig == nil {
		return nil, errors.New("infrastructureConfig is nil")
	}

	var zones []string

	switch providerType {
	case hyperscaler.TypeAWS:
		infraConfig, err := aws.DecodeInfrastructureConfig(extensionConfig.Raw)
		if err != nil {
			return nil, err
		}
		for _, zone := range infraConfig.Networks.Zones {
			zones = append(zones, zone.Name)
		}
	case hyperscaler.TypeAzure:
		infraConfig, err := azure.DecodeInfrastructureConfig(extensionConfig.Raw)
		if err != nil {
			return nil, err
		}
		for _, zone := range infraConfig.Networks.Zones {
			zones = append(zones, fmt.Sprint(zone.Name))
		}
	case hyperscaler.TypeGCP:
		{
			ctrlPlaneConfig, err := gcp.DecodeControlPlaneConfig(extensionConfig.Raw)
			if err != nil {
				return nil, err
			}
			zones = append(zones, ctrlPlaneConfig.Zone)
		}
	default:
		return nil, errors.New("read zones from infrastructureConfig - provider not supported")
	}
	return zones, nil
}

type InfrastructureProviderFunc func(workersCidr string, zones []string) ([]byte, error)
type ControlPlaneProviderFunc func(zones []string) ([]byte, error)

func getConfig(runtimeShoot imv1.RuntimeShoot, zones []string, existingInfrastructureConfig []byte) (infrastructureConfig *runtime.RawExtension, controlPlaneConfig *runtime.RawExtension, err error) {
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
			if existingInfrastructureConfig != nil {
				return getConfigForProvider(runtimeShoot, func(workersCidr string, zones []string) ([]byte, error) {
					return aws.GetInfrastructureConfigForPatch(workersCidr, zones, existingInfrastructureConfig)
				}, aws.GetControlPlaneConfig)
			}
			return getConfigForProvider(runtimeShoot, aws.GetInfrastructureConfig, aws.GetControlPlaneConfig)
		}
	case hyperscaler.TypeAzure:
		{
			if existingInfrastructureConfig != nil {
				return getConfigForProvider(runtimeShoot, func(workersCidr string, zones []string) ([]byte, error) {
					return azure.GetInfrastructureConfigForPatch(workersCidr, zones, existingInfrastructureConfig)
				}, azure.GetControlPlaneConfig)
			}
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

func getNetworkingZonesFromWorkers(workers []gardener.Worker) ([]string, error) {
	var zones []string

	if len(workers) == 0 {
		return nil, errors.New("no workers provided")
	}

	for _, worker := range workers {
		for _, zone := range worker.Zones {
			if !slices.Contains(zones, zone) {
				zones = append(zones, zone)
			}
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
func alignWorkersWithGardener(provider *gardener.Provider, existingWorkers []gardener.Worker) {
	existingWorkersMap := make(map[string]gardener.Worker)
	for _, existing := range existingWorkers {
		existingWorkersMap[existing.Name] = existing
	}

	for i := range provider.Workers {
		alignedWorker := &provider.Workers[i]

		if existing, found := existingWorkersMap[alignedWorker.Name]; found {
			if alignedWorker.UpdateStrategy == nil {
				alignedWorker.UpdateStrategy = existing.UpdateStrategy
			}

			alignWorkerZonesForExtension(alignedWorker, existing)
			alignWorkerMachineImageVersion(alignedWorker.Machine.Image, existing.Machine.Image)
		}
	}
}

func alignWorkerZonesForExtension(worker *gardener.Worker, existing gardener.Worker) {
	// first check if zones are the same
	if slices.Equal(worker.Zones, existing.Zones) {
		return
	}
	// if not, align zones with existing worker
	providedZones := make([]string, len(worker.Zones))
	copy(providedZones, worker.Zones)

	worker.Zones = existing.Zones
	// if some preexisting zones are missing in the new worker, they will be added anyway
	// if there are any zones that are not in the existing worker, append them at the end
	for _, zone := range providedZones {
		if !slices.Contains(worker.Zones, zone) {
			worker.Zones = append(worker.Zones, zone)
		}
	}
}

// If the current image version with the same name on Shoot is greater than the version, it sets the version to the current machine image version.
func alignWorkerMachineImageVersion(workerImage *gardener.ShootMachineImage, shootWorkerImage *gardener.ShootMachineImage) {
	if shootWorkerImage == nil || workerImage == nil || workerImage.Name != shootWorkerImage.Name {
		return
	}

	if shootWorkerImage.Version == nil || *shootWorkerImage.Version == *workerImage.Version {
		return
	}

	if result, err := extender.CompareVersions(*workerImage.Version, *shootWorkerImage.Version); err == nil && result < 0 {
		workerImage.Version = shootWorkerImage.Version
	}
}
