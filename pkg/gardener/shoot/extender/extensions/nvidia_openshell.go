package extensions

import (
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"k8s.io/utils/ptr"
)

const (
	NvidiaOpenshellExtensionType = "shoot-nvidia-openshell"
)

func NewNvidiaOpenshellExtension() (*gardener.Extension, error) {
	return &gardener.Extension{
		Type:     NvidiaOpenshellExtensionType,
		Disabled: ptr.To(false),
	}, nil
}
