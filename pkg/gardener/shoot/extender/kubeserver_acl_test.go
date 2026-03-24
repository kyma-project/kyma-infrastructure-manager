package extender

import (
	"encoding/json"
	"os"
	"testing"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/config"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	k8s "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
)

func TestNewKubeServerACLExtenderCreate(t *testing.T) {
	t.Run("Should skip when ACL is disabled in config", func(t *testing.T) {
		// given
		shoot := testutils.FixEmptyGardenerShoot("shoot", "kcp-system")
		runtime := fixRuntimeWithACL("aws", []string{"1.2.3.4/32"})
		aclConfig := config.ACL{}

		extender := NewKubeServerACLExtenderCreate(aclConfig, false)

		// when
		err := extender(runtime, &shoot)

		// then
		require.NoError(t, err)
		assert.Empty(t, shoot.Spec.Extensions)
	})

	t.Run("Should skip when ACL object is nil in RuntimeCR", func(t *testing.T) {
		// given
		shoot := testutils.FixEmptyGardenerShoot("shoot", "kcp-system")
		runtime := imv1.Runtime{}
		runtime.Spec.Shoot.Provider.Type = "aws"
		aclConfig := config.ACL{}

		extender := NewKubeServerACLExtenderCreate(aclConfig, true)

		// when
		err := extender(runtime, &shoot)

		// then
		require.NoError(t, err)
		assert.Empty(t, shoot.Spec.Extensions)
	})

	t.Run("Should skip when AllowedCIDRs is empty", func(t *testing.T) {
		// given
		shoot := testutils.FixEmptyGardenerShoot("shoot", "kcp-system")
		runtime := fixRuntimeWithACL("aws", []string{})
		aclConfig := config.ACL{}

		extender := NewKubeServerACLExtenderCreate(aclConfig, true)

		// when
		err := extender(runtime, &shoot)

		// then
		require.NoError(t, err)
		assert.Empty(t, shoot.Spec.Extensions)
	})

	t.Run("Should skip when hyperscaler is GCP", func(t *testing.T) {
		// given
		shoot := testutils.FixEmptyGardenerShoot("shoot", "kcp-system")
		runtime := fixRuntimeWithACL("gcp", []string{"1.2.3.4/32"})
		aclConfig := config.ACL{}

		extender := NewKubeServerACLExtenderCreate(aclConfig, true)

		// when
		err := extender(runtime, &shoot)

		// then
		require.NoError(t, err)
		assert.Empty(t, shoot.Spec.Extensions)
	})

	t.Run("Should create ACL extension on shoot for AWS", func(t *testing.T) {
		// given
		shoot := testutils.FixEmptyGardenerShoot("shoot", "kcp-system")
		runtime := fixRuntimeWithACL("aws", []string{"1.2.3.4/32", "5.6.0.0/16"})
		aclConfig := fixACLConfig(t, `["10.0.0.1/32","10.0.0.2/32"]`, `"172.16.0.1/32"`)

		extender := NewKubeServerACLExtenderCreate(aclConfig, true)

		// when
		err := extender(runtime, &shoot)

		// then
		require.NoError(t, err)
		require.Len(t, shoot.Spec.Extensions, 1)

		ext := shoot.Spec.Extensions[0]
		assert.Equal(t, "acl", ext.Type)
		assert.Equal(t, ptr.To(false), ext.Disabled)

		var providerConfig aclProviderConfig
		require.NoError(t, json.Unmarshal(ext.ProviderConfig.Raw, &providerConfig))

		assert.Equal(t, "ALLOW", providerConfig.Rule.Action)
		assert.Equal(t, "remote_ip", providerConfig.Rule.Type)
		assert.Equal(t, []string{"1.2.3.4/32", "5.6.0.0/16", "10.0.0.1/32", "10.0.0.2/32", "172.16.0.1/32"}, providerConfig.Rule.Cidrs)
	})

	t.Run("Should create ACL extension on shoot for Azure", func(t *testing.T) {
		// given
		shoot := testutils.FixEmptyGardenerShoot("shoot", "kcp-system")
		runtime := fixRuntimeWithACL("azure", []string{"192.168.1.0/24"})
		aclConfig := fixACLConfig(t, `["10.0.0.1/32"]`, `"172.16.0.1/32"`)

		extender := NewKubeServerACLExtenderCreate(aclConfig, true)

		// when
		err := extender(runtime, &shoot)

		// then
		require.NoError(t, err)
		require.Len(t, shoot.Spec.Extensions, 1)

		var providerConfig aclProviderConfig
		require.NoError(t, json.Unmarshal(shoot.Spec.Extensions[0].ProviderConfig.Raw, &providerConfig))

		assert.Equal(t, []string{"192.168.1.0/24", "10.0.0.1/32", "172.16.0.1/32"}, providerConfig.Rule.Cidrs)
	})

	t.Run("Should append ACL extension to existing extensions", func(t *testing.T) {
		// given
		shoot := testutils.FixEmptyGardenerShoot("shoot", "kcp-system")
		shoot.Spec.Extensions = []gardener.Extension{
			{Type: "shoot-cert-service"},
		}
		runtime := fixRuntimeWithACL("aws", []string{"1.2.3.4/32"})
		aclConfig := fixACLConfig(t, `["10.0.0.1/32"]`, `"172.16.0.1/32"`)

		extender := NewKubeServerACLExtenderCreate(aclConfig, true)

		// when
		err := extender(runtime, &shoot)

		// then
		require.NoError(t, err)
		require.Len(t, shoot.Spec.Extensions, 2)
		assert.Equal(t, "shoot-cert-service", shoot.Spec.Extensions[0].Type)
		assert.Equal(t, "acl", shoot.Spec.Extensions[1].Type)
	})

	t.Run("Should return error when operator IP file does not exist", func(t *testing.T) {
		// given
		shoot := testutils.FixEmptyGardenerShoot("shoot", "kcp-system")
		runtime := fixRuntimeWithACL("aws", []string{"1.2.3.4/32"})
		aclConfig := config.ACL{
			IpAddressesPath: "dir/missing.json",
			KcpAddressPath:  "dir/kcp-ip.json",
		}

		extender := NewKubeServerACLExtenderCreate(aclConfig, true)

		// when
		err := extender(runtime, &shoot)

		// then
		require.Error(t, err)
		assert.Empty(t, shoot.Spec.Extensions)
	})

	t.Run("Should return error when KCP IP file does not exist", func(t *testing.T) {
		// given
		shoot := testutils.FixEmptyGardenerShoot("shoot", "kcp-system")
		runtime := fixRuntimeWithACL("aws", []string{"1.2.3.4/32"})

		dir := t.TempDir()
		writeFile(t, dir, "operator-ips.json", `["10.0.0.1/32"]`)

		aclConfig := config.ACL{
			IpAddressesPath: dir + "/operator-ips.json",
			KcpAddressPath:  dir + "/missing.json",
		}

		extender := NewKubeServerACLExtenderCreate(aclConfig, true)

		// when
		err := extender(runtime, &shoot)

		// then
		require.Error(t, err)
		assert.Empty(t, shoot.Spec.Extensions)
	})

	t.Run("Should return error when operator IP file contains invalid JSON", func(t *testing.T) {
		// given
		shoot := testutils.FixEmptyGardenerShoot("shoot", "kcp-system")
		runtime := fixRuntimeWithACL("aws", []string{"1.2.3.4/32"})

		dir := t.TempDir()
		writeFile(t, dir, "operator-ips.json", `not-valid-json`)
		writeFile(t, dir, "kcp-ip.json", `"172.16.0.1/32"`)

		aclConfig := config.ACL{
			IpAddressesPath: dir + "/operator-ips.json",
			KcpAddressPath:  dir + "/kcp-ip.json",
		}

		extender := NewKubeServerACLExtenderCreate(aclConfig, true)

		// when
		err := extender(runtime, &shoot)

		// then
		require.Error(t, err)
		assert.Empty(t, shoot.Spec.Extensions)
	})
}

func TestNewKubeServerACLExtenderPatch(t *testing.T) {
	t.Run("Should skip when ACL is disabled in config and no existing ACL", func(t *testing.T) {
		// given
		shoot := testutils.FixEmptyGardenerShoot("shoot", "kcp-system")
		runtime := fixRuntimeWithACL("aws", []string{"1.2.3.4/32"})
		aclConfig := config.ACL{}

		extender := NewKubeServerACLExtenderPatch(aclConfig, false)

		// when
		err := extender(runtime, &shoot)

		// then
		require.NoError(t, err)
		assert.Empty(t, shoot.Spec.Extensions)
	})

	t.Run("Should leave existing ACL when ACL is disabled in config", func(t *testing.T) {
		// given
		shoot := testutils.FixEmptyGardenerShoot("shoot", "kcp-system")
		shoot.Spec.Extensions = []gardener.Extension{
			{Type: "acl"},
			{Type: "shoot-cert-service"},
		}
		runtime := fixRuntimeWithACL("aws", []string{"1.2.3.4/32"})
		aclConfig := config.ACL{}

		extender := NewKubeServerACLExtenderPatch(aclConfig, false)

		// when
		err := extender(runtime, &shoot)

		// then
		require.NoError(t, err)
		require.Len(t, shoot.Spec.Extensions, 2)
		assert.Equal(t, "acl", shoot.Spec.Extensions[0].Type)
		assert.Equal(t, "shoot-cert-service", shoot.Spec.Extensions[1].Type)
	})

	t.Run("Should skip when hyperscaler is GCP and no existing ACL", func(t *testing.T) {
		// given
		shoot := testutils.FixEmptyGardenerShoot("shoot", "kcp-system")
		runtime := fixRuntimeWithACL("gcp", []string{"1.2.3.4/32"})
		aclConfig := config.ACL{}

		extender := NewKubeServerACLExtenderPatch(aclConfig, true)

		// when
		err := extender(runtime, &shoot)

		// then
		require.NoError(t, err)
		assert.Empty(t, shoot.Spec.Extensions)
	})

	t.Run("Should leave existing ACL when hyperscaler is GCP", func(t *testing.T) {
		// given
		shoot := testutils.FixEmptyGardenerShoot("shoot", "kcp-system")
		shoot.Spec.Extensions = []gardener.Extension{
			{Type: "shoot-cert-service"},
			{Type: "acl"},
		}
		runtime := fixRuntimeWithACL("gcp", []string{"1.2.3.4/32"})
		aclConfig := config.ACL{}

		extender := NewKubeServerACLExtenderPatch(aclConfig, true)

		// when
		err := extender(runtime, &shoot)

		// then
		require.NoError(t, err)
		require.Len(t, shoot.Spec.Extensions, 2)
		assert.Equal(t, "shoot-cert-service", shoot.Spec.Extensions[0].Type)
		assert.Equal(t, "acl", shoot.Spec.Extensions[1].Type)
	})

	t.Run("Should remove ACL extension when ACL object is nil in RuntimeCR", func(t *testing.T) {
		// given
		shoot := testutils.FixEmptyGardenerShoot("shoot", "kcp-system")
		shoot.Spec.Extensions = []gardener.Extension{
			{Type: "acl"},
			{Type: "shoot-cert-service"},
		}
		runtime := imv1.Runtime{}
		runtime.Spec.Shoot.Provider.Type = "aws"
		aclConfig := config.ACL{}

		extender := NewKubeServerACLExtenderPatch(aclConfig, true)

		// when
		err := extender(runtime, &shoot)

		// then
		require.NoError(t, err)
		require.Len(t, shoot.Spec.Extensions, 1)
		assert.Equal(t, "shoot-cert-service", shoot.Spec.Extensions[0].Type)
	})

	t.Run("Should remove ACL extension when AllowedCIDRs is empty", func(t *testing.T) {
		// given
		shoot := testutils.FixEmptyGardenerShoot("shoot", "kcp-system")
		shoot.Spec.Extensions = []gardener.Extension{
			{Type: "shoot-cert-service"},
			{Type: "acl"},
		}
		runtime := fixRuntimeWithACL("aws", []string{})
		aclConfig := config.ACL{}

		extender := NewKubeServerACLExtenderPatch(aclConfig, true)

		// when
		err := extender(runtime, &shoot)

		// then
		require.NoError(t, err)
		require.Len(t, shoot.Spec.Extensions, 1)
		assert.Equal(t, "shoot-cert-service", shoot.Spec.Extensions[0].Type)
	})

	t.Run("Should apply ACL extension on shoot for AWS", func(t *testing.T) {
		// given
		shoot := testutils.FixEmptyGardenerShoot("shoot", "kcp-system")
		runtime := fixRuntimeWithACL("aws", []string{"1.2.3.4/32", "5.6.0.0/16"})
		aclConfig := fixACLConfig(t, `["10.0.0.1/32","10.0.0.2/32"]`, `"172.16.0.1/32"`)

		extender := NewKubeServerACLExtenderPatch(aclConfig, true)

		// when
		err := extender(runtime, &shoot)

		// then
		require.NoError(t, err)
		require.Len(t, shoot.Spec.Extensions, 1)

		ext := shoot.Spec.Extensions[0]
		assert.Equal(t, "acl", ext.Type)
		assert.Equal(t, ptr.To(false), ext.Disabled)

		var providerConfig aclProviderConfig
		require.NoError(t, json.Unmarshal(ext.ProviderConfig.Raw, &providerConfig))

		assert.Equal(t, "ALLOW", providerConfig.Rule.Action)
		assert.Equal(t, "remote_ip", providerConfig.Rule.Type)
		assert.Equal(t, []string{"1.2.3.4/32", "5.6.0.0/16", "10.0.0.1/32", "10.0.0.2/32", "172.16.0.1/32"}, providerConfig.Rule.Cidrs)
	})

	t.Run("Should apply ACL extension on shoot for Azure", func(t *testing.T) {
		// given
		shoot := testutils.FixEmptyGardenerShoot("shoot", "kcp-system")
		runtime := fixRuntimeWithACL("azure", []string{"192.168.1.0/24"})
		aclConfig := fixACLConfig(t, `["10.0.0.1/32"]`, `"172.16.0.1/32"`)

		extender := NewKubeServerACLExtenderPatch(aclConfig, true)

		// when
		err := extender(runtime, &shoot)

		// then
		require.NoError(t, err)
		require.Len(t, shoot.Spec.Extensions, 1)

		var providerConfig aclProviderConfig
		require.NoError(t, json.Unmarshal(shoot.Spec.Extensions[0].ProviderConfig.Raw, &providerConfig))

		assert.Equal(t, []string{"192.168.1.0/24", "10.0.0.1/32", "172.16.0.1/32"}, providerConfig.Rule.Cidrs)
	})

	t.Run("Should replace existing ACL extension with new one", func(t *testing.T) {
		// given
		shoot := testutils.FixEmptyGardenerShoot("shoot", "kcp-system")
		oldACLExtension := aclProviderConfig{Rule: aclRule{Action: "ALLOW", Cidrs: []string{"8.8.8.8/32"}, Type: "remote_ip"}}
		oldRawExtension, _ := json.Marshal(oldACLExtension)
		shoot.Spec.Extensions = []gardener.Extension{
			{Type: "shoot-cert-service"},
			{
				Type:           "acl",
				ProviderConfig: &k8s.RawExtension{Raw: oldRawExtension},
				Disabled:       ptr.To(false),
			},
		}
		runtimeCR := fixRuntimeWithACL("aws", []string{"1.2.3.4/32"})
		aclConfig := fixACLConfig(t, `["10.0.0.1/32"]`, `"172.16.0.1/32"`)

		extender := NewKubeServerACLExtenderPatch(aclConfig, true)

		// when
		err := extender(runtimeCR, &shoot)

		// then
		require.NoError(t, err)
		require.Len(t, shoot.Spec.Extensions, 2)
		assert.Equal(t, "shoot-cert-service", shoot.Spec.Extensions[0].Type)
		assert.Equal(t, "acl", shoot.Spec.Extensions[1].Type)

		var providerConfig aclProviderConfig
		require.NoError(t, json.Unmarshal(shoot.Spec.Extensions[1].ProviderConfig.Raw, &providerConfig))
		assert.Equal(t, []string{"1.2.3.4/32", "10.0.0.1/32", "172.16.0.1/32"}, providerConfig.Rule.Cidrs)
		assert.NotContains(t, providerConfig.Rule.Cidrs, "8.8.8.8/32")
	})
}

func fixRuntimeWithACL(providerType string, cidrs []string) imv1.Runtime {
	return imv1.Runtime{
		Spec: imv1.RuntimeSpec{
			Shoot: imv1.RuntimeShoot{
				Provider: imv1.Provider{
					Type: providerType,
				},
				Kubernetes: imv1.Kubernetes{
					KubeAPIServer: imv1.APIServer{
						ACL: &imv1.ACL{
							AllowedCIDRs: cidrs,
						},
					},
				},
			},
		},
	}
}

func fixACLConfig(t *testing.T, operatorIPsJSON, kcpIPJSON string) config.ACL {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, dir, "operator-ips.json", operatorIPsJSON)
	writeFile(t, dir, "kcp-ip.json", kcpIPJSON)
	return config.ACL{
		IpAddressesPath: dir + "/operator-ips.json",
		KcpAddressPath:  dir + "/kcp-ip.json",
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	err := os.WriteFile(dir+"/"+name, []byte(content), 0644)
	require.NoError(t, err)
}
