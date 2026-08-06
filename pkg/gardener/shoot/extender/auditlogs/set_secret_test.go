package auditlogs

import (
	"slices"
	"testing"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_oSetSecret(t *testing.T) {
	for _, testCase := range []struct {
		shoot         gardener.Shoot
		referenceName string
		secretName    string
	}{
		{
			shoot:         gardener.Shoot{},
			referenceName: SharedAuditlogSecretReference,
			secretName:    "test-secret",
		},
	} {
		// given
		operate := oSetSecret(testCase.referenceName, testCase.secretName)

		// when
		err := operate(&testCase.shoot)

		// then
		require.NoError(t, err)
		requireNoErrorAssertContainsSecretResource(t, testCase.referenceName, testCase.secretName, testCase.shoot.Spec.Resources)
	}
}

func requireNoErrorAssertContainsSecretResource(t *testing.T, expectedRefName, expectedSecretName string, actual []gardener.NamedResourceReference) {
	index := slices.IndexFunc(actual, func(r gardener.NamedResourceReference) bool {
		return r.Name == expectedRefName
	})
	require.NotEqual(t, -1, index, "'%s' NamedResourceReference not found", expectedRefName)
	assert.Equal(t, expectedRefName, actual[index].Name)
	assert.Equal(t, expectedSecretName, actual[index].ResourceRef.Name)
}
