package extensions

import (
	"encoding/json"
	"slices"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/hyperscaler"

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

func loadIPsFromConfigMap(aclMapName string) (operatorIPs []string, kcpIp string, err error) {
	return []string{"2.2.2.2/29", "3.3.3.3/29", "4.4.4.4/29"}, "1.1.1.1/32", nil
}

func aclNeedsToBeEnabled(apiServerAclEnabled bool, runtime imv1.Runtime) bool {
	runtimeType := runtime.Spec.Shoot.Provider.Type

	return apiServerAclEnabled &&
		(runtimeType == hyperscaler.TypeAWS || runtimeType == hyperscaler.TypeAzure) &&
		runtime.Spec.Shoot.Kubernetes.KubeAPIServer.ACL != nil &&
		len(runtime.Spec.Shoot.Kubernetes.KubeAPIServer.ACL.AllowedCIDRs) > 0
}
