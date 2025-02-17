package extender

import (
	"encoding/json"
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	apimachineryruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
)

const NetworkFilterType = "shoot-networking-filter"

func ExtendWithNetworkFilter(runtime imv1.Runtime, shoot *gardener.Shoot) error { //nolint:revive
	disabled := !runtime.Spec.Security.Networking.Filter.Egress.Enabled


	filterProviderConfig := EgressFilterProviderConfig{
		BlackholingEnabled: egress.Enabled,
		Workers:            egress.Workers,
	}

	extensionJSON, err := json.Marshal(filterProviderConfig)
	if err != nil {
		return nil, err
	}

	networkingFilter := gardener.Extension{
		Type:     NetworkFilterType,
		Disabled: &disabled,
		ProviderConfig: &apimachineryruntime.RawExtension{
			Raw: extensionJSON,
		},
	}

	shoot.Spec.Extensions = append(shoot.Spec.Extensions, networkingFilter)

	return nil
}
