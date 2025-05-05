package openstack

import (
	"encoding/json"
	"testing"

	"github.com/gardener/gardener-extension-provider-openstack/pkg/apis/openstack/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestControlPlaneConfig(t *testing.T) {
	t.Run("Create Control Plane config", func(t *testing.T) {
		// when
		controlPlaneConfigBytes, err := GetControlPlaneConfig([]string{"europe-west3a"})

		// then
		require.NoError(t, err)

		var controlPlaneConfig v1alpha1.ControlPlaneConfig
		err = json.Unmarshal(controlPlaneConfigBytes, &controlPlaneConfig)
		require.NoError(t, err)

		assert.Equal(t, apiVersion, controlPlaneConfig.APIVersion)
		assert.Equal(t, controlPlaneConfigKind, controlPlaneConfig.Kind)
		assert.Equal(t, defaultLoadBalancerProvider, controlPlaneConfig.LoadBalancerProvider)
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
		require.NoError(t, err)

		assert.Equal(t, apiVersion, infrastructureConfig.APIVersion)
		assert.Equal(t, infrastructureConfigKind, infrastructureConfig.Kind)
		assert.Equal(t, "10.250.0.0/22", infrastructureConfig.Networks.Workers)
		assert.Equal(t, defaultFloatingPoolName, infrastructureConfig.FloatingPoolName)
	})
}
