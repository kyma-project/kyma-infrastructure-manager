package extensions

import (
	"encoding/json"
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apimachineryRuntime "k8s.io/apimachinery/pkg/runtime"
)

const NetworkFilterType = "shoot-networking-filter"

func NewNetworkFilterExtension(filter imv1.Filter) (*gardener.Extension, error) {
	disabled := isNetworkingFilterDisabled(filter)

	networkingFilterExtension := gardener.Extension{
		Type:     NetworkFilterType,
		Disabled: &disabled,
	}
 if disabled {
    return return &networkingFilterExtension, nil
  }
	if isIngressBlackholingEnabled(filter) {
		ingressFilterConfig := filter.Ingress

		filterProviderConfig := Configuration{
			TypeMeta: metav1.TypeMeta{},
			EgressFilter: &EgressFilter{
				BlackholingEnabled: ingressFilterConfig.Enabled,
			},
		}

		providerJson, encodingErr := json.Marshal(filterProviderConfig)
		if encodingErr != nil {
			return nil, encodingErr
		}

		networkingFilterExtension.ProviderConfig = &apimachineryRuntime.RawExtension{Raw: providerJson}
	}

	return &networkingFilterExtension, nil
}

func isNetworkingFilterDisabled(filter imv1.Filter) bool {
	return !filter.Egress.Enabled
}

func isIngressBlackholingEnabled(filter imv1.Filter) bool {
	return filter.Ingress != nil && filter.Ingress.Enabled
}

// Configuration represents `RawExtension` we want to set as `ProviderConfig` in the `gardener.Extension`
// copied partially from https://github.com/gardener/gardener-extension-shoot-networking-filter/blob/master/pkg/apis/config/v1alpha1/types.go#L16C1-L26C2
type Configuration struct {
	metav1.TypeMeta `json:",inline"`

	// EgressFilter contains the configuration for the egress filter
	// +optional
	EgressFilter *EgressFilter `json:"egressFilter,omitempty"`
}

// EgressFilter contains the configuration for the egress filter.
// copied and adjusted from https://github.com/gardener/gardener-extension-shoot-networking-filter/blob/master/pkg/apis/config/v1alpha1/types.go#L29
type EgressFilter struct {
	// BlackholingEnabled is a flag to set blackholing or firewall approach.
	BlackholingEnabled bool `json:"blackholingEnabled"`
}

// Policy is the access policy
// copied partially from https://github.com/gardener/gardener-extension-shoot-networking-filter/blob/master/pkg/apis/config/types.go#L62
type Policy string

const (
	// PolicyBlockAccess is the `BLOCK_ACCESS` policy
	PolicyBlockAccess Policy = "BLOCK_ACCESS"
)
