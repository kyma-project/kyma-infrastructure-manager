package gdch

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewInfrastructureConfig(t *testing.T) {
	t.Run("builds config with expected TypeMeta and networks", func(t *testing.T) {
		workerCIDR := "10.72.0.0/24"
		zoneNames := []string{"us-west16-b", "us-west16-c", "us-west16-d"}

		got, err := NewInfrastructureConfig(workerCIDR, zoneNames)

		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, infrastructureConfigKind, got.Kind)
		assert.Equal(t, apiVersion, got.APIVersion)
		assert.Equal(t, workerCIDR, got.Networks.NodeCIDR)
		assert.False(t, got.EnableEgress)

		require.Len(t, got.Networks.Zones, len(zoneNames))
		wantZones, zonesErr := generateGDCHZones(workerCIDR, zoneNames)
		require.NoError(t, zonesErr)
		assert.Equal(t, wantZones, got.Networks.Zones)
	})

	t.Run("returns error and empty config for invalid zone count", func(t *testing.T) {
		got, err := NewInfrastructureConfig("10.72.0.0/24", []string{})

		require.Error(t, err)
		assert.ErrorIs(t, err, errInvalidZoneCount)
		assert.Equal(t, &InfrastructureConfig{}, got)
	})

	t.Run("returns error and empty config for invalid CIDR", func(t *testing.T) {
		got, err := NewInfrastructureConfig("not a cidr", []string{"a", "b", "c"})

		require.Error(t, err)
		assert.ErrorIs(t, err, errInvalidCIDR)
		assert.Equal(t, &InfrastructureConfig{}, got)
	})

	t.Run("returns error and empty config for duplicate zone names", func(t *testing.T) {
		got, err := NewInfrastructureConfig("10.72.0.0/24", []string{"a", "a", "b"})

		require.Error(t, err)
		assert.ErrorIs(t, err, errDuplicateZone)
		assert.Equal(t, &InfrastructureConfig{}, got)
	})

	t.Run("returns error and empty config for CIDR too small", func(t *testing.T) {
		got, err := NewInfrastructureConfig("10.72.0.0/31", []string{"a", "b", "c"})

		require.Error(t, err)
		assert.ErrorIs(t, err, errCIDRTooSmall)
		assert.Equal(t, &InfrastructureConfig{}, got)
	})
}

func TestGetInfrastructureConfig(t *testing.T) {
	t.Run("marshals a valid config to JSON", func(t *testing.T) {
		workerCIDR := "10.72.0.0/24"
		zoneNames := []string{"us-west16-b", "us-west16-c", "us-west16-d"}

		raw, err := GetInfrastructureConfig(workerCIDR, zoneNames)

		require.NoError(t, err)
		require.NotNil(t, raw)

		var got InfrastructureConfig
		require.NoError(t, json.Unmarshal(raw, &got))

		want, wantErr := NewInfrastructureConfig(workerCIDR, zoneNames)
		require.NoError(t, wantErr)
		assert.Equal(t, *want, got)

		assert.JSONEq(t, `{
			"kind": "InfrastructureConfig",
			"apiVersion": "gdch.provider.extensions.gardener.gdc.goog/v1alpha1",
			"enableEgress": false,
			"networks": {
				"nodeCIDR": "10.72.0.0/24",
				"zones": [
					{"name": "us-west16-b", "cidr": "10.72.0.0/26"},
					{"name": "us-west16-c", "cidr": "10.72.0.64/26"},
					{"name": "us-west16-d", "cidr": "10.72.0.128/26"}
				]
			}
		}`, string(raw))
	})

	t.Run("returns error and nil bytes for invalid input", func(t *testing.T) {
		raw, err := GetInfrastructureConfig("not a cidr", []string{"a", "b", "c"})

		require.Error(t, err)
		assert.ErrorIs(t, err, errInvalidCIDR)
		assert.Nil(t, raw)
	})
}

func TestNewControlPlaneConfig(t *testing.T) {
	got := NewControlPlaneConfig()

	require.NotNil(t, got)
	assert.Equal(t, controlPlaneConfigKind, got.Kind)
	assert.Equal(t, apiVersion, got.APIVersion)
}

func TestGetControlPlaneConfig(t *testing.T) {
	t.Run("marshals control plane config regardless of zone input", func(t *testing.T) {
		raw, err := GetControlPlaneConfig([]string{"a", "b", "c"})

		require.NoError(t, err)
		assert.JSONEq(t, `{
			"kind": "ControlPlaneConfig",
			"apiVersion": "gdch.provider.extensions.gardener.gdc.goog/v1alpha1"
		}`, string(raw))
	})

	t.Run("ignores nil zone names", func(t *testing.T) {
		raw, err := GetControlPlaneConfig(nil)

		require.NoError(t, err)
		assert.JSONEq(t, `{
			"kind": "ControlPlaneConfig",
			"apiVersion": "gdch.provider.extensions.gardener.gdc.goog/v1alpha1"
		}`, string(raw))
	})
}

func TestGetInfrastructureConfig_VariableZones(t *testing.T) {
	t.Run("N=1 at landscape /19 emits single-zone JSON", func(t *testing.T) {
		raw, err := GetInfrastructureConfig("10.72.0.0/19", []string{"a"})

		require.NoError(t, err)
		assert.JSONEq(t, `{
			"kind": "InfrastructureConfig",
			"apiVersion": "gdch.provider.extensions.gardener.gdc.goog/v1alpha1",
			"enableEgress": false,
			"networks": {
				"nodeCIDR": "10.72.0.0/19",
				"zones": [
					{"name": "a", "cidr": "10.72.0.0/19"}
				]
			}
		}`, string(raw))
	})

	t.Run("N=2 at landscape /19 emits two /20 zones", func(t *testing.T) {
		raw, err := GetInfrastructureConfig("10.72.0.0/19", []string{"a", "b"})

		require.NoError(t, err)
		assert.JSONEq(t, `{
			"kind": "InfrastructureConfig",
			"apiVersion": "gdch.provider.extensions.gardener.gdc.goog/v1alpha1",
			"enableEgress": false,
			"networks": {
				"nodeCIDR": "10.72.0.0/19",
				"zones": [
					{"name": "a", "cidr": "10.72.0.0/20"},
					{"name": "b", "cidr": "10.72.16.0/20"}
				]
			}
		}`, string(raw))
	})
}
