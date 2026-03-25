package extensions

import (
	"encoding/json"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
)

const ApiServerACLExtensionType string = "acl"

type AclList struct {
	OperatorIPs []string
	KCPIp       string
}

func NewApiServerACLExtension(operatorIPs []string, kcpIP string) (*gardener.Extension, error) {
	rawExtension, err := json.Marshal("test")

	return &gardener.Extension{
		Type:           "acl",
		ProviderConfig: &runtime.RawExtension{Raw: rawExtension},
		Disabled:       ptr.To(false),
	}, err
}
