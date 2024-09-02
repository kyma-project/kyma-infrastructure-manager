package test

import (
	"context"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/klient"
)

var s *Suite

func TestMain(m *testing.M) {
	s = NewSuite(m, NewEnvConf(""), WithCRDsInstalled, WithKindCluster, WithDockerBuild, WithKIMDeployed, WithExportOfClusterLogs)
	s.Run()
}

func TestKCPSystem(t *testing.T) {
	f := s.NewFeature(t, "Get list of kcp-system pods and check for KIM")

	f.Assert("KCP-system namespace exists", func(client klient.Client) {
		var ns v1.Namespace
		err := client.Resources(KCPNamespace).Get(context.TODO(), "kcp-system", "", &ns)
		assert.NoError(t, err)
		assert.Equal(t, ns.Name, "kcp-system")
	})

	f.Assert("KIM Pod exists", func(client klient.Client) {
		var pods v1.PodList
		err := client.Resources(KCPNamespace).List(context.TODO(), &pods)
		assert.NoError(t, err)
		assert.Len(t, pods.Items, 1)
		assert.Contains(t, pods.Items[0].Name, "infrastructure-manager")
	})
	f.Run()
}

func TestRuntimeCR(t *testing.T) {
	f := s.NewFeature(t, "Compare Runtime CR with Shoot")
	f.WithRuntimeCRs(path.Join("assets", "runtime-example.yaml"))
	f.Assert("Check for the correct Shoot", func(client klient.Client) {
		var ns v1.Namespace
		err := client.Resources(KCPNamespace).Get(context.TODO(), "kcp-system", "", &ns)
		assert.NoError(t, err)
	})
	f.Run()
}

func TestKubeSytem(t *testing.T) {
	f := s.NewFeature(t, "Get list of kube-system pods")
	f.Assert("Kube-system pods exist", func(client klient.Client) {
		var pods v1.PodList
		err := client.Resources("kube-system").List(context.TODO(), &pods)
		assert.NoError(t, err)
		assert.True(t, len(pods.Items) > 5)
	})
	f.Run()
}
