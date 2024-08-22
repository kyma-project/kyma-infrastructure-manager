package test

import (
	"context"
	"os"
	"testing"

	"sigs.k8s.io/e2e-framework/klient"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envfuncs"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"sigs.k8s.io/e2e-framework/pkg/types"
	"sigs.k8s.io/e2e-framework/support/kind"

	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

func NewTestSuite(m *testing.M) *TestSuite {
	clusterName := envconf.RandomName("E2E", 16)
	namespace := envconf.RandomName("", 10)
	cfg, _ := envconf.NewFromFlags()
	e2eTest := &TestSuite{
		testEnv: env.NewWithConfig(cfg),
	}
	e2eTest.testEnv.Setup(
		envfuncs.CreateCluster(kind.NewProvider(), clusterName),
		envfuncs.CreateNamespace(namespace),
	)
	e2eTest.testEnv.Finish(
		envfuncs.DeleteNamespace(namespace),
		envfuncs.DestroyCluster(clusterName),
	)

	os.Exit(e2eTest.testEnv.Run(m))

	return e2eTest
}

type TestSuite struct {
	t           *testing.T
	testEnv     env.Environment
	clusterName string
	namespace   string
	testCases   []*TestCase
}

type TestCase struct {
	feature *features.FeatureBuilder
}

func (ts *TestSuite) NewTestCase(name string) *TestCase {
	tc := &TestCase{
		feature: features.New(name),
	}
	ts.testCases = append(ts.testCases, tc)
	return tc
}

func (ts *TestSuite) Run(t *testing.T) {
	var features []types.Feature
	for _, tc := range ts.testCases {
		features = append(features, tc.feature.Feature())
	}
	ts.testEnv.Test(t, features...)
}

func (tc *TestCase) Assert(desc string, assess func(t *testing.T, k8sClient klient.Client)) {
	tc.feature.Assess(desc, func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		assess(t, cfg.Client())
		return ctx
	})
}
