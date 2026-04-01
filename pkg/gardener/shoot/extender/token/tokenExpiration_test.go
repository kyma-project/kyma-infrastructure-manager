package token

import (
	"testing"
	"time"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewExpirationTimeExtender(t *testing.T) {
	for _, testCase := range []struct {
		name                       string
		providedTokenExpiration    string
		expectedExpirationDuration time.Duration
		error                      bool
	}{
		{
			name:                       "Should set token expiration to 30 days if provided token time is shorter than 30 days",
			providedTokenExpiration:    "24h",
			expectedExpirationDuration: lowerBound,
		},
		{
			name:                       "Should set token expiration to 90 days if provided token time is longer than 90 days",
			providedTokenExpiration:    "2400h",
			expectedExpirationDuration: upperBound,
		},
		{
			name:                       "Should set user specific token expiration time if it is between boundaries 30 days and 90 days",
			providedTokenExpiration:    "1200h", // 50 days
			expectedExpirationDuration: 1200 * time.Hour,
		},
		{
			name:                       "Should set default token expiration time (30 days) if token time is not provided",
			providedTokenExpiration:    "",
			expectedExpirationDuration: lowerBound,
		},
		{
			name:                    "Should return an error if it was problem during parsing time format",
			providedTokenExpiration: "30d",
			error:                   true,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			// given
			extender := NewExpirationTimeExtender(testCase.providedTokenExpiration)
			shoot := testutils.FixEmptyGardenerShoot("test", "dev")

			// when
			err := extender(imv1.Runtime{}, &shoot)

			// then
			if testCase.error {
				require.Error(t, err)
				assert.Nil(t, shoot.Spec.Kubernetes.KubeAPIServer.ServiceAccountConfig.MaxTokenExpiration)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, testCase.expectedExpirationDuration, shoot.Spec.Kubernetes.KubeAPIServer.ServiceAccountConfig.MaxTokenExpiration.Duration)
		})
	}
}
