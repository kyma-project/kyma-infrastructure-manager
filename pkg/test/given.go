package test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"time"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/e2e-framework/klient"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/envfuncs"
	"sigs.k8s.io/e2e-framework/support/kind"
)

const (
	KCPNamespace     = "kcp-system"
	imageName        = "controller:e2e" //:latest cannot be used, see https://kind.sigs.k8s.io/docs/user/quick-start/#loading-an-image-into-your-cluster
	kimPodTimeoutSec = 15
)

type Given func(clusterName string) *Requirement

type Requirement struct {
	setup  func(ctx context.Context, c *envconf.Config) (context.Context, error)
	finish func(ctx context.Context, c *envconf.Config) (context.Context, error)
	order  int
}

func WithExportOfClusterLogs(clusterName string) *Requirement {
	return &Requirement{
		order: 0,
		setup: nil,
		finish: func(ctx context.Context, c *envconf.Config) (context.Context, error) {
			logDest, err := pathToMakefile() //write logs to the folder of the Makefile
			if err != nil {
				return ctx, err
			}
			logFile := path.Join(logDest, fmt.Sprintf("logs_%s", clusterName))
			return envfuncs.ExportClusterLogs(clusterName, logFile)(ctx, c)
		},
	}
}

func WithKindCluster(clusterName string) *Requirement {
	return &Requirement{
		order:  1,
		setup:  envfuncs.CreateCluster(kind.NewProvider(), clusterName),
		finish: envfuncs.DestroyCluster(clusterName),
	}
}

func WithCRDsInstalled(_ string) *Requirement {
	return &Requirement{
		order: 2,
		setup: func(ctx context.Context, c *envconf.Config) (context.Context, error) {
			if err := callMake(c, exec.Command("make", "install")); err != nil { //install CRD in cluster
				return ctx, err
			}
			return ctx, imv1.AddToScheme(c.Client().Resources().GetScheme()) //add CRD scheme to K8s client
		},
		finish: nil,
	}
}

func WithDockerBuild(clusterName string) *Requirement {
	return &Requirement{
		order: 3,
		setup: func(ctx context.Context, c *envconf.Config) (context.Context, error) {
			if err := callMake(c, exec.Command("make", "docker-build-notest", fmt.Sprintf("IMG=%s", imageName))); err != nil {
				return ctx, err
			}
			return envfuncs.LoadImageToCluster(clusterName, "controller:e2e")(ctx, c)
		},
		finish: nil,
	}
}

func WithKIMDeployed(_ string) *Requirement {
	return &Requirement{
		order: 4,
		setup: func(ctx context.Context, c *envconf.Config) (context.Context, error) {
			//create kcp-system namespace
			ns := &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: KCPNamespace,
				},
			}
			if err := c.Client().Resources(KCPNamespace).Create(ctx, ns); err != nil {
				return ctx, err
			}

			//create gardener credentials (pointoint to local KIND cluster)
			kc, err := os.ReadFile(c.KubeconfigFile())
			if err != nil {
				return ctx, err
			}
			secret := &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "gardener-credentials",
					Namespace: KCPNamespace,
				},
				Data: map[string][]byte{
					"kubeconfig": kc,
				},
			}
			if err := c.Client().Resources(KCPNamespace).Create(ctx, secret); err != nil {
				return ctx, err
			}

			//deploy KIM
			if err := callMake(c, exec.Command("make", "deploy", fmt.Sprintf("IMG=%s", imageName))); err != nil {
				return ctx, err
			}

			return ctx, waitForKIMPod(c.Client())
		},
		finish: nil,
	}
}

func waitForKIMPod(client klient.Client) error {
	for range kimPodTimeoutSec {
		var pods v1.PodList
		if err := client.Resources(KCPNamespace).List(context.TODO(), &pods); err != nil {
			return err
		}
		if len(pods.Items) == 1 {
			return nil
		}
		time.Sleep(time.Second)

	}
	return fmt.Errorf("KIM pod did not become ready with %v seconds", kimPodTimeoutSec)
}

func callMake(c *envconf.Config, cmd *exec.Cmd) error {
	//set kubeconfig of kind cluster
	oldKubeconfig := os.Getenv("KUBECONFIG")
	os.Setenv("KUBECONFIG", c.KubeconfigFile())
	defer func() {
		if oldKubeconfig != "" {
			os.Setenv("KUBECONFIG", oldKubeconfig)
		} else {
			os.Unsetenv("KUBECONFIG")
		}
	}()

	//get working directory
	cwd, err := pathToMakefile()
	if err != nil {
		return err
	}
	cmd.Dir = cwd

	//execute the make command
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to execute make - returned value: %v (%s)", err, output)
	}

	return nil
}

func pathToMakefile() (string, error) {
	var dir string
	//set the working directory to the folder where the Makefile is located
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %v", err)
	}
	dir, err = recursiveFileLookup(cwd)
	if err != nil {
		return "", fmt.Errorf("failed to find KIM's Makefile (lookup started at '%s'): %v", cwd, err)
	}
	return dir, nil
}

func recursiveFileLookup(lookupPath string) (string, error) {
	if _, err := os.Stat(path.Join(lookupPath, "Makefile")); errors.Is(err, os.ErrNotExist) {
		if lookupPath == "/" {
			return "", os.ErrNotExist
		}
		return recursiveFileLookup(path.Dir(lookupPath))
	}
	return lookupPath, nil
}
