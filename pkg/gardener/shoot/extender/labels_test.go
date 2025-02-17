package extender

import (
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/testutils"
	"testing"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestLabelsExtender(t *testing.T) {
	t.Run("Create labels", func(t *testing.T) {
		// given
		shoot := testutils.FixEmptyGardenerShoot("shoot", "kcp-system")
		runtime := imv1.Runtime{
			ObjectMeta: v1.ObjectMeta{
				Name:      "runtime",
				Namespace: "namespace",
				Labels: map[string]string{
					"kyma-project.io/global-account-id": "global-account-id",
					"kyma-project.io/subaccount-id":     "subaccount-id",
				},
			},
		}

		// when
		err := ExtendWithLabels(runtime, &shoot)
		require.NoError(t, err)

		// then
		assert.Equal(t, map[string]string{"account": "global-account-id", "subaccount": "subaccount-id"}, shoot.Labels)
	})
}
