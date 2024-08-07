package extender

import (
	"testing"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apimachineryRuntime "k8s.io/apimachinery/pkg/runtime"
)

func TestCertConfigExtender(t *testing.T) {
	t.Run("Extend with cert-config", func(t *testing.T) {
		// given
		shoot := fixEmptyGardenerShoot("shoot", "kcp-system")
		runtime := imv1.Runtime{
			ObjectMeta: v1.ObjectMeta{
				Name:      "runtime",
				Namespace: "namespace",
				Labels: map[string]string{
					"kyma-project.io/global-account-id": "global-account-id",
					"kyma-project.io/subaccount-id":     "subaccount-id",
				},
			},
		}

		// when
		err := ExtendWithCertConfig(runtime, &shoot)
		require.NoError(t, err)

		expectedCertConfig := gardener.Extension{
			Type: "shoot-cert-service",
			ProviderConfig: &apimachineryRuntime.RawExtension{
				Raw: []byte(`{"apiVersion":"service.cert.extensions.gardener.cloud/v1alpha1","shootIssuers":{"enabled":true},"kind":"CertConfig"}`),
			},
			Disabled: nil,
		}

		extensions := []gardener.Extension{expectedCertConfig}

		// then
		assert.Equal(t, extensions, shoot.Spec.Extensions)
	})
}
