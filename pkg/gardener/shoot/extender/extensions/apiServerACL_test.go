package extensions

import (
	"encoding/json"
	"testing"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/hyperscaler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"
)

func TestNewApiServerACLExtension(t *testing.T) {

	for _, testCase := range []struct {
		name            string
		userIPs         []string
		operatorIPs     []string
		kcpIP           string
		expectDisabled  bool
		expectedCIDRs   []string
		expectNilConfig bool
	}{
		{
			name:            "Should create enabled ACL extension with user, operator and KCP IPs",
			userIPs:         []string{"10.0.0.1/32", "10.0.0.2/32"},
			operatorIPs:     []string{"192.168.1.0/24"},
			kcpIP:           "172.16.0.1/32",
			expectDisabled:  false,
			expectedCIDRs:   []string{"10.0.0.1/32", "10.0.0.2/32", "192.168.1.0/24", "172.16.0.1/32"},
			expectNilConfig: false,
		},
		{
			name:            "Should create disabled ACL extension when user IPs are empty",
			userIPs:         []string{},
			operatorIPs:     []string{"192.168.1.0/24"},
			kcpIP:           "172.16.0.1/32",
			expectDisabled:  true,
			expectNilConfig: true,
		},
		{
			name:            "Should create disabled ACL extension when user IPs are nil",
			userIPs:         nil,
			operatorIPs:     []string{"192.168.1.0/24"},
			kcpIP:           "172.16.0.1/32",
			expectDisabled:  true,
			expectNilConfig: true,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			// when
			ext, err := NewApiServerACLExtension(testCase.userIPs, testCase.operatorIPs, testCase.kcpIP)

			// then
			require.NoError(t, err)
			require.NotNil(t, ext)

			assert.Equal(t, ApiServerACLExtensionType, ext.Type)
			assert.Equal(t, ptr.To(testCase.expectDisabled), ext.Disabled)

			if testCase.expectNilConfig {
				assert.Nil(t, ext.ProviderConfig)
			} else {
				require.NotNil(t, ext.ProviderConfig)
				require.NotNil(t, ext.ProviderConfig.Raw)

				var config aclProviderConfig
				err = json.Unmarshal(ext.ProviderConfig.Raw, &config)
				require.NoError(t, err)

				assert.Equal(t, "ALLOW", config.Rule.Action)
				assert.Equal(t, "remote_ip", config.Rule.Type)
				assert.Equal(t, testCase.expectedCIDRs, config.Rule.Cidrs)
			}
		})
	}
}

func TestAclNeedsToBeEnabled(t *testing.T) {
	for _, testCase := range []struct {
		name         string
		providerType string
		flagEnabled  bool
		aclNil       bool
		cidrs        []string
		expected     bool
	}{
		{
			name:         "OpenStack + flag on + CIDRs present == true",
			providerType: hyperscaler.TypeOpenStack,
			flagEnabled:  true,
			aclNil:       false,
			cidrs:        []string{"10.0.0.1/32"},
			expected:     true,
		},
		{
			name:         "AWS + flag on + CIDRs present == true (regression)",
			providerType: hyperscaler.TypeAWS,
			flagEnabled:  true,
			aclNil:       false,
			cidrs:        []string{"10.0.0.1/32"},
			expected:     true,
		},
		{
			name:         "Azure + flag on + CIDRs present == true (regression)",
			providerType: hyperscaler.TypeAzure,
			flagEnabled:  true,
			aclNil:       false,
			cidrs:        []string{"10.0.0.1/32"},
			expected:     true,
		},
		{
			name:         "GCP == false (still unsupported; guards against accidental enablement)",
			providerType: hyperscaler.TypeGCP,
			flagEnabled:  true,
			aclNil:       false,
			cidrs:        []string{"10.0.0.1/32"},
			expected:     false,
		},
		{
			name:         "OpenStack + flag off == false",
			providerType: hyperscaler.TypeOpenStack,
			flagEnabled:  false,
			aclNil:       false,
			cidrs:        []string{"10.0.0.1/32"},
			expected:     false,
		},
		{
			name:         "OpenStack + empty CIDRs == false",
			providerType: hyperscaler.TypeOpenStack,
			flagEnabled:  true,
			aclNil:       false,
			cidrs:        []string{},
			expected:     false,
		},
		{
			name:         "OpenStack + nil ACL == false",
			providerType: hyperscaler.TypeOpenStack,
			flagEnabled:  true,
			aclNil:       true,
			cidrs:        nil,
			expected:     false,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			runtime := imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Name: "test-shoot",
						Provider: imv1.Provider{
							Type: testCase.providerType,
						},
						Kubernetes: imv1.Kubernetes{
							KubeAPIServer: imv1.APIServer{
								ACL: nil,
							},
						},
					},
				},
			}

			if !testCase.aclNil {
				runtime.Spec.Shoot.Kubernetes.KubeAPIServer.ACL = &imv1.ACL{
					AllowedCIDRs: testCase.cidrs,
				}
			}

			result := AclNeedsToBeEnabled(testCase.flagEnabled, runtime)
			assert.Equal(t, testCase.expected, result)
		})
	}
}
