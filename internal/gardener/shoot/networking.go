package shoot

import (
	gardenerv1beta "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
)

func networkingExtender(runtimeShoot imv1.RuntimeShoot, shoot *gardenerv1beta.Shoot) error {
	runtimeNetworking := runtimeShoot.Networking

	shoot.Spec.Networking = &gardenerv1beta.Networking{
		Nodes:    &runtimeNetworking.Nodes,
		Pods:     &runtimeNetworking.Pods,
		Services: &runtimeNetworking.Services,
	}

	return nil
}
