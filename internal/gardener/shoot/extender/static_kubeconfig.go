package extender

import (
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"k8s.io/utils/ptr"
)

func ExtendWithStaticKubeconfig(_ imv1.Runtime, shoot *gardener.Shoot) error {
	shoot.Spec.Kubernetes.EnableStaticTokenKubeconfig = ptr.To(false)

	return nil
}
