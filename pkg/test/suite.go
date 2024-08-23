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

func NewTestSuite(m *testing.M, name string) *TestSuite {
	clusterName := envconf.RandomName(name, 25)
	namespace := envconf.RandomName("kcp-system", 10)
	cfg, _ := envconf.NewFromFlags()
	e2eTest := &TestSuite{
		testEnv: env.NewWithConfig(cfg),
		m:       m,
	}
	e2eTest.testEnv.Setup(
		envfuncs.CreateCluster(kind.NewProvider(), clusterName),
		envfuncs.CreateNamespace(namespace),
	)
	e2eTest.testEnv.Finish(
		envfuncs.DeleteNamespace(namespace),
		envfuncs.DestroyCluster(clusterName),
	)
	return e2eTest
}

type TestSuite struct {
	testEnv env.Environment
	m       *testing.M
}

func (ts *TestSuite) NewTestCase(t *testing.T, name string) *TestCase {
	tc := &TestCase{
		feature: features.New(name),
		testEnv: ts.testEnv,
		t:       t,
	}
	return tc
}

func (ts *TestSuite) Run() {
	os.Exit(ts.testEnv.Run(ts.m))
}
