package test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path"
	"testing"

	"sigs.k8s.io/e2e-framework/klient"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envfuncs"
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

func (tc *TestCase) DeployKIM(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
	//set kubeconfig of kind cluster
	oldKubeconfig := os.Getenv("KUBEONFIG")
	os.Setenv("KUBECONFIG", c.KubeconfigFile())
	defer func() {
		if oldKubeconfig != "" {
			os.Setenv("KUBECONFIG", oldKubeconfig)
		} else {
			os.Unsetenv("KUBECONFIG")
		}
	}()

	//prepare the call `make deploy``
	cmd := exec.Command("make", "docker-build", "install", "deploy")

	//set the working directory to the folder where the Makefile is located
	cwd, err := os.Getwd()
	if err != nil {
		t.Logf("Failed to get current working directory: %v", err)
		t.FailNow()
	}
	cmd.Dir, err = pathToMakefile(cwd)
	if err != nil {
		t.Logf("Failed to find KIM's Makefile (lookup started at '%s'): %v", cwd, err)
		t.FailNow()
	}

	//execute the make command
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Logf("Failed to deploy KIM - make returned: %v (%s)", err, output)
		t.FailNow()
	}

	//upload controller image to kind cluster
	envfuncs.LoadImageToCluster(tc.clusterName, "controller:latest")

	return ctx
}

func pathToMakefile(lookupPath string) (string, error) {
	if _, err := os.Stat(path.Join(lookupPath, "Makefile")); errors.Is(err, os.ErrNotExist) {
		if lookupPath == "/" {
			return "", os.ErrNotExist
		}
		return pathToMakefile(path.Dir(lookupPath))
	} else {
		return lookupPath, nil
	}
}

func (tc *TestCase) Given(given ...func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context) *TestCase {
	tc.feature.Setup(func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		for _, givenFct := range given { //call all given fcts
			givenFct(ctx, t, c)
		}
		return ctx
	})
	return tc
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
