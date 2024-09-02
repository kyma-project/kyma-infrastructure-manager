package test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/yaml"

	"sigs.k8s.io/e2e-framework/klient"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/features"

	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

type Feature struct {
	feature          *features.FeatureBuilder
	testEnv          env.Environment
	clusterName      string
	clusterNamespace string
	t                *testing.T
}

func (f *Feature) WithRuntimeCRs(runtimeCRFiles ...string) *Feature {
	f.feature.Assess("Installing Runtime CRs", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		for _, runtimeCRFile := range runtimeCRFiles {
			rawRuntimeCR, err := os.ReadFile(runtimeCRFile)
			require.NoError(t, err)

			runtime := &imv1.Runtime{}
			yaml.Unmarshal(rawRuntimeCR, runtime)
			switch filepath.Ext(runtimeCRFile) {
			case ".json":
				require.NoError(t, json.Unmarshal(rawRuntimeCR, runtime))
			case ".yaml", ".yml":
				require.NoError(t, yaml.Unmarshal(rawRuntimeCR, runtime))
			default:
				t.Logf("Cannot read Runtime CR file '%s' because only file extesnion .json or .yaml is supported", runtimeCRFile)
				t.Fail()
			}

			err = cfg.Client().Resources(KCPNamespace).Create(context.TODO(), runtime)
			require.NoError(t, err)
		}
		return ctx
	})
	return f
}

func (f *Feature) Assert(desc string, assertFct func(client klient.Client)) *Feature {
	f.feature.Assess(desc, func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		assertFct(cfg.Client())
		return ctx
	})
	return f
}

func (f *Feature) Run() {
	f.testEnv.Test(f.t, f.feature.Feature())
}
