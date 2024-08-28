package test

import (
	"os"
	"testing"

	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envfuncs"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"sigs.k8s.io/e2e-framework/support/kind"

	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

func NewTestSuite(m *testing.M, name string, kcFile string) *TestSuite {
	clusterName := envconf.RandomName(name, 25)
	namespace := envconf.RandomName("kcp-system", 10)

	var cfg *envconf.Config
	if kcFile == "" {
		cfg, _ = envconf.NewFromFlags()
	} else {
		cfg = envconf.NewWithKubeConfig(kcFile)
	}

	e2eTest := &TestSuite{
		testEnv: env.NewWithConfig(cfg),
		m:       m,
	}
	if kcFile == "" { //create adhoc cluster
		e2eTest.testEnv.Setup(
			envfuncs.CreateCluster(kind.NewProvider(), clusterName),
			envfuncs.CreateNamespace(namespace),
		)
		e2eTest.testEnv.Finish(
			envfuncs.DeleteNamespace(namespace),
			envfuncs.DestroyCluster(clusterName),
		)
	}
	return e2eTest
}

type TestSuite struct {
	testEnv          env.Environment
	m                *testing.M
	clusterName      string
	clusterNamespace string
}

func (ts *TestSuite) NewTestCase(t *testing.T, name string) *TestCase {
	tc := &TestCase{
		feature:          features.New(name),
		testEnv:          ts.testEnv,
		t:                t,
		clusterName:      ts.clusterName,
		clusterNamespace: ts.clusterNamespace,
	}
	return tc
}

func (ts *TestSuite) Run() {
	os.Exit(ts.testEnv.Run(ts.m))
}
