package extender

import (
	"testing"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestExtendWithCredentialsBinding_CredentialBindingEnabled(t *testing.T) {
	runtime := imv1.Runtime{}
	runtime.Spec.Shoot.SecretBindingName = "my-secret"
	shoot := &gardener.Shoot{}

	err := ExtendWithCredentialsBinding(true)(runtime, shoot)
	assert.NoError(t, err)
	assert.Equal(t, "my-secret", *shoot.Spec.CredentialsBindingName)
	assert.Nil(t, shoot.Spec.SecretBindingName) //nolint:staticcheck
}

func TestExtendWithCredentialsBinding_CredentialBindingDisabled(t *testing.T) {
	runtime := imv1.Runtime{}
	runtime.Spec.Shoot.SecretBindingName = "my-secret"
	shoot := &gardener.Shoot{}

	err := ExtendWithCredentialsBinding(false)(runtime, shoot)
	assert.NoError(t, err)
	assert.Equal(t, "my-secret", *shoot.Spec.SecretBindingName)
	assert.Nil(t, shoot.Spec.CredentialsBindingName)
}
