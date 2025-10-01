package token

import (
	"testing"
	"time"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestNewExpirationTimeExtender(t *testing.T) {
	t.Run("Should create an token expiration config for Shoot", func(t *testing.T) {
		// given
		maxTokenExpirationTime := &metav1.Duration{
			Duration: time.Second * 2592001, // 30 days and 1 second
		}
		extendTokenExpiration := ptr.To(true)

		extender := NewExpirationTimeExtender(*maxTokenExpirationTime, extendTokenExpiration)
		shoot := testutils.FixEmptyGardenerShoot("test", "dev")

		// when
		err := extender(imv1.Runtime{}, &shoot)

		// then
		require.NoError(t, err)
		assert.Equal(t, maxTokenExpirationTime, shoot.Spec.Kubernetes.KubeAPIServer.ServiceAccountConfig.MaxTokenExpiration)
		assert.Equal(t, extendTokenExpiration, shoot.Spec.Kubernetes.KubeAPIServer.ServiceAccountConfig.ExtendTokenExpiration)
	})
}
