package extensions

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestDNSExtensionsExtender(t *testing.T) {
	for _, testcase := range []struct {
		name         string
		shootName    string
		secretName   string
		prefix       string
		providerType string
	}{
		{
			name:         "Should generate DNS extension for provided external DNS configuration",
			shootName:    "myshoot",
			secretName:   "aws-route53-secret-dev",
			prefix:       "dev.kyma.ondemand.com",
			providerType: "aws-route53",
		},
		{
			name:      "Should generate DNS extension for internal Gardener DNS configuration",
			shootName: "myshoot",
			prefix:    "dev.kyma.ondemand.com",
		},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			_, err := NewDNSExtension(testcase.shootName, testcase.secretName, testcase.prefix, testcase.providerType)
			require.NoError(t, err)

		})
	}

}
