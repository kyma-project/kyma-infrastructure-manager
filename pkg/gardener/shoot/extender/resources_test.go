package extender

import (
	registrycache "github.com/kyma-project/kim-snatch/api/v1beta1"
	"k8s.io/utils/ptr"
	"testing"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/autoscaling/v1"
)

func TestNewResourcesExtenderForPatch(t *testing.T) {
	t.Run("should remove old reg-cache resources and add new ones", func(t *testing.T) {
		// given
		shoot := gardener.Shoot{
			Spec: gardener.ShootSpec{
				Resources: []gardener.NamedResourceReference{
					{Name: "reg-cache-old-resource"},
					{Name: "some-other-resource"},
				},
			},
		}

		runtime := imv1.Runtime{
			Spec: imv1.RuntimeSpec{
				Caching: []imv1.ImageRegistryCache{
					{
						UID: "cache-1",
						Config: registrycache.RegistryCacheConfigSpec{
							SecretReferenceName: ptr.To("secret-1"),
						},
					},
					{
						UID: "cache-2",
						Config: registrycache.RegistryCacheConfigSpec{
							SecretReferenceName: ptr.To("secret-2"),
						},
					},
				},
			},
		}

		extender := NewResourcesExtenderForPatch(shoot.Spec.Resources)

		// when
		err := extender(runtime, &shoot)

		// then
		require.NoError(t, err)

		// Verify old "reg-cache-" resources are removed
		assert.NotContains(t, shoot.Spec.Resources, gardener.NamedResourceReference{Name: "reg-cache-old-resource"})

		// Verify new resources are added
		assert.Contains(t, shoot.Spec.Resources, gardener.NamedResourceReference{
			Name: "reg-cache-cache-1",
			ResourceRef: v1.CrossVersionObjectReference{
				Kind:       "Secret",
				APIVersion: "v1",
				Name:       "reg-cache-cache-1",
			},
		})
		assert.Contains(t, shoot.Spec.Resources, gardener.NamedResourceReference{
			Name: "reg-cache-cache-2",
			ResourceRef: v1.CrossVersionObjectReference{
				Kind:       "Secret",
				APIVersion: "v1",
				Name:       "reg-cache-cache-2",
			},
		})

		// Verify unrelated resources are preserved
		assert.Contains(t, shoot.Spec.Resources, gardener.NamedResourceReference{Name: "some-other-resource"})
	})
}
