package extender

import (
	"testing"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAnnotationsExtender(t *testing.T) {
	licenceType := "licence"

	for _, testCase := range []struct {
		name                string
		runtime             imv1.Runtime
		expectedAnnotations map[string]string
	}{
		{
			name: "Create with basic annotations",
			runtime: imv1.Runtime{
				ObjectMeta: v1.ObjectMeta{
					Name:      "runtime",
					Namespace: "namespace",
					Labels: map[string]string{
						"kyma-project.io/runtime-id": "runtime-id",
					},
					Generation: 100,
				},
			},
			expectedAnnotations: map[string]string{
				"infrastructuremanager.kyma-project.io/runtime-id":         "runtime-id",
				"infrastructuremanager.kyma-project.io/runtime-generation": "should-fail"},
		},
		{
			name: "Create licence type annotation",
			runtime: imv1.Runtime{
				ObjectMeta: v1.ObjectMeta{
					Name:      "runtime",
					Namespace: "namespace",
					Labels: map[string]string{
						"kyma-project.io/runtime-id": "runtime-id",
					},
				},
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						LicenceType: &licenceType,
					},
				},
			},
			expectedAnnotations: map[string]string{
				"infrastructuremanager.kyma-project.io/runtime-id":         "runtime-id",
				"infrastructuremanager.kyma-project.io/licence-type":       "licence",
				"infrastructuremanager.kyma-project.io/runtime-generation": "0"},
		},
		{
			name: "Create restricted EU access annotation for cf-eu11 region",
			runtime: imv1.Runtime{
				ObjectMeta: v1.ObjectMeta{
					Name:      "runtime",
					Namespace: "namespace",
					Labels: map[string]string{
						"kyma-project.io/runtime-id": "runtime-id",
					},
				},
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						PlatformRegion: "cf-eu11",
					},
				},
			},
			expectedAnnotations: map[string]string{
				"infrastructuremanager.kyma-project.io/runtime-id":         "runtime-id",
				"support.gardener.cloud/eu-access-for-cluster-nodes":       "true",
				"infrastructuremanager.kyma-project.io/runtime-generation": "0"},
		},
		{
			name: "Create restricted EU access annotation for cf-ch20 region",
			runtime: imv1.Runtime{
				ObjectMeta: v1.ObjectMeta{
					Name:      "runtime",
					Namespace: "namespace",
					Labels: map[string]string{
						"kyma-project.io/runtime-id": "runtime-id",
					},
				},
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						PlatformRegion: "cf-ch20",
					},
				},
			},
			expectedAnnotations: map[string]string{
				"infrastructuremanager.kyma-project.io/runtime-id":         "runtime-id",
				"support.gardener.cloud/eu-access-for-cluster-nodes":       "true",
				"infrastructuremanager.kyma-project.io/runtime-generation": "0"},
		},
	} {
		// given
		shoot := fixEmptyGardenerShoot("shoot", "kcp-system")

		// when
		err := ExtendWithAnnotations(testCase.runtime, &shoot)
		require.NoError(t, err)

		// then
		assert.Equal(t, testCase.expectedAnnotations, shoot.Annotations)
	}
}
