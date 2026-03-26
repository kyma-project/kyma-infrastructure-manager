package extensions

import (
	"encoding/json"
	"slices"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
)

const ApiServerACLExtensionType string = "acl"

type aclProviderConfig struct {
	Rule aclRule `json:"rule"`
}

type aclRule struct {
	Action string   `json:"action"`
	Cidrs  []string `json:"cidrs"`
	Type   string   `json:"type"`
}

func NewApiServerACLExtension(userIPs, operatorIPs []string, kcpIP string) (*gardener.Extension, error) {
	if len(userIPs) != 0 {
		return applyAccessControlList(slices.Concat(userIPs, operatorIPs, []string{kcpIP}))
	}

	return &gardener.Extension{
		Type:     ApiServerACLExtensionType,
		Disabled: ptr.To(true),
	}, nil
}

func applyAccessControlList(aclList []string) (*gardener.Extension, error) {
	aclExtension := aclProviderConfig{Rule: aclRule{Action: "ALLOW", Cidrs: aclList, Type: "remote_ip"}}
	rawExtension, encodingErr := json.Marshal(aclExtension)
	if encodingErr != nil {
		return nil, encodingErr
	}

	return &gardener.Extension{
		Type:           ApiServerACLExtensionType,
		ProviderConfig: &runtime.RawExtension{Raw: rawExtension},
		Disabled:       ptr.To(false),
	}, nil
}
