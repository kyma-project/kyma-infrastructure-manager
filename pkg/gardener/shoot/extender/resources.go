package extender

import (
	"fmt"
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/extensions"
	v1 "k8s.io/api/autoscaling/v1"
	"slices"
	"strings"
)

func NewResourcesExtenderForPatch(resources []gardener.NamedResourceReference) func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
	return func(r imv1.Runtime, shoot *gardener.Shoot) error {

		resources = slices.DeleteFunc(resources, func(r gardener.NamedResourceReference) bool {
			return strings.Contains(r.Name, extensions.RegistryCacheSecretPrefix)
		})

		if resources != nil {
			shoot.Spec.Resources = resources
		}

		for _, cache := range r.Spec.Caching {
			if cache.Config.SecretReferenceName != nil && *cache.Config.SecretReferenceName != "" {
				shoot.Spec.Resources = append(shoot.Spec.Resources, gardener.NamedResourceReference{
					Name: fmt.Sprintf(extensions.RegistryCacheSecretNameFmt, cache.UID),
					ResourceRef: v1.CrossVersionObjectReference{
						Kind:       "Secret",
						APIVersion: "v1",
						Name:       fmt.Sprintf(extensions.RegistryCacheSecretNameFmt, cache.UID),
					},
				})
			}
		}

		return nil
	}
}
