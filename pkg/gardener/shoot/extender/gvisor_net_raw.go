package extender

import (
	"encoding/json"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gvisorv1alpha1 "github.com/gardener/gardener-extension-runtime-gvisor/pkg/apis/config/v1alpha1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	gvisorContainerRuntimeType = "gvisor"
	gvisorNetRawConfigKey      = "net-raw"
	gvisorNetRawDefaultValue   = "true"
	gvisorProviderConfigAPIVer = "gvisor.runtime.extensions.config.gardener.cloud/v1alpha1"
	gvisorProviderConfigKind   = "GVisorConfiguration"
)

// ExtendWithGVisorNetRawDefault sets configFlags["net-raw"] = "true" on each gVisor container runtime
// when the Runtime CR does not define that key (required for Istio). If net-raw is present,
// the Runtime CR value is kept.
func ExtendWithGVisorNetRawDefault(_ imv1.Runtime, shoot *gardener.Shoot) error {
	return applyDefaultGVisorNetRaw(shoot.Spec.Provider.Workers)
}

func applyDefaultGVisorNetRaw(workers []gardener.Worker) error {
	for i := range workers {
		if workers[i].CRI == nil {
			continue
		}
		for j := range workers[i].CRI.ContainerRuntimes {
			if workers[i].CRI.ContainerRuntimes[j].Type != gvisorContainerRuntimeType {
				continue
			}
			pc, err := ensureGVisorNetRawDefault(workers[i].CRI.ContainerRuntimes[j].ProviderConfig)
			if err != nil {
				return err
			}
			workers[i].CRI.ContainerRuntimes[j].ProviderConfig = pc
		}
	}
	return nil
}

func ensureGVisorNetRawDefault(pc *runtime.RawExtension) (*runtime.RawExtension, error) {
	if pc == nil || len(pc.Raw) == 0 {
		return newGVisorProviderConfigWithNetRawDefault()
	}

	var config gvisorv1alpha1.GVisorConfiguration
	if err := json.Unmarshal(pc.Raw, &config); err != nil {
		return nil, errors.Wrap(err, "unmarshal gVisor providerConfig failed")
	}

	if config.ConfigFlags == nil {
		config.APIVersion = gvisorProviderConfigAPIVer
		config.Kind = gvisorProviderConfigKind
		flags := map[string]string{
			gvisorNetRawConfigKey: gvisorNetRawDefaultValue,
		}
		config.ConfigFlags = &flags
	} else if _, has := (*config.ConfigFlags)[gvisorNetRawConfigKey]; !has {
		(*config.ConfigFlags)[gvisorNetRawConfigKey] = gvisorNetRawDefaultValue
	}

	raw, err := json.Marshal(config)
	if err != nil {
		return nil, errors.Wrap(err, "marshal gVisor providerConfig")
	}
	return &runtime.RawExtension{Raw: raw}, nil
}

func newGVisorProviderConfigWithNetRawDefault() (*runtime.RawExtension, error) {
	flags := map[string]string{
		gvisorNetRawConfigKey: gvisorNetRawDefaultValue,
	}
	config := gvisorv1alpha1.GVisorConfiguration{
		ConfigFlags: &flags,
	}
	config.APIVersion = gvisorProviderConfigAPIVer
	config.Kind = gvisorProviderConfigKind

	raw, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}
	return &runtime.RawExtension{Raw: raw}, nil
}
