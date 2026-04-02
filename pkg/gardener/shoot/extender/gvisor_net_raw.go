package extender

import (
	"encoding/json"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
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

	var obj map[string]interface{}
	if err := json.Unmarshal(pc.Raw, &obj); err != nil {
		return nil, errors.Wrap(err, "unmarshal gVisor providerConfig failed")
	}

	flagsRaw, exists := obj["configFlags"]
	if !exists || flagsRaw == nil {
		if _, hasAPIVersion := obj["apiVersion"]; !hasAPIVersion {
			obj["apiVersion"] = gvisorProviderConfigAPIVer
		}
		if _, hasKind := obj["kind"]; !hasKind {
			obj["kind"] = gvisorProviderConfigKind
		}
		obj["configFlags"] = map[string]interface{}{
			gvisorNetRawConfigKey: gvisorNetRawDefaultValue,
		}

			gvisorNetRawConfigKey: gvisorNetRawDefaultValue,
		}
	} else {
		flags, ok := flagsRaw.(map[string]interface{})
		if !ok {
			return nil, errors.New("gVisor configFlags must be a JSON object")
		}
		if _, has := flags[gvisorNetRawConfigKey]; !has {
			flags[gvisorNetRawConfigKey] = gvisorNetRawDefaultValue
		}
	}

	raw, err := json.Marshal(obj)
	if err != nil {
		return nil, errors.Wrap(err, "marshal gVisor providerConfig")
	}
	return &runtime.RawExtension{Raw: raw}, nil
}

func newGVisorProviderConfigWithNetRawDefault() (*runtime.RawExtension, error) {
	obj := map[string]interface{}{
		"apiVersion": gvisorProviderConfigAPIVer,
		"kind":       gvisorProviderConfigKind,
		"configFlags": map[string]interface{}{
			gvisorNetRawConfigKey: gvisorNetRawDefaultValue,
		},
	}
	raw, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	return &runtime.RawExtension{Raw: raw}, nil
}
