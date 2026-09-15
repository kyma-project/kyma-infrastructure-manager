package auditlog

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_AuditlogsConfigurationError(t *testing.T) {
	for _, tc := range []struct {
		cfg               Configuration
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
		cfg          Configuration
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

func TestLoadConfiguration(t *testing.T) {
	t.Run("loads valid configuration file", func(t *testing.T) {
		content := `{
			"aws": {
				"eu-central-1": {
					"tenantID": "tenant-1",
					"serviceURL": "https://service1.example.com",
					"secretName": "secret-1"
				}
			},
			"gcp": {
				"us-east1": {
					"tenantID": "tenant-2",
					"serviceURL": "https://service2.example.com",
					"secretName": "secret-2"
				}
			}
		}`

		tmpFile := createTempFile(t, content)
		defer os.Remove(tmpFile)

		config, err := LoadConfiguration(tmpFile)
		require.NoError(t, err)
		require.NotNil(t, config)

		data, err := config.GetAuditLogData("aws", "eu-central-1")
		require.NoError(t, err)
		assert.Equal(t, "tenant-1", data.TenantID)
		assert.Equal(t, "https://service1.example.com", data.ServiceURL)
		assert.Equal(t, "secret-1", data.SecretName)
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		_, err := LoadConfiguration("/non/existent/path.json")
		require.Error(t, err)
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		content := `{invalid json`

		tmpFile := createTempFile(t, content)
		defer os.Remove(tmpFile)

		_, err := LoadConfiguration(tmpFile)
		require.Error(t, err)
	})

	t.Run("returns error for missing required fields", func(t *testing.T) {
		content := `{
			"aws": {
				"eu-central-1": {
					"tenantID": "tenant-1"
				}
			}
		}`

		tmpFile := createTempFile(t, content)
		defer os.Remove(tmpFile)

		_, err := LoadConfiguration(tmpFile)
		require.Error(t, err)
	})

	t.Run("returns error for invalid URL", func(t *testing.T) {
		content := `{
			"aws": {
				"eu-central-1": {
					"tenantID": "tenant-1",
					"serviceURL": "not-a-valid-url",
					"secretName": "secret-1"
				}
			}
		}`

		tmpFile := createTempFile(t, content)
		defer os.Remove(tmpFile)

		_, err := LoadConfiguration(tmpFile)
		require.Error(t, err)
	})
}

func createTempFile(t *testing.T, content string) string {
	t.Helper()
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "config.json")
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	require.NoError(t, err)
	return tmpFile
}
