package extensions

import (
	"encoding/json"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apimachineryRuntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"testing"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNetworkingFilterExtender(t *testing.T) {

	for _, testCase := range []struct {
		name                   string
		egressFilterEnabled    bool
		ingressEnabled         bool
		expectedFilterDisabled bool
	}{
		{
			name:                   "Networking filter should be enabled when both egress filter and blackholing are enabled",
			egressFilterEnabled:    true,
			ingressEnabled:         true,
			expectedFilterDisabled: false,
		},
		{
			name:                   "Networking filter should be disabled when both egress filter and blackholing are disabled",
			egressFilterEnabled:    false,
			ingressEnabled:         false,
			expectedFilterDisabled: true,
		},
		{
			name:                   "Networking filter should be disabled when egress filter is disabled, even if blackholing is enabled",
			egressFilterEnabled:    false,
			ingressEnabled:         true,
			expectedFilterDisabled: true,
		},
		{
			name:                   "Networking filter should be enabled when egress filter is enabled, also when blackholing is enabled",
			egressFilterEnabled:    true,
			ingressEnabled:         false,
			expectedFilterDisabled: false,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			// given
			filter := imv1.Filter{
				Ingress: &imv1.Ingress{Enabled: testCase.ingressEnabled},
				Egress:  imv1.Egress{Enabled: testCase.egressFilterEnabled},
			}
			// when
			disabled := isNetworkingFilterDisabled(filter)

			//then
			assert.Equal(t, testCase.expectedFilterDisabled, disabled)
		})
	}

	t.Run("Enable networking-filter extension", func(t *testing.T) {
		// given
		runtimeShoot := getRuntimeWithNetworkingFilter(true)

		// when
		extension, err := NewNetworkFilterExtension(runtimeShoot.Spec.Security.Networking.Filter)

		// then
		require.NoError(t, err)
		assert.Equal(t, ptr.To(false), extension.Disabled)
		assert.Equal(t, NetworkFilterType, extension.Type)
	})

	t.Run("Disable networking-filter extension", func(t *testing.T) {
		// given
		runtimeShoot := getRuntimeWithNetworkingFilter(false)

		// when
		extension, err := NewNetworkFilterExtension(runtimeShoot.Spec.Security.Networking.Filter)

		// then
		require.NoError(t, err)
		assert.Equal(t, ptr.To(true), extension.Disabled)
		assert.Equal(t, NetworkFilterType, extension.Type)
	})

	t.Run("Ingress-filter with blackholing raw provider config", func(t *testing.T) {
		// given
		runtimeShoot := getRuntimeWithIngressFiltering()

		// when
		extension, err := NewNetworkFilterExtension(runtimeShoot.Spec.Security.Networking.Filter)

		// then
		require.NoError(t, err)
		assert.Equal(t, false, ptr.Deref(extension.Disabled, true))
		assert.Equal(t, NetworkFilterType, extension.Type)

		filterProviderConfig := fixExpectedProviderConfiguration()

		providerJson, encodingErr := json.Marshal(filterProviderConfig)
		assert.NoError(t, encodingErr)

		rawConfig := &apimachineryRuntime.RawExtension{Raw: providerJson}
		assert.Equal(t, rawConfig.String(), extension.ProviderConfig.String())
	})
}

func fixExpectedProviderConfiguration() Configuration {
	filterProviderConfig := Configuration{
		TypeMeta: metav1.TypeMeta{},
		EgressFilter: &EgressFilter{
			BlackholingEnabled: true,
		},
	}
	return filterProviderConfig
}

func getRuntimeWithNetworkingFilter(enabled bool) imv1.Runtime {
	return imv1.Runtime{
		Spec: imv1.RuntimeSpec{
			Shoot: imv1.RuntimeShoot{
				Name: "myshoot",
			},
			Security: imv1.Security{
				Networking: imv1.NetworkingSecurity{
					Filter: imv1.Filter{
						Egress: imv1.Egress{
							Enabled: enabled,
						},
					},
				},
			},
		},
	}
}

func getRuntimeWithIngressFiltering() imv1.Runtime {
	runtime := getRuntimeWithNetworkingFilter(true)
	runtime.Spec.Security.Networking.Filter.Ingress = &imv1.Ingress{
		Enabled: true,
	}

	return runtime
}
