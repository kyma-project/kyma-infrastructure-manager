package extensions

import (
	"encoding/json"
	"fmt"
	"os"
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

func loadIPsFromFile(kcpIpPath string, operatorIPPath string) (operatorIPs []string, kcpIp string, err error) {
	loadIPs := func(path string, ips any) error {
		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("opening %s: %w", path, err)
		}

		defer func() {
			_ = f.Close()
		}()

		if err := json.NewDecoder(f).Decode(ips); err != nil {
			return fmt.Errorf("decoding %s: %w", path, err)
		}
		return nil
	}

	err = loadIPs(operatorIPPath, &operatorIPs)
	if err != nil {
		return nil, "", err
	}

	err = loadIPs(kcpIpPath, &kcpIp)
	if err != nil {
		return nil, "", err
	}

	return operatorIPs, kcpIp, nil
}

func aclNeedsToBeEnabled(apiServerAclEnabled bool, runtime imv1.Runtime) bool {
	runtimeType := runtime.Spec.Shoot.Provider.Type

	return apiServerAclEnabled &&
		(runtimeType == hyperscaler.TypeAWS || runtimeType == hyperscaler.TypeAzure) &&
		runtime.Spec.Shoot.Kubernetes.KubeAPIServer.ACL != nil &&
		len(runtime.Spec.Shoot.Kubernetes.KubeAPIServer.ACL.AllowedCIDRs) > 0
}
