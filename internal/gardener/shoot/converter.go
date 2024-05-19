package shoot

import (
	gardenerv1beta "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Converter struct {
	extenders []extender
}

type extender func(imv1.RuntimeShoot, *gardenerv1beta.Shoot) error

func NewConverter(defaultKubernetesVersion, defaultImage string) Converter {
	extenders := []extender{
		annotationsExtender,
		newKubernetesExtender(defaultKubernetesVersion),
		networkingExtender,
		providerExtender,
		dnsExtender,
	}

	return Converter{
		extenders: extenders,
	}
}

func (c Converter) ToShoot(runtime imv1.Runtime) (gardenerv1beta.Shoot, error) {
	shoot := gardenerv1beta.Shoot{
		ObjectMeta: v1.ObjectMeta{
			Name:      runtime.Spec.Shoot.Name,
			Namespace: runtime.Namespace,
		},
		Spec: gardenerv1beta.ShootSpec{
			Purpose:           &runtime.Spec.Shoot.Purpose,
			Region:            runtime.Spec.Shoot.Region,
			SecretBindingName: &runtime.Spec.Shoot.SecretBindingName,
			ControlPlane:      &runtime.Spec.Shoot.ControlPlane,
		},
	}

	for _, extender := range c.extenders {
		if err := extender(runtime.Spec.Shoot, &shoot); err != nil {
			return gardenerv1beta.Shoot{}, err
		}
	}

	return shoot, nil
}
