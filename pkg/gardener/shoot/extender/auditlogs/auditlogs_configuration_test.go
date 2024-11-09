package auditlogs

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_AuditlogsConfigurationError(t *testing.T) {
	for _, tc := range []struct {
		cfg               AuditlogsConfiguration
		expectedErr       error
		expectedErrString string
	}{
		{
			expectedErr:       ErrConfigurationNotFound,
			expectedErrString: "audit logs configuration not found: missing providerType: 'test'",
		},
		{
			cfg: map[string]map[string]AuditLogData{
				"test": {},
			},
			expectedErr:       ErrConfigurationNotFound,
			expectedErrString: "audit logs configuration not found: missing region: 'me' for providerType: 'test'",
		},
	} {
		// when
		_, err := tc.cfg.GetAuditLogData("test", "me")

		// then
		assert.ErrorIs(t, err, tc.expectedErr)
		assert.Equal(t, tc.expectedErrString, err.Error())
	}
}

func Test_AuditlogsConfigurationNoError(t *testing.T) {
	for _, testCase := range []struct {
		cfg          AuditlogsConfiguration
		providerType string
		region       string
		expected     AuditLogData
	}{
		{
			cfg: map[string]map[string]AuditLogData{
				"p1": {
					"r1": fixTestAuditlogData(1),
				},
				"p2": {
					"r2": fixTestAuditlogData(2),
					"r3": fixTestAuditlogData(3),
				},
			},
			providerType: "p2",
			region:       "r3",
			expected:     fixTestAuditlogData(3),
		},
	} {
		// when
		actual, err := testCase.cfg.GetAuditLogData(testCase.providerType, testCase.region)

		// then
		require.NoError(t, err)
		assert.Equal(t, testCase.expected, actual)
	}
}

func fixTestAuditlogData(id int) AuditLogData {
	return AuditLogData{
		TenantID:   fmt.Sprintf("%d", id),
		ServiceURL: fmt.Sprintf("https://test.service.%d", id),
		SecretName: fmt.Sprintf("test-service-%d", id),
	}

}
