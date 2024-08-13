package extender

import (
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestExtendWithStaticKubeconfig(t *testing.T) {
	// given
	runtime := imv1.Runtime{}
	shoot := fixEmptyGardenerShoot("test", "dev")

	// when
	err := ExtendWithStaticKubeconfig(runtime, &shoot)

	// then
	require.NoError(t, err)
	assert.Equal(t, false, *shoot.Spec.Kubernetes.EnableStaticTokenKubeconfig)
}
