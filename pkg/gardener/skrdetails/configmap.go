package skrdetails

import (
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"
)

type KymaProvisioningInfo struct {
	WorkerPools           WorkerPools          `json:"workerPools"`
	GlobalAccountID       string               `json:"globalAccountID,omitzero"`
	SubaccountID          string               `json:"subaccountID,omitzero"`
	EnvironmentInstanceID string               `json:"environmentInstanceID,omitzero"`
	InstanceName          string               `json:"instanceName,omitzero"`
	InfrastructureConfig  runtime.RawExtension `json:"infrastructureConfig,omitzero"`
	NetworkDetails        NetworkDetails       `json:"networkDetails"`
}

type WorkerPools struct {
	Kyma   WorkerPool   `json:"kyma"`
	Custom []WorkerPool `json:"custom,omitzero"`
}
type WorkerPool struct {
	Name                  string `json:"name"`
	MachineType           string `json:"machineType"`
	HighAvailabilityZones bool   `json:"haZones"`
	AutoScalerMin         int32  `json:"autoScalerMin"`
	AutoScalerMax         int32  `json:"autoScalerMax"`
}

type NetworkDetails struct {
	DualStackIPEnabled bool `json:"dualStackIPEnabled"`
}

func ToKymaProvisioningInfo(runtime imv1.Runtime, shoot *gardener.Shoot) KymaProvisioningInfo {
	var kymaWorkerPool WorkerPool
	var customWorkerPools []WorkerPool

	// There is an existing check if number of workers != 1 in pkg/gardener/shoot/extender/provider/provider.go:27
	// that will be later moved to validator webhook
	if len(runtime.Spec.Shoot.Provider.Workers) > 0 {
		mainRuntimeCRWorker := runtime.Spec.Shoot.Provider.Workers[0]
		kymaWorkerPool = WorkerPool{
			Name:                  mainRuntimeCRWorker.Name,
			MachineType:           mainRuntimeCRWorker.Machine.Type,
			HighAvailabilityZones: IsHighAvailability(mainRuntimeCRWorker.Zones),
			AutoScalerMin:         mainRuntimeCRWorker.Minimum,
			AutoScalerMax:         mainRuntimeCRWorker.Maximum,
		}
	}

	additionalWorkers := runtime.Spec.Shoot.Provider.AdditionalWorkers

	if additionalWorkers != nil {
		for _, worker := range *additionalWorkers {
			customWorkerPools = append(customWorkerPools, WorkerPool{
				Name:                  worker.Name,
				MachineType:           worker.Machine.Type,
				HighAvailabilityZones: IsHighAvailability(worker.Zones),
				AutoScalerMin:         worker.Minimum,
				AutoScalerMax:         worker.Maximum,
			})
		}
	}

	return KymaProvisioningInfo{
		WorkerPools: WorkerPools{
			Kyma:   kymaWorkerPool,
			Custom: customWorkerPools,
		},
		GlobalAccountID:       runtime.Labels[imv1.LabelKymaGlobalAccountID],
		SubaccountID:          runtime.Labels[imv1.LabelKymaSubaccountID],
		EnvironmentInstanceID: runtime.Labels[imv1.LabelKymaInstanceID],
		InstanceName:          runtime.Labels[imv1.LabelKymaName],
		InfrastructureConfig:  *shoot.Spec.Provider.InfrastructureConfig,
		NetworkDetails: NetworkDetails{
			DualStackIPEnabled: IsDualStackEnabled(shoot),
		},
	}
}

func ToKymaProvisioningInfoConfigMap(runtime imv1.Runtime, shoot *gardener.Shoot) (v1.ConfigMap, error) {
	details := ToKymaProvisioningInfo(runtime, shoot)
	authConfigBytes, err := yaml.Marshal(details)

	if err != nil {
		return v1.ConfigMap{}, err
	}

	return v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kyma-provisioning-info",
			Namespace: "kyma-system",
			Labels: map[string]string{
				"reconciler.kyma-project.io/managed-by": "infrastructure-manager",
			},
		},
		Data: map[string]string{
			"details": string(authConfigBytes),
		},
	}, nil
}
