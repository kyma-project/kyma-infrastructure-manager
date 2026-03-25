package extensions

import (
	"encoding/json"
	"io"
	"os"
	"slices"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/config"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/auditlogs"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/hyperscaler"
)

type CreateExtensionFunc func(runtime imv1.Runtime, shoot gardener.Shoot) (*gardener.Extension, error)

type Extension struct {
	Type   string
	Create CreateExtensionFunc
}

func NewExtensionsExtenderForCreate(config config.ConverterConfig, auditLogData auditlogs.AuditLogData, registryCache []imv1.ImageRegistryCache, apiServerAclEnabled bool) func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
	return newExtensionsExtender([]Extension{
		{
			Type: NetworkFilterType,
			Create: func(runtime imv1.Runtime, _ gardener.Shoot) (*gardener.Extension, error) {
				return NewNetworkFilterExtension(runtime.Spec.Security.Networking.Filter)
			},
		},
		{
			Type: CertExtensionType,
			Create: func(_ imv1.Runtime, _ gardener.Shoot) (*gardener.Extension, error) {
				return NewCertExtension()
			},
		},
		{
			Type: DNSExtensionType,
			Create: func(_ imv1.Runtime, shoot gardener.Shoot) (*gardener.Extension, error) {
				if config.DNS.IsGardenerInternal() {
					return NewDNSExtensionInternal()
				}
				return NewDNSExtensionExternal(shoot.Name, config.DNS.SecretName, config.DNS.DomainPrefix, config.DNS.ProviderType)
			},
		},
		{
			Type: OidcExtensionType,
			Create: func(_ imv1.Runtime, _ gardener.Shoot) (*gardener.Extension, error) {
				return NewOIDCExtension()
			},
		},
		{
			Type: AuditlogExtensionType,
			Create: func(_ imv1.Runtime, _ gardener.Shoot) (*gardener.Extension, error) {
				if auditLogData == (auditlogs.AuditLogData{}) {
					return nil, nil
				}

				return NewAuditLogExtension(auditLogData)
			},
		},
		{
			Type: RegistryCacheExtensionType,
			Create: func(runtime imv1.Runtime, shoot gardener.Shoot) (*gardener.Extension, error) {
				if len(registryCache) == 0 {
					return nil, nil
				}

				return NewRegistryCacheExtension(registryCache, nil)
			},
		},
		{
			Type: ApiServerACLExtensionType,
			Create: func(runtime imv1.Runtime, shoot gardener.Shoot) (*gardener.Extension, error) {
				if !aclNeedToBeEnabled(apiServerAclEnabled, runtime) {
					return nil, nil
				}

				aclList := AclList{}

				err := aclList.loadOperatorData(func() (io.Reader, error) {
					return os.Open(config.Kubernetes.KubeApiServer.ACL.IpAddressesPath)
				})
				if err != nil {
					return nil, err
				}

				err = aclList.loadKcpData(func() (io.Reader, error) {
					return os.Open(config.Kubernetes.KubeApiServer.ACL.KcpAddressPath)
				})
				if err != nil {
					return nil, err
				}

				return NewApiServerACLExtension(runtime.Spec.Shoot.Kubernetes.KubeAPIServer.ACL.AllowedCIDRs, aclList.OperatorIPs, aclList.KCPIp)
			},
		},
	}, nil)
}

func NewExtensionsExtenderForPatch(config config.ConverterConfig, auditLogData auditlogs.AuditLogData, extensionsOnTheShoot []gardener.Extension, apiServerAclEnabled bool) func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
	return newExtensionsExtender([]Extension{
		{
			AuditlogExtensionType,
			func(_ imv1.Runtime, shoot gardener.Shoot) (*gardener.Extension, error) {
				if auditLogData == (auditlogs.AuditLogData{}) {
					return nil, nil
				}

				newAuditLogExtension, err := NewAuditLogExtension(auditLogData)
				if err != nil {
					return nil, err
				}

				auditLogIndex := slices.IndexFunc(shoot.Spec.Extensions, func(e gardener.Extension) bool {
					return e.Type == AuditlogExtensionType
				})

				if auditLogIndex == -1 {
					return newAuditLogExtension, nil
				}
				var existingAuditLogConfig AuditlogExtensionConfig
				if err := json.Unmarshal(shoot.Spec.Extensions[auditLogIndex].ProviderConfig.Raw, &existingAuditLogConfig); err != nil {
					return nil, err
				}

				var newAuditLogConfig AuditlogExtensionConfig
				if err := json.Unmarshal(newAuditLogExtension.ProviderConfig.Raw, &newAuditLogConfig); err != nil {
					return nil, err
				}

				if newAuditLogConfig != existingAuditLogConfig {
					return newAuditLogExtension, nil
				}

				return nil, nil
			},
		},
		{
			Type: NetworkFilterType,
			Create: func(runtime imv1.Runtime, _ gardener.Shoot) (*gardener.Extension, error) {
				return NewNetworkFilterExtension(runtime.Spec.Security.Networking.Filter)
			},
		},
		{
			Type: RegistryCacheExtensionType,
			Create: func(runtime imv1.Runtime, shoot gardener.Shoot) (*gardener.Extension, error) {
				var existingExtension *gardener.Extension

				for _, ext := range shoot.Spec.Extensions {
					if ext.Type == RegistryCacheExtensionType {
						existingExtension = &ext
						break
					}
				}

				return NewRegistryCacheExtension(runtime.Spec.Caching, existingExtension)
			},
		},
		{
			Type: ApiServerACLExtensionType,
			Create: func(runtime imv1.Runtime, shoot gardener.Shoot) (*gardener.Extension, error) {
				if !aclNeedToBeEnabled(apiServerAclEnabled, runtime) {
					if existingExtension(ApiServerACLExtensionType, shoot) == nil {
						return nil, nil
					}

					return NewApiServerACLExtension(nil, nil, "")
				}
				aclList := AclList{}

				err := aclList.loadOperatorData(func() (io.Reader, error) {
					return os.Open(config.Kubernetes.KubeApiServer.ACL.IpAddressesPath)
				})
				if err != nil {
					return nil, err
				}

				err = aclList.loadKcpData(func() (io.Reader, error) {
					return os.Open(config.Kubernetes.KubeApiServer.ACL.KcpAddressPath)
				})
				if err != nil {
					return nil, err
				}

				return NewApiServerACLExtension(runtime.Spec.Shoot.Kubernetes.KubeAPIServer.ACL.AllowedCIDRs, aclList.OperatorIPs, aclList.KCPIp)
			},
		},
	}, extensionsOnTheShoot)
}

func newExtensionsExtender(extensionsToApply []Extension, currentGardenerExtensions []gardener.Extension) func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
	return func(runtime imv1.Runtime, shoot *gardener.Shoot) error {

		for _, currentExtension := range currentGardenerExtensions {
			shoot.Spec.Extensions = append(shoot.Spec.Extensions, *currentExtension.DeepCopy())
		}

		for _, ext := range extensionsToApply {
			gardenerExtension, err := ext.Create(runtime, *shoot)
			if err != nil {
				return err
			}

			// If the extension should not be applied we skip it
			if gardenerExtension == nil {
				continue
			}

			index := slices.IndexFunc(currentGardenerExtensions, func(e gardener.Extension) bool {
				return e.Type == ext.Type
			})

			if index == -1 {
				shoot.Spec.Extensions = append(shoot.Spec.Extensions, *gardenerExtension)
			} else {
				shoot.Spec.Extensions[index] = *gardenerExtension
			}
		}

		return nil
	}
}

type readerGetter = func() (io.Reader, error)

func (ac *AclList) loadOperatorData(f readerGetter) error {
	r, err := f()
	if err != nil {
		return err
	}
	if closer, ok := r.(io.Closer); ok {
		defer closer.Close()
	}
	return json.NewDecoder(r).Decode(&ac.OperatorIPs)
}

func (ac *AclList) loadKcpData(f readerGetter) error {
	r, err := f()
	if err != nil {
		return err
	}
	if closer, ok := r.(io.Closer); ok {
		defer closer.Close()
	}
	return json.NewDecoder(r).Decode(&ac.KCPIp)
}

func aclNeedToBeEnabled(apiServerAclEnabled bool, runtime imv1.Runtime) bool {
	//if !apiServerAclEnabled {
	//	return false
	//}
	//
	//runtimeType := runtime.Spec.Shoot.Provider.Type
	//if runtimeType =0= hyperscaler.TypeAWS && runtimeType != hyperscaler.TypeAzure {
	//	return false
	//}
	//
	//if runtime.Spec.Shoot.Kubernetes.KubeAPIServer.ACL == nil {
	//	return false
	//}
	//
	//if len(runtime.Spec.Shoot.Kubernetes.KubeAPIServer.ACL.AllowedCIDRs) == 0 {
	//	return false
	//}
	//
	//return true

	runtimeType := runtime.Spec.Shoot.Provider.Type

	return apiServerAclEnabled &&
		(runtimeType == hyperscaler.TypeAWS || runtimeType == hyperscaler.TypeAzure) &&
		runtime.Spec.Shoot.Kubernetes.KubeAPIServer.ACL != nil &&
		len(runtime.Spec.Shoot.Kubernetes.KubeAPIServer.ACL.AllowedCIDRs) > 0
}

func existingExtension(extensionType string, shoot gardener.Shoot) *gardener.Extension {
	for _, ext := range shoot.Spec.Extensions {
		if ext.Type == extensionType {
			return &ext
		}
	}
	return nil
}
