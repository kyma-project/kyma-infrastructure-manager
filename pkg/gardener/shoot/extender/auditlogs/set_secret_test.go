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
		shoot      gardener.Shoot
		secretName string
	}{
		{
			shoot:      gardener.Shoot{},
			secretName: "test-secret",
		},
	} {
		// given
		operate := oSetSecret(testCase.secretName)

		// when
		err := operate(&testCase.shoot)

		// then
		require.NoError(t, err)
		requireNoErrorAssertContainsSecretResource(t, testCase.secretName, testCase.shoot.Spec.Resources)
	}
}

func requireNoErrorAssertContainsSecretResource(t *testing.T, expected string, actual []gardener.NamedResourceReference) {
	index := slices.IndexFunc(actual, func(r gardener.NamedResourceReference) bool {
		return r.Name == auditlogSecretReference
	})
	require.NotEqual(t, -1, index, "'%s' NamedResourceReference not found", auditlogSecretReference)
	assert.Equal(t, auditlogSecretReference, actual[index].Name)
	assert.Equal(t, expected, actual[index].ResourceRef.Name)
}
