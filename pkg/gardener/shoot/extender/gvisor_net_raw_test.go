package extender

import (
	"encoding/json"
	"testing"

	gvisorv1alpha1 "github.com/gardener/gardener-extension-runtime-gvisor/pkg/apis/config/v1alpha1"
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestEnsureGVisorNetRawDefault(t *testing.T) {
	t.Run("nil providerConfig yields default GVisorConfiguration with net-raw true", func(t *testing.T) {
		out, err := ensureGVisorNetRawDefault(nil)
		require.NoError(t, err)
		require.NotNil(t, out)
		var config gvisorv1alpha1.GVisorConfiguration
		require.NoError(t, json.Unmarshal(out.Raw, &config))
		require.Equal(t, gvisorv1alpha1.SchemeGroupVersion.String(), config.APIVersion)
		require.Equal(t, gvisorProviderConfigKind, config.Kind)
		require.NotNil(t, config.ConfigFlags)
		require.Equal(t, gvisorNetRawDefaultValue, (*config.ConfigFlags)[gvisorNetRawConfigKey])
	})

	t.Run("existing config without net-raw adds net-raw true", func(t *testing.T) {
		flags := map[string]string{"debug": "false"}
		config := gvisorv1alpha1.GVisorConfiguration{
			ConfigFlags: &flags,
		}
		config.APIVersion = gvisorv1alpha1.SchemeGroupVersion.String()
		config.Kind = gvisorProviderConfigKind
		raw, err := json.Marshal(config)
		require.NoError(t, err)
		out, err := ensureGVisorNetRawDefault(&runtime.RawExtension{Raw: raw})
		require.NoError(t, err)
		var outConfig gvisorv1alpha1.GVisorConfiguration
		require.NoError(t, json.Unmarshal(out.Raw, &outConfig))
		require.NotNil(t, outConfig.ConfigFlags)
		require.Equal(t, "false", (*outConfig.ConfigFlags)["debug"])
		require.Equal(t, gvisorNetRawDefaultValue, (*outConfig.ConfigFlags)[gvisorNetRawConfigKey])
	})

	t.Run("explicit net-raw is preserved", func(t *testing.T) {
		flags := map[string]string{gvisorNetRawConfigKey: "false"}
		config := gvisorv1alpha1.GVisorConfiguration{
			ConfigFlags: &flags,
		}
		config.APIVersion = gvisorv1alpha1.SchemeGroupVersion.String()
		config.Kind = gvisorProviderConfigKind
		raw, err := json.Marshal(config)
		require.NoError(t, err)
		out, err := ensureGVisorNetRawDefault(&runtime.RawExtension{Raw: raw})
		require.NoError(t, err)
		var outConfig gvisorv1alpha1.GVisorConfiguration
		require.NoError(t, json.Unmarshal(out.Raw, &outConfig))
		require.NotNil(t, outConfig.ConfigFlags)
		require.Equal(t, "false", (*outConfig.ConfigFlags)[gvisorNetRawConfigKey])
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		_, err := ensureGVisorNetRawDefault(&runtime.RawExtension{Raw: []byte(`{`)})
		require.Error(t, err)
	})

	t.Run("empty raw extension creates default config", func(t *testing.T) {
		out, err := ensureGVisorNetRawDefault(&runtime.RawExtension{Raw: []byte{}})
		require.NoError(t, err)
		require.NotNil(t, out)
		var config gvisorv1alpha1.GVisorConfiguration
		require.NoError(t, json.Unmarshal(out.Raw, &config))
		require.Equal(t, gvisorNetRawDefaultValue, (*config.ConfigFlags)[gvisorNetRawConfigKey])
	})

	t.Run("config with nil ConfigFlags gets initialized", func(t *testing.T) {
		config := gvisorv1alpha1.GVisorConfiguration{
			ConfigFlags: nil,
		}
		config.APIVersion = gvisorv1alpha1.SchemeGroupVersion.String()
		config.Kind = gvisorProviderConfigKind
		raw, err := json.Marshal(config)
		require.NoError(t, err)
		out, err := ensureGVisorNetRawDefault(&runtime.RawExtension{Raw: raw})
		require.NoError(t, err)
		var outConfig gvisorv1alpha1.GVisorConfiguration
		require.NoError(t, json.Unmarshal(out.Raw, &outConfig))
		require.NotNil(t, outConfig.ConfigFlags)
		require.Equal(t, gvisorNetRawDefaultValue, (*outConfig.ConfigFlags)[gvisorNetRawConfigKey])
		require.Equal(t, gvisorv1alpha1.SchemeGroupVersion.String(), outConfig.APIVersion)
		require.Equal(t, gvisorProviderConfigKind, outConfig.Kind)
	})

	t.Run("existing net-raw true is preserved", func(t *testing.T) {
		flags := map[string]string{gvisorNetRawConfigKey: "true"}
		config := gvisorv1alpha1.GVisorConfiguration{
			ConfigFlags: &flags,
		}
		config.APIVersion = gvisorv1alpha1.SchemeGroupVersion.String()
		config.Kind = gvisorProviderConfigKind
		raw, err := json.Marshal(config)
		require.NoError(t, err)
		out, err := ensureGVisorNetRawDefault(&runtime.RawExtension{Raw: raw})
		require.NoError(t, err)
		var outConfig gvisorv1alpha1.GVisorConfiguration
		require.NoError(t, json.Unmarshal(out.Raw, &outConfig))
		require.Equal(t, "true", (*outConfig.ConfigFlags)[gvisorNetRawConfigKey])
	})

	t.Run("multiple existing flags are preserved", func(t *testing.T) {
		flags := map[string]string{
			"debug":   "true",
			"nvproxy": "false",
			"foo":     "bar",
		}
		config := gvisorv1alpha1.GVisorConfiguration{
			ConfigFlags: &flags,
		}
		config.APIVersion = gvisorv1alpha1.SchemeGroupVersion.String()
		config.Kind = gvisorProviderConfigKind
		raw, err := json.Marshal(config)
		require.NoError(t, err)
		out, err := ensureGVisorNetRawDefault(&runtime.RawExtension{Raw: raw})
		require.NoError(t, err)
		var outConfig gvisorv1alpha1.GVisorConfiguration
		require.NoError(t, json.Unmarshal(out.Raw, &outConfig))
		require.Equal(t, "true", (*outConfig.ConfigFlags)["debug"])
		require.Equal(t, "false", (*outConfig.ConfigFlags)["nvproxy"])
		require.Equal(t, "bar", (*outConfig.ConfigFlags)["foo"])
		require.Equal(t, gvisorNetRawDefaultValue, (*outConfig.ConfigFlags)[gvisorNetRawConfigKey])
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
		var config gvisorv1alpha1.GVisorConfiguration
		require.NoError(t, json.Unmarshal(workers[0].CRI.ContainerRuntimes[0].ProviderConfig.Raw, &config))
		require.NotNil(t, config.ConfigFlags)
		require.Equal(t, gvisorNetRawDefaultValue, (*config.ConfigFlags)[gvisorNetRawConfigKey])
	})

	t.Run("worker with nil CRI is skipped", func(t *testing.T) {
		workers := []gardener.Worker{{
			Name: "w1",
			CRI:  nil,
		}}
		require.NoError(t, applyDefaultGVisorNetRaw(workers))
		require.Nil(t, workers[0].CRI)
	})

	t.Run("multiple container runtimes with mixed types", func(t *testing.T) {
		workers := []gardener.Worker{{
			Name: "w",
			CRI: &gardener.CRI{
				Name: "containerd",
				ContainerRuntimes: []gardener.ContainerRuntime{
					{Type: "runc"},
					{Type: gvisorContainerRuntimeType},
					{Type: "kata"},
				},
			},
		}}
		require.NoError(t, applyDefaultGVisorNetRaw(workers))
		require.Nil(t, workers[0].CRI.ContainerRuntimes[0].ProviderConfig)
		require.NotNil(t, workers[0].CRI.ContainerRuntimes[1].ProviderConfig)
		require.Nil(t, workers[0].CRI.ContainerRuntimes[2].ProviderConfig)

		var config gvisorv1alpha1.GVisorConfiguration
		require.NoError(t, json.Unmarshal(workers[0].CRI.ContainerRuntimes[1].ProviderConfig.Raw, &config))
		require.Equal(t, gvisorNetRawDefaultValue, (*config.ConfigFlags)[gvisorNetRawConfigKey])
	})

	t.Run("multiple gvisor runtimes in same worker", func(t *testing.T) {
		workers := []gardener.Worker{{
			Name: "w",
			CRI: &gardener.CRI{
				Name: "containerd",
				ContainerRuntimes: []gardener.ContainerRuntime{
					{Type: gvisorContainerRuntimeType},
					{Type: gvisorContainerRuntimeType},
				},
			},
		}}
		require.NoError(t, applyDefaultGVisorNetRaw(workers))
		for i := range workers[0].CRI.ContainerRuntimes {
			require.NotNil(t, workers[0].CRI.ContainerRuntimes[i].ProviderConfig)
			var config gvisorv1alpha1.GVisorConfiguration
			require.NoError(t, json.Unmarshal(workers[0].CRI.ContainerRuntimes[i].ProviderConfig.Raw, &config))
			require.Equal(t, gvisorNetRawDefaultValue, (*config.ConfigFlags)[gvisorNetRawConfigKey])
		}
	})

	t.Run("invalid providerConfig returns error", func(t *testing.T) {
		workers := []gardener.Worker{{
			Name: "w",
			CRI: &gardener.CRI{
				Name: "containerd",
				ContainerRuntimes: []gardener.ContainerRuntime{
					{
						Type:           gvisorContainerRuntimeType,
						ProviderConfig: &runtime.RawExtension{Raw: []byte(`{invalid`)},
					},
				},
			},
		}}
		err := applyDefaultGVisorNetRaw(workers)
		require.Error(t, err)
		require.Contains(t, err.Error(), "unmarshal")
	})

	t.Run("multiple workers with gvisor", func(t *testing.T) {
		workers := []gardener.Worker{
			{
				Name: "w1",
				CRI: &gardener.CRI{
					Name: "containerd",
					ContainerRuntimes: []gardener.ContainerRuntime{
						{Type: gvisorContainerRuntimeType},
					},
				},
			},
			{
				Name: "w2",
				CRI: &gardener.CRI{
					Name: "containerd",
					ContainerRuntimes: []gardener.ContainerRuntime{
						{Type: "runc"},
					},
				},
			},
			{
				Name: "w3",
				CRI: &gardener.CRI{
					Name: "containerd",
					ContainerRuntimes: []gardener.ContainerRuntime{
						{Type: gvisorContainerRuntimeType},
					},
				},
			},
		}
		require.NoError(t, applyDefaultGVisorNetRaw(workers))
		require.NotNil(t, workers[0].CRI.ContainerRuntimes[0].ProviderConfig)
		require.Nil(t, workers[1].CRI.ContainerRuntimes[0].ProviderConfig)
		require.NotNil(t, workers[2].CRI.ContainerRuntimes[0].ProviderConfig)
	})
}

func TestExtendWithGVisorNetRawDefault(t *testing.T) {
	t.Run("invokes defaulting on shoot provider workers", func(t *testing.T) {
		shoot := gardener.Shoot{
			Spec: gardener.ShootSpec{
				Provider: gardener.Provider{
					Workers: []gardener.Worker{{
						Name: "w",
						CRI: &gardener.CRI{
							Name: "containerd",
							ContainerRuntimes: []gardener.ContainerRuntime{
								{Type: gvisorContainerRuntimeType},
							},
						},
					}},
				},
			},
		}
		require.NoError(t, ExtendWithGVisorNetRawDefault(imv1.Runtime{}, &shoot))
		pc := shoot.Spec.Provider.Workers[0].CRI.ContainerRuntimes[0].ProviderConfig
		require.NotNil(t, pc)
		var config gvisorv1alpha1.GVisorConfiguration
		require.NoError(t, json.Unmarshal(pc.Raw, &config))
		require.NotNil(t, config.ConfigFlags)
		require.Equal(t, gvisorNetRawDefaultValue, (*config.ConfigFlags)[gvisorNetRawConfigKey])
	})
}
