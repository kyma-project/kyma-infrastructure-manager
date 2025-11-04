package alicloud

import (
	"encoding/json"
	"testing"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestControlPlaneConfig(t *testing.T) {
	t.Run("Create Control Plane config", func(t *testing.T) {
		// when
		controlPlaneConfigBytes, err := GetControlPlaneConfig(nil)

		// then
		require.NoError(t, err)

		var controlPlaneConfig v1alpha1.ControlPlaneConfig
		err = json.Unmarshal(controlPlaneConfigBytes, &controlPlaneConfig)
		assert.NoError(t, err)

		assert.Equal(t, apiVersion, controlPlaneConfig.APIVersion)
		assert.Equal(t, controlPlaneConfigKind, controlPlaneConfig.Kind)
	})

}

func TestInfrastructureConfig(t *testing.T) {
	t.Run("Create Infrastructure config", func(t *testing.T) {
		// when
		infrastructureConfigBytes, err := GetInfrastructureConfig("10.250.0.0/22", nil)

		// then
		require.NoError(t, err)

		var infrastructureConfig v1alpha1.InfrastructureConfig
		err = json.Unmarshal(infrastructureConfigBytes, &infrastructureConfig)
		assert.NoError(t, err)

		assert.Equal(t, "10.250.0.0/22", infrastructureConfig.Networks.Zones)
	})
}
