package networking

import (
	"encoding/json"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	hyperscaler2 "github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/hyperscaler"
	"k8s.io/apimachinery/pkg/runtime"
)

type VXLanConfig struct {
	ApiVersion string  `json:"apiVersion"`
	Kind       string  `json:"kind"`
	VXlan      VXLan   `json:"vxlan"`
	Overlay    Overlay `json:"overlay"`
}

type VXLan struct {
	Enabled bool `json:"enabled"`
}

type Overlay struct {
	Enabled bool `json:"enabled"`
}

func ExtendWithNetworking(infraSupportsDualStack bool) func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
	return func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
		if canEnableDualStackIPs(runtime.Spec.Shoot.Provider.Type, runtime.Spec.Shoot.Networking.DualStack) && infraSupportsDualStack {
			extendWithDualIPs(shoot)
		}
		if runtime.Spec.Shoot.Provider.Type == hyperscaler2.TypeGDCH {
			extendWithVXLan(shoot)
		}
		// if other provider is used, Gardener by default configures IPv4 only, so no action is needed
		return nil
	}
}

func canEnableDualStackIPs(providerType string, dualStackForRuntime *bool) bool {
	if dualStackForRuntime == nil || !*dualStackForRuntime {
		return false
	}
	return providerType == hyperscaler2.TypeGCP || providerType == hyperscaler2.TypeAWS
}

func extendWithDualIPs(shoot *gardener.Shoot) {
	if shoot.Spec.Networking == nil {
		shoot.Spec.Networking = &gardener.Networking{
			IPFamilies: []gardener.IPFamily{gardener.IPFamilyIPv4, gardener.IPFamilyIPv6},
		}
	} else {
		shoot.Spec.Networking.IPFamilies = []gardener.IPFamily{gardener.IPFamilyIPv4, gardener.IPFamilyIPv6}
	}
}

func extendWithVXLan(shoot *gardener.Shoot) {
	if shoot.Spec.Networking == nil {
		shoot.Spec.Networking = &gardener.Networking{
			ProviderConfig: networkingConfig(),
		}
	} else {
		shoot.Spec.Networking.ProviderConfig = networkingConfig()
	}
}

func networkingConfig() *runtime.RawExtension {
	cfg := VXLanConfig{
		ApiVersion: "calico.networking.extensions.gardener.cloud/v1alpha1",
		Kind:       "NetworkConfig",
		VXlan: VXLan{
			Enabled: true,
		},
		Overlay: Overlay{
			Enabled: true,
		},
	}

	providerJson, encodingErr := json.Marshal(cfg)
	if encodingErr != nil {
		return nil
	}
	return &runtime.RawExtension{Raw: providerJson}
}
