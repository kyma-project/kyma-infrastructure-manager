package extender

import (
	"github.com/kyma-project/infrastructure-manager/pkg/config"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/testutils"
	"testing"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestTolerationsExtender(t *testing.T) {
	for _, testCase := range []struct {
		name                string
		region              string
		expectedTolerations []gardener.Toleration
	}{
		{
			name:   "Should extend Tolerations for me-central2",
			region: "me-central2",
			expectedTolerations: []gardener.Toleration{
				{
					Key: "ksa-assured-workload",
				},
			},
		},
		{
			name:                "Should not extend Tolerations for eu-de-1",
			region:              "eu-de-1",
			expectedTolerations: nil,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			// given
			shoot := testutils.FixEmptyGardenerShoot("shoot", "kcp-system")
			basicRuntime := imv1.Runtime{
				ObjectMeta: v1.ObjectMeta{
					Name:      "runtime",
					Namespace: "namespace",
				},
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Region: testCase.region,
					},
				},
			}

			// when
			extendWithTolerations := NewTolerationsExtender(config.TolerationsConfig{
				"me-central2": {gardener.Toleration{Key: "ksa-assured-workload"}},
			})
			err := extendWithTolerations(basicRuntime, &shoot)
			require.NoError(t, err)

			// then
			assert.Equal(t, testCase.expectedTolerations, shoot.Spec.Tolerations)
		})
	}
}
