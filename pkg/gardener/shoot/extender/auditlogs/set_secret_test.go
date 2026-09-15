package auditlogs

import (
	"slices"
	"testing"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
)

func Test_oSetSecret(t *testing.T) {
	t.Run("when dedicated=false, only shared auditlog reference is upserted and dedicated reference is not created", func(t *testing.T) {
		// given
		shoot := gardener.Shoot{}
		secretName := "test-secret"
		operate := oSetSecret(false, secretName)

		// when
		err := operate(&shoot)

		// then
		require.NoError(t, err)
		requireNoErrorAssertContainsSecretResource(t, SharedAuditlogSecretReference, secretName, shoot.Spec.Resources)
		assertDoesNotContainSecretResource(t, DedicatedAuditlogSecretReference, shoot.Spec.Resources)
	})

	t.Run("when dedicated=true and starting empty, dedicated reference is upserted and shared reference is created once as fallback", func(t *testing.T) {
		// given
		shoot := gardener.Shoot{}
		secretName := "test-secret"
		operate := oSetSecret(true, secretName)

		// when
		err := operate(&shoot)

		// then
		require.NoError(t, err)
		requireNoErrorAssertContainsSecretResource(t, DedicatedAuditlogSecretReference, secretName, shoot.Spec.Resources)
		requireNoErrorAssertContainsSecretResource(t, SharedAuditlogSecretReference, secretName, shoot.Spec.Resources)
	})

	t.Run("when dedicated=true and shared reference already exists with old secret, dedicated reference gets new secret and shared reference remains frozen", func(t *testing.T) {
		// given
		oldSharedSecret := "old-shared-secret"
		newDedicatedSecret := "new-dedicated-secret"
		shoot := gardener.Shoot{
			Spec: gardener.ShootSpec{
				Resources: []gardener.NamedResourceReference{
					{
						Name: SharedAuditlogSecretReference,
						ResourceRef: autoscalingv1.CrossVersionObjectReference{
							Name:       oldSharedSecret,
							Kind:       "Secret",
							APIVersion: "v1",
						},
					},
				},
			},
		}
		operate := oSetSecret(true, newDedicatedSecret)

		// when
		err := operate(&shoot)

		// then
		require.NoError(t, err)
		requireNoErrorAssertContainsSecretResource(t, DedicatedAuditlogSecretReference, newDedicatedSecret, shoot.Spec.Resources)
		requireNoErrorAssertContainsSecretResource(t, SharedAuditlogSecretReference, oldSharedSecret, shoot.Spec.Resources)
	})
}

func requireNoErrorAssertContainsSecretResource(t *testing.T, expectedRefName, expectedSecretName string, actual []gardener.NamedResourceReference) {
	index := slices.IndexFunc(actual, func(r gardener.NamedResourceReference) bool {
		return r.Name == expectedRefName
	})
	require.NotEqual(t, -1, index, "'%s' NamedResourceReference not found", expectedRefName)
	assert.Equal(t, expectedRefName, actual[index].Name)
	assert.Equal(t, expectedSecretName, actual[index].ResourceRef.Name)
}

func assertDoesNotContainSecretResource(t *testing.T, refName string, actual []gardener.NamedResourceReference) {
	index := slices.IndexFunc(actual, func(r gardener.NamedResourceReference) bool {
		return r.Name == refName
	})
	assert.Equal(t, -1, index, "'%s' NamedResourceReference should not be found", refName)
}
