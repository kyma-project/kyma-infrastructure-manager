package extensions

import (
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"k8s.io/utils/ptr"
)

const (
	OidcExtensionType = "shoot-oidc-service"
)

func NewOIDCExtension() (gardener.Extension, error) {
	return gardener.Extension{
		Type:     OidcExtensionType,
		Disabled: ptr.To(false),
	}, nil
}
