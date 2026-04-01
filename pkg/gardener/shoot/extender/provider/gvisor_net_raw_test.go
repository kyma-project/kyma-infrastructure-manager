package provider

import (
	"encoding/json"
	"testing"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestEnsureGVisorNetRawDefault(t *testing.T) {
	t.Run("nil providerConfig yields default GVisorConfiguration with net-raw true", func(t *testing.T) {
		out, err := ensureGVisorNetRawDefault(nil)
		require.NoError(t, err)
		require.NotNil(t, out)
		var m map[string]interface{}
		require.NoError(t, json.Unmarshal(out.Raw, &m))
		require.Equal(t, gvisorProviderConfigAPIVer, m["apiVersion"])
		require.Equal(t, gvisorProviderConfigKind, m["kind"])
		flags := m["configFlags"].(map[string]interface{})
		require.Equal(t, gvisorNetRawDefaultValue, flags[gvisorNetRawConfigKey])
	})

	t.Run("existing config without net-raw adds net-raw true", func(t *testing.T) {
		raw, err := json.Marshal(map[string]interface{}{
			"apiVersion": gvisorProviderConfigAPIVer,
			"kind":       gvisorProviderConfigKind,
			"configFlags": map[string]interface{}{
				"debug": "false",
			},
		})
		require.NoError(t, err)
		out, err := ensureGVisorNetRawDefault(&runtime.RawExtension{Raw: raw})
		require.NoError(t, err)
		var m map[string]interface{}
		require.NoError(t, json.Unmarshal(out.Raw, &m))
		flags := m["configFlags"].(map[string]interface{})
		require.Equal(t, "false", flags["debug"])
		require.Equal(t, gvisorNetRawDefaultValue, flags[gvisorNetRawConfigKey])
	})

	t.Run("explicit net-raw is preserved", func(t *testing.T) {
		raw, err := json.Marshal(map[string]interface{}{
			"apiVersion": gvisorProviderConfigAPIVer,
			"kind":       gvisorProviderConfigKind,
			"configFlags": map[string]interface{}{
				gvisorNetRawConfigKey: "false",
			},
		})
		require.NoError(t, err)
		out, err := ensureGVisorNetRawDefault(&runtime.RawExtension{Raw: raw})
		require.NoError(t, err)
		var m map[string]interface{}
		require.NoError(t, json.Unmarshal(out.Raw, &m))
		flags := m["configFlags"].(map[string]interface{})
		require.Equal(t, "false", flags[gvisorNetRawConfigKey])
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		_, err := ensureGVisorNetRawDefault(&runtime.RawExtension{Raw: []byte(`{`)})
		require.Error(t, err)
	})
}

func TestApplyDefaultGVisorNetRaw(t *testing.T) {
	t.Run("non-gvisor runtime unchanged", func(t *testing.T) {
		workers := []gardener.Worker{{
			Name: "w",
			CRI: &gardener.CRI{
				Name: "containerd",
				ContainerRuntimes: []gardener.ContainerRuntime{
					{Type: "runc"},
				},
			},
		}}
		require.NoError(t, applyDefaultGVisorNetRaw(workers))
		require.Nil(t, workers[0].CRI.ContainerRuntimes[0].ProviderConfig)
	})

	t.Run("gvisor without providerConfig gets default", func(t *testing.T) {
		workers := []gardener.Worker{{
			Name: "w",
			CRI: &gardener.CRI{
				Name: "containerd",
				ContainerRuntimes: []gardener.ContainerRuntime{
					{Type: gvisorContainerRuntimeType},
				},
			},
		}}
		require.NoError(t, applyDefaultGVisorNetRaw(workers))
		require.NotNil(t, workers[0].CRI.ContainerRuntimes[0].ProviderConfig)
		var m map[string]interface{}
		require.NoError(t, json.Unmarshal(workers[0].CRI.ContainerRuntimes[0].ProviderConfig.Raw, &m))
		flags := m["configFlags"].(map[string]interface{})
		require.Equal(t, gvisorNetRawDefaultValue, flags[gvisorNetRawConfigKey])
	})
}
