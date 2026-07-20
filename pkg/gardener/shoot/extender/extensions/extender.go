package extensions

import (
	"context"
	"encoding/json"
	"reflect"
	"slices"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/config"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/auditlogs"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CreateExtensionFunc func(runtime imv1.Runtime, shoot gardener.Shoot) (*gardener.Extension, error)

type Strategy int

const (
	StrategyKeep Strategy = iota
	StrategyRemove
)

type Extension struct {
	Type     string
	Create   CreateExtensionFunc
	Strategy Strategy
}

func NewExtensionsExtenderForCreate(ctx context.Context, kcpClient client.Client, config config.ConverterConfig, auditLogData auditlogs.AuditLogData, apiServerAclEnabled, networkingRestrictionsGlobalEnabled bool) func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
	return newExtensionsExtender([]Extension{
		{
			Type: NetworkFilterType,
			Create: func(runtime imv1.Runtime, _ gardener.Shoot) (*gardener.Extension, error) {
				if !networkingRestrictionsGlobalEnabled {
					return nil, nil
				}
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
				return buildDNSExtension(config.DNS, shoot.Name)
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
			Type: ApiServerACLExtensionType,
			Create: func(runtime imv1.Runtime, shoot gardener.Shoot) (*gardener.Extension, error) {
				if !AclNeedsToBeEnabled(apiServerAclEnabled, runtime) {
					return nil, nil
				}

				operatorIPs, kcpIPs, err := loadIPsFromConfigMap(ctx, kcpClient, config.Kubernetes.KubeApiServer.ACL.ConfigMapName)
				if err != nil {
					return nil, err
				}
				return NewApiServerACLExtension(runtime.Spec.Shoot.Kubernetes.KubeAPIServer.ACL.AllowedCIDRs, operatorIPs, kcpIPs)
			},
		},
		{
			Type: NvidiaOpenshellExtensionType,
			Create: func(runtime imv1.Runtime, shoot gardener.Shoot) (*gardener.Extension, error) {
				if !isNvidiaOpenshellEnabled(runtime) {
					return nil, nil
				}

				return EnableNvidiaOpenshellExtension()
			},
		},
	}, nil)
}

func NewExtensionsExtenderForPatch(ctx context.Context, kcpClient client.Client, config config.ConverterConfig, auditLogData auditlogs.AuditLogData, extensionsOnTheShoot []gardener.Extension, apiServerAclEnabled, networkingRestrictionsGlobalEnabled bool, registryCacheGardenSecretNames map[string]string) func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
	return newExtensionsExtender([]Extension{
		{
			Type: AuditlogExtensionType,
			Create: func(_ imv1.Runtime, shoot gardener.Shoot) (*gardener.Extension, error) {

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
				if !networkingRestrictionsGlobalEnabled {
					return nil, nil
				}
				return NewNetworkFilterExtension(runtime.Spec.Security.Networking.Filter)
			},
		},
		{
			Type: DNSExtensionType,
			Create: func(_ imv1.Runtime, shoot gardener.Shoot) (*gardener.Extension, error) {
				return dnsExtensionForPatch(config.DNS, shoot)
			},
		},
		{
			Type: RegistryCacheExtensionType,
			Create: func(runtime imv1.Runtime, shoot gardener.Shoot) (*gardener.Extension, error) {
				return NewRegistryCacheExtension(runtime.Spec.Caching, registryCacheGardenSecretNames, existingExtension(RegistryCacheExtensionType, shoot))
			},
			Strategy: StrategyRemove,
		},
		{
			Type: ApiServerACLExtensionType,
			Create: func(runtime imv1.Runtime, shoot gardener.Shoot) (*gardener.Extension, error) {
				if !AclNeedsToBeEnabled(apiServerAclEnabled, runtime) {
					if existingExtension(ApiServerACLExtensionType, shoot) == nil {
						return nil, nil
					}

					return NewApiServerACLExtension(nil, nil, "")
				}

				operatorIPs, kcpIPs, err := loadIPsFromConfigMap(ctx, kcpClient, config.Kubernetes.KubeApiServer.ACL.ConfigMapName)
				if err != nil {
					return nil, err
				}

				return NewApiServerACLExtension(runtime.Spec.Shoot.Kubernetes.KubeAPIServer.ACL.AllowedCIDRs, operatorIPs, kcpIPs)
			},
		},
		{
			Type: NvidiaOpenshellExtensionType,
			Create: func(runtime imv1.Runtime, shoot gardener.Shoot) (*gardener.Extension, error) {
				if isNvidiaOpenshellEnabled(runtime) {
					return EnableNvidiaOpenshellExtension()
				}

				if existingExtension(NvidiaOpenshellExtensionType, shoot) == nil {
					return nil, nil
				}

				return DisableNvidiaOpenshellExtension()
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
			updatedGardenerExtension, err := ext.Create(runtime, *shoot)
			if err != nil {
				return err
			}

			index := slices.IndexFunc(shoot.Spec.Extensions, func(e gardener.Extension) bool {
				return e.Type == ext.Type
			})

			if index == -1 {
				if updatedGardenerExtension != nil {
					shoot.Spec.Extensions = append(shoot.Spec.Extensions, *updatedGardenerExtension)
				}
			} else {
				if updatedGardenerExtension != nil {
					shoot.Spec.Extensions[index] = *updatedGardenerExtension
				} else if ext.Strategy == StrategyRemove {
					shoot.Spec.Extensions = slices.Delete(shoot.Spec.Extensions, index, index+1)
				}
			}
		}

		return nil
	}
}

func existingExtension(extensionType string, shoot gardener.Shoot) *gardener.Extension {
	for _, ext := range shoot.Spec.Extensions {
		if ext.Type == extensionType {
			return &ext
		}
	}
	return nil
}

func isNvidiaOpenshellEnabled(runtime imv1.Runtime) bool {
	return runtime.Spec.Shoot.EnableNvidiaOpenshell != nil && *runtime.Spec.Shoot.EnableNvidiaOpenshell
}

func buildDNSExtension(dnsConfig config.DNSConfig, shootName string) (*gardener.Extension, error) {
	if dnsConfig.IsGardenerInternal() {
		return NewDNSExtensionInternal()
	}
	return NewDNSExtensionExternal(shootName, dnsConfig.SecretName, dnsConfig.DomainPrefix, dnsConfig.ProviderType)
}

func dnsExtensionForPatch(dnsConfig config.DNSConfig, shoot gardener.Shoot) (*gardener.Extension, error) {
	desired, err := buildDNSExtension(dnsConfig, shoot.Name)
	if err != nil {
		return nil, err
	}

	existing := existingExtension(DNSExtensionType, shoot)
	if existing == nil || existing.ProviderConfig == nil {
		return desired, nil
	}

	var existingCfg DNSExtensionProviderConfig
	if err := json.Unmarshal(existing.ProviderConfig.Raw, &existingCfg); err != nil {
		return nil, err
	}

	var desiredCfg DNSExtensionProviderConfig
	if err := json.Unmarshal(desired.ProviderConfig.Raw, &desiredCfg); err != nil {
		return nil, err
	}

	// The cluster was created prior to introducing custom domains for the DNS extension. In this case, we want to preserve the existing configuration
	if len(existingCfg.Providers) == 0 {
		return existing, nil
	}

	if reflect.DeepEqual(existingCfg, desiredCfg) {
		return nil, nil
	}
	return desired, nil
}
