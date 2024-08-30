package test

import (
	"context"
	"testing"

	"sigs.k8s.io/e2e-framework/klient"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/features"

	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

type TestCase struct {
	feature          *features.FeatureBuilder
	testEnv          env.Environment
	clusterName      string
	clusterNamespace string
	t                *testing.T
}

func (tc *TestCase) Assert(desc string, assert func(t *testing.T, k8sClient klient.Client)) *TestCase {
	tc.feature.Assess(desc, func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		assert(t, cfg.Client())
		return ctx
	})
	return tc
}

func (tc *TestCase) Run() {
	tc.testEnv.Test(tc.t, tc.feature.Feature())
}
