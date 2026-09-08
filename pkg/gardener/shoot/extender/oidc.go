package extender

import (
	"fmt"
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
)

const (
	StructuredAuthConfigFmt = "structured-auth-config-%s"
)

func NewOidcExtender() func(runtime imv1.Runtime, shoot *gardener.Shoot) error {

	return func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
		cmName := fmt.Sprintf(StructuredAuthConfigFmt, runtime.Spec.Shoot.Name)

		shoot.Spec.Kubernetes.KubeAPIServer = &gardener.KubeAPIServerConfig{
			StructuredAuthentication: &gardener.StructuredAuthentication{
				ConfigMapName: cmName,
			},
		}

		return nil
	}
}
