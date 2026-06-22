package extensions

import (
	"encoding/json"
	"testing"

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
