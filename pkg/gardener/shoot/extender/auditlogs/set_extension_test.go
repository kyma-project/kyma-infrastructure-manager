package auditlogs

import (
	"bytes"
	"encoding/json"
	"slices"
	"testing"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_oSetExtension(t *testing.T) {
	for _, testCase := range []struct {
		shoot gardener.Shoot
		data  AuditLogData
	}{
		{
			shoot: gardener.Shoot{},
			data: AuditLogData{
				TenantID:   "tenant-id",
				ServiceURL: "testme",
			},
		},
	} {
		// given
		operate := oSetExtension(testCase.data)

		// when
		err := operate(&testCase.shoot)

		// then
		require.NoError(t, err)
		requireNoErrorAssetContainsAuditlogExtension(t, testCase.data, testCase.shoot.Spec.Extensions)
	}
}

func requireNoErrorAssetContainsAuditlogExtension(t *testing.T, data AuditLogData, actual []gardener.Extension) {
	index := slices.IndexFunc(actual, func(e gardener.Extension) bool {
		return e.Type == auditlogExtensionType
	})

	assert.NotEqual(t, -1, index, "no %s extension found", auditlogExtensionType)

	reader := bytes.NewReader(actual[index].ProviderConfig.Raw)
	var cfg AuditlogExtensionConfig

	err := json.NewDecoder(reader).Decode(&cfg)
	require.NoError(t, err)

	assert.Equal(t, data.TenantID, cfg.TenantID)
	assert.Equal(t, data.ServiceURL, cfg.ServiceURL)
	assert.Equal(t, auditlogReferenceName, cfg.SecretReferenceName)
}
