package shoot

import (
	"fmt"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/config"
	extender2 "github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Extend func(imv1.Runtime, *gardener.Shoot) error

type Converter struct {
	extenders []Extend
	config    config.ConverterConfig
}

func NewConverter(config config.ConverterConfig, shootFromState *gardener.Shoot) Converter {
	extenders := []Extend{
		extender2.ExtendWithAnnotations,
		extender2.ExtendWithLabels,
		extender2.NewKubernetesExtender(config.Kubernetes.DefaultVersion),
		extender2.NewProviderExtender(config.Provider.AWS.EnableIMDSv2, config.MachineImage.DefaultName, config.MachineImage.DefaultVersion, shootFromState),
		extender2.NewDNSExtender(config.DNS.SecretName, config.DNS.DomainPrefix, config.DNS.ProviderType),
		extender2.NewOidcExtender(config.Kubernetes.DefaultOperatorOidc),
		extender2.ExtendWithCloudProfile,
		extender2.ExtendWithNetworkFilter,
		extender2.ExtendWithCertConfig,
		extender2.ExtendWithExposureClassName,
		extender2.ExtendWithTolerations,
		extender2.NewMaintenanceExtender(config.Kubernetes.EnableKubernetesVersionAutoUpdate, config.Kubernetes.EnableMachineImageVersionAutoUpdate),
	}

	return Converter{
		extenders: extenders,
		config:    config,
	}
}

func (c Converter) ToShoot(runtime imv1.Runtime, shootFromState *gardener.Shoot) (gardener.Shoot, error) {
	// The original implementation in the Provisioner: https://github.com/kyma-project/control-plane/blob/3dd257826747384479986d5d79eb20f847741aa6/components/provisioner/internal/model/gardener_config.go#L127

	// If you need to enhance the converter please adhere to the following convention:
	// - fields taken directly from Runtime CR must be added in this function
	// - if any logic is needed to be implemented, either enhance existing, or create a new extender

	shoot := gardener.Shoot{
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

	//TODO: check how it behaves for a new cluster

	return shoot, nil
}
