package test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/klient"
)

var ts *Suite

func TestMain(m *testing.M) {
	ts = NewSuite(m, NewEnvConf(""), WithCRDsInstalled, WithKindCluster, WithDockerBuild, WithKIMDeployed, WithExportOfClusterLogs)
	ts.Run()
}

func TestKCPSystem(t *testing.T) {
	tc := ts.NewScenario(t, "Get list of kcp-system pods and check for KIM")
	tc.Assert("KIM Pod exists", func(t *testing.T, client klient.Client) {
		var pods v1.PodList
		err := client.Resources(KCPNamespace).List(context.TODO(), &pods)
		assert.NoError(t, err)
		assert.Len(t, pods.Items, 1)
		assert.Contains(t, pods.Items[0].Name, "infrastructure-manager")
	})
	tc.Run()
}

func TestKubeSytem(t *testing.T) {
	tc := ts.NewScenario(t, "Get list of kube-system pods")
	tc.Assert("Kube-system pods exist", func(t *testing.T, client klient.Client) {
		var pods v1.PodList
		err := client.Resources("kube-system").List(context.TODO(), &pods)
		assert.NoError(t, err)
		assert.True(t, len(pods.Items) > 5)
	})
	tc.Run()
}
