package gcp

import (
	"encoding/json"
	"testing"

	"github.com/gardener/gardener-extension-provider-gcp/pkg/apis/gcp/v1alpha1"
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
		assert.NoError(t, err)

		assert.Equal(t, apiVersion, controlPlaneConfig.TypeMeta.APIVersion)
		assert.Equal(t, controlPlaneConfigKind, controlPlaneConfig.TypeMeta.Kind)
		assert.Equal(t, "europe-west3a", controlPlaneConfig.Zone)
	})

	t.Run("Return error when zones not provided", func(t *testing.T) {
		// when
		_, err := GetControlPlaneConfig([]string{})

		// then
		require.Error(t, err)

		// when
		_, err = GetControlPlaneConfig(nil)

		// then
		require.Error(t, err)
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

		assert.Equal(t, "10.250.0.0/22", infrastructureConfig.Networks.Workers)
		assert.Equal(t, "10.250.0.0/22", infrastructureConfig.Networks.Worker)
	})
}
