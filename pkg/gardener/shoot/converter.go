package shoot

import (
	"fmt"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/config"
	extender2 "github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/auditlogs"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Extend func(imv1.Runtime, *gardener.Shoot) error

func baseExtenders(cfg config.ConverterConfig) []Extend {
	return []Extend{
		extender2.ExtendWithAnnotations,
		extender2.ExtendWithLabels,
		extender2.NewDNSExtender(cfg.DNS.SecretName, cfg.DNS.DomainPrefix, cfg.DNS.ProviderType),
		extender2.NewOidcExtender(cfg.Kubernetes.DefaultOperatorOidc),
		extender2.ExtendWithCloudProfile,
		extender2.ExtendWithNetworkFilter,
		extender2.ExtendWithCertConfig,
		extender2.ExtendWithExposureClassName,
		extender2.ExtendWithTolerations,
		extender2.NewMaintenanceExtender(cfg.Kubernetes.EnableKubernetesVersionAutoUpdate, cfg.Kubernetes.EnableMachineImageVersionAutoUpdate),
	}
}

type Converter struct {
	extenders []Extend
	config    config.ConverterConfig
}

func newConverter(config config.ConverterConfig, extenders ...Extend) Converter {
	return Converter{
		extenders: extenders,
		config:    config,
	}
}

type CreateOpts struct {
	config.ConverterConfig
	auditlogs.AuditLogData
}

type PatchOpts struct {
	config.ConverterConfig
	auditlogs.AuditLogData
	Zones             []string
	ShootK8SVersion   string
	ShootImageName    string
	ShootImageVersion string
}

func NewConverterCreate(opts CreateOpts) Converter {
	baseExtenders := baseExtenders(opts.ConverterConfig)

	baseExtenders = append(baseExtenders,
		extender2.NewProviderExtenderForCreateOperation(
			opts.Provider.AWS.EnableIMDSv2,
			opts.MachineImage.DefaultName,
			opts.MachineImage.DefaultVersion))

	baseExtenders = append(baseExtenders,
		extender2.NewKubernetesExtender(opts.Kubernetes.DefaultVersion, opts.Kubernetes.DefaultVersion))

	var zero auditlogs.AuditLogData
	if opts.AuditLogData == zero {
		return newConverter(opts.ConverterConfig, baseExtenders...)
	}

	baseExtenders = append(baseExtenders,
		auditlogs.NewAuditlogExtender(
			opts.AuditLog.PolicyConfigMapName,
			opts.AuditLogData))

	return newConverter(opts.ConverterConfig, baseExtenders...)
}

func NewConverterPatch(opts PatchOpts) Converter {
	baseExtenders := baseExtenders(opts.ConverterConfig)

	//k8SVersion, _ := selectKubernetesVersion(cfg.Kubernetes.DefaultVersion, k8sVersionFromShoot)
	//imageName, imageVersion, _ := selectImageVersion(cfg.MachineImage.DefaultName, cfg.MachineImage.DefaultVersion, imageNameFromShoot, imageVersionFromShoot)
	//kubernetesExtender := extender2.NewKubernetesExtender(k8SVersion)

	baseExtenders = append(baseExtenders,
		extender2.NewProviderExtenderPatchOperation(
			opts.Provider.AWS.EnableIMDSv2,
			opts.MachineImage.DefaultName,
			opts.MachineImage.DefaultVersion,
			opts.Zones))

	baseExtenders = append(baseExtenders,
		extender2.NewKubernetesExtender(opts.Kubernetes.DefaultVersion, opts.ShootK8SVersion))

	var zero auditlogs.AuditLogData
	if opts.AuditLogData == zero {
		return newConverter(opts.ConverterConfig, baseExtenders...)
	}

	baseExtenders = append(baseExtenders,
		auditlogs.NewAuditlogExtender(
			opts.AuditLog.PolicyConfigMapName,
			opts.AuditLogData))

	return newConverter(opts.ConverterConfig, baseExtenders...)
}

//func selectImageVersion(defaultName, defaultVersion, currentName, currentVersion string) (string, string, error) {
//	if currentVersion == "" || currentName == "" {
//		return defaultName, defaultVersion, nil
//	}
//
//	if defaultName != currentName {
//		return defaultName, defaultVersion, nil
//	}
//
//	result, err := compareVersions(defaultVersion, currentVersion)
//	if err != nil {
//		return "", "", err
//	}
//
//	if result < 0 {
//		// current version is greater than default version
//		return currentName, currentVersion, nil
//	}
//
//	return defaultName, defaultVersion, nil
//}

func (c Converter) ToShoot(runtime imv1.Runtime) (gardener.Shoot, error) {
	// The original implementation in the Provisioner: https://github.com/kyma-project/control-plane/blob/3dd257826747384479986d5d79eb20f847741aa6/components/provisioner/internal/model/gardener_config.go#L127

	// If you need to enhance the converter please adhere to the following convention:
	// - fields taken directly from Runtime CR must be added in this function
	// - if any logic is needed to be implemented, either enhance existing, or create a new extender

	shoot := gardener.Shoot{
		TypeMeta: v1.TypeMeta{
			Kind:       "Shoot",
			APIVersion: "core.gardener.cloud/v1beta1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      runtime.Spec.Shoot.Name,
			Namespace: fmt.Sprintf("garden-%s", c.config.Gardener.ProjectName),
		},
		Spec: gardener.ShootSpec{
			Purpose:           &runtime.Spec.Shoot.Purpose,
			Region:            runtime.Spec.Shoot.Region,
			SecretBindingName: &runtime.Spec.Shoot.SecretBindingName,
			Networking: &gardener.Networking{
				Type:     runtime.Spec.Shoot.Networking.Type,
				Nodes:    &runtime.Spec.Shoot.Networking.Nodes,
				Pods:     &runtime.Spec.Shoot.Networking.Pods,
				Services: &runtime.Spec.Shoot.Networking.Services,
			},
			ControlPlane: runtime.Spec.Shoot.ControlPlane,
		},
	}

	for _, extend := range c.extenders {
		if err := extend(runtime, &shoot); err != nil {
			return gardener.Shoot{}, err
		}
	}

	return shoot, nil
}
