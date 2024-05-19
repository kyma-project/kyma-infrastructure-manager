package shoot

import (
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func fixEmptyGardenerShoot(name, namespace string) gardener.Shoot {
	return gardener.Shoot{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: gardener.ShootSpec{},
	}
}
