package extensions

import gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"

const NetworkFilterType = "shoot-networking-filter"

func NewNetworkFilterExtension(disabled bool) (*gardener.Extension, error) {
	return &gardener.Extension{
		Type:     NetworkFilterType,
		Disabled: &disabled,
	}, nil
}
