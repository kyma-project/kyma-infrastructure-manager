package extender

import (
	"encoding/json"
	"io"
	"os"
	"path"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/config"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/hyperscaler"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
)

type ACLList struct {
	OperatorIPs []string
	KCPIp       string
}

type aclProviderConfig struct {
	Rule aclRule `json:"rule"`
}

type aclRule struct {
	Action string   `json:"action"`
	Cidrs  []string `json:"cidrs"`
	Type   string   `json:"type"`
}

func NewKubeServerACLExtenderCreate(aclConfig config.ACL) func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
	return func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
		if !aclConfig.EnableACL {
			return nil
		}

		runtimeType := runtime.Spec.Shoot.Provider.Type
		if runtimeType != hyperscaler.TypeAWS && runtimeType != hyperscaler.TypeAzure {
			return nil
		}

		if runtime.Spec.Shoot.Kubernetes.KubeAPIServer.ACL == nil {
			return nil
		}

		if len(runtime.Spec.Shoot.Kubernetes.KubeAPIServer.ACL.AllowedCIDRs) == 0 {
			return nil
		}

		acl, err := createAccessControlList(
			runtime.Spec.Shoot.Kubernetes.KubeAPIServer.ACL.AllowedCIDRs,
			aclConfig.VolumeMountPath,
			aclConfig.IpAddressesKey,
			aclConfig.KcpAddressKey)
		if err != nil {
			// there was an error during os opening, file not existing
			return err
		}

		err = applyAccessControlList(shoot, acl)
		if err != nil {
			return err
		}
		return nil
	}
}

func NewKubeServerACLExtenderPatch(aclConfig config.ACL) func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
	return func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
		if !aclConfig.EnableACL {
			return nil
		}

		runtimeType := runtime.Spec.Shoot.Provider.Type
		if runtimeType != hyperscaler.TypeAWS && runtimeType != hyperscaler.TypeAzure {
			return nil
		}

		aclNilOrEmpty := runtime.Spec.Shoot.Kubernetes.KubeAPIServer.ACL == nil ||
			len(runtime.Spec.Shoot.Kubernetes.KubeAPIServer.ACL.AllowedCIDRs) == 0

		if aclNilOrEmpty {
			if checkIfACLExists(*shoot) {
				removeAccessControlList(shoot)
			}
			return nil
		}

		return applyAccessControlList(shoot, runtime.Spec.Shoot.Kubernetes.KubeAPIServer.ACL.AllowedCIDRs)
	}
}

func applyAccessControlList(shoot *gardener.Shoot, aclList []string) error {
	aclExtension := aclProviderConfig{Rule: aclRule{Action: "ALLOW", Cidrs: aclList, Type: "remote_ip"}}
	rawExtension, encodingErr := json.Marshal(aclExtension)
	if encodingErr != nil {
		return encodingErr
	}
	aclExt := gardener.Extension{
		Type:           "acl",
		ProviderConfig: &runtime.RawExtension{Raw: rawExtension},
		Disabled:       ptr.To(false),
	}
	shoot.Spec.Extensions = append(shoot.Spec.Extensions, aclExt)
	return nil
}

func createAccessControlList(userCIDRs []string, volumeMountPath, ipKey, kcpKey string) ([]string, error) {
	aclList := ACLList{}
	var allowedCIDRs []string

	operatorPath := path.Join(volumeMountPath, ipKey)
	err := aclList.loadOperatorData(func() (io.Reader, error) {
		return os.Open(operatorPath)
	})
	if err != nil {
		return nil, err
	}

	kcpPath := path.Join(volumeMountPath, kcpKey)
	err = aclList.loadKcpData(func() (io.Reader, error) {
		return os.Open(kcpPath)
	})
	if err != nil {
		return nil, err
	}

	allowedCIDRs = append(allowedCIDRs, userCIDRs...)
	allowedCIDRs = append(allowedCIDRs, aclList.OperatorIPs...)
	allowedCIDRs = append(allowedCIDRs, aclList.KCPIp)

	return allowedCIDRs, nil
}

func removeAccessControlList(shoot *gardener.Shoot) {
	extensions := make([]gardener.Extension, 0, len(shoot.Spec.Extensions))
	for _, ext := range shoot.Spec.Extensions {
		if ext.Type != "acl" {
			extensions = append(extensions, ext)
		}
	}
	shoot.Spec.Extensions = extensions
}

func checkIfACLExists(shoot gardener.Shoot) bool {
	if shoot.Spec.Extensions == nil {
		return false
	}

	for _, ext := range shoot.Spec.Extensions {
		if ext.Type == "acl" {
			return true
		}
	}
	return false
}

type readerGetter = func() (io.Reader, error)

func (ac *ACLList) loadOperatorData(f readerGetter) error {
	r, err := f()
	if err != nil {
		return err
	}
	if closer, ok := r.(io.Closer); ok {
		defer closer.Close()
	}
	return json.NewDecoder(r).Decode(&ac.OperatorIPs)
}

func (ac *ACLList) loadKcpData(f readerGetter) error {
	r, err := f()
	if err != nil {
		return err
	}
	if closer, ok := r.(io.Closer); ok {
		defer closer.Close()
	}
	return json.NewDecoder(r).Decode(&ac.KCPIp)
}
