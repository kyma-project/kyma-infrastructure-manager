package e2e

import (
	"context"
	"fmt"
	testutils "github.com/kyma-project/infrastructure-manager/test/utils"
	"log"
	"os"
	"os/exec"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/envfuncs"
	"sigs.k8s.io/e2e-framework/pkg/utils"
	"sigs.k8s.io/e2e-framework/third_party/k3d"

	"testing"
	"time"
)

const (
	setupTimeout  = 3 * time.Minute
	setupInterval = 5 * time.Second
)

var (
	testEnv env.Environment

	dockerImage  = "kyma-infrastructure-manage:local"
	kustomizeVer = "v5.5.0"
	ctrlGenVer   = "v0.16.5"

	certMgrVer        = "v1.16.3"
	certMgrUrl        = fmt.Sprintf("https://github.com/cert-manager/cert-manager/releases/download/%s/cert-manager.yaml", certMgrVer)
	promVer           = "v0.77.1"
	promUrl           = fmt.Sprintf("https://github.com/prometheus-operator/prometheus-operator/releases/download/%s/bundle.yaml", promVer)
	setupResourcesDir = "resources/setup"
	namespace         = "kcp-system"
)

func TestMain(m *testing.M) {
	testEnv = env.New()
	k3dClusterName := "kim-e2e-k3d-cluster"
	k3dCluster := k3d.NewCluster(k3dClusterName)

	log.Print("Create k3d cluster...")
	testEnv.Setup(
		envfuncs.CreateCluster(k3dCluster, k3dClusterName),
		envfuncs.CreateNamespace(namespace),

		// build and load the Docker image of KIM into the k3d cluster
		func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
			log.Printf("Building and loading Docker image %s...", dockerImage)
			if err := testutils.CallMake(exec.Command("make", "docker-build", fmt.Sprintf("IMG=%s", dockerImage))); err != nil {
				log.Printf("Failed to build Docker image: %s", err)
				return ctx, err
			}

			if p := utils.RunCommand(fmt.Sprintf("k3d image import %s -c %s", dockerImage, k3dClusterName)); p.Err() != nil {
				log.Printf("Failed to load Docker image into k3d cluster: %s: %s", p.Err(), p.Out())
				return ctx, p.Err()
			}
			log.Printf("Docker image loaded to k3d cluster...")
			return ctx, nil
		},

		// install CertManager and Prometheus Operator
		func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
			log.Println("Installing prometheus operator...")
			if p := utils.RunCommand(fmt.Sprintf("kubectl apply --server-side -f %s", promUrl)); p.Err() != nil {
				log.Printf("Failed to deploy prometheus: %s: %s", p.Err(), p.Out())
				return ctx, p.Err()
			}

			log.Println("Installing cert-manager...")
			client := cfg.Client()

			if p := utils.RunCommand(fmt.Sprintf("kubectl apply -f %s", certMgrUrl)); p.Err() != nil {
				log.Printf("Failed to deploy cert-manager: %s: %s", p.Err(), p.Out())
				return ctx, p.Err()
			}

			log.Println("Waiting for cert-manager deployment to be available...")
			if err := wait.For(
				conditions.New(client.Resources()).DeploymentAvailable("cert-manager-webhook", "cert-manager"),
				wait.WithTimeout(setupTimeout),
				wait.WithInterval(setupInterval),
			); err != nil {
				log.Printf("Timedout while waiting for cert-manager deployment: %s", err)
				return ctx, err
			}
			return ctx, nil
		},

		// install test tool dependencies
		func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
			log.Println("Installing bin tools...")
			if p := utils.RunCommand(fmt.Sprintf("go install sigs.k8s.io/kustomize/kustomize/v5@%s", kustomizeVer)); p.Err() != nil {
				log.Printf("Failed to install kustomize binary: %s: %s", p.Err(), p.Out())
				return ctx, p.Err()
			}
			if p := utils.RunCommand(fmt.Sprintf("go install sigs.k8s.io/controller-tools/cmd/controller-gen@%s", ctrlGenVer)); p.Err() != nil {
				log.Printf("Failed to install controller-gen binary: %s: %s", p.Err(), p.Out())
				return ctx, p.Err()
			}
			return ctx, nil
		},

		// install all the required ConfigMaps and Secrets for KIM controller
		func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
			log.Println("Installing ConfigMaps and Secrets for KIM controller...")
			if p := utils.RunCommand(fmt.Sprintf("kubectl apply -f %s ", setupResourcesDir)); p.Err() != nil {
				log.Printf("Failed to create ConfigMap and Secret for KIM controller: %s: %s", p.Err(), p.Out())
				return ctx, p.Err()
			}

			return ctx, nil
		},

		func(ctx context.Context, config *envconf.Config) (context.Context, error) {
			// make manifests
			log.Println("Generate manifests...")
			if err := testutils.CallMake(exec.Command("make", "manifests")); err != nil {
				log.Printf("Failed to generate manifests: %s", err)
				return ctx, err
			}

			// make generate
			log.Println("Generate API objects...")
			if err := testutils.CallMake(exec.Command("make", "generate")); err != nil {
				log.Printf("Failed to generate API objects: %s", err)
				return ctx, err
			}
			return ctx, nil
		},

		// install CRDs into the K8s cluster using Makefile
		func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
			log.Println("Installing CRDs for KIM controller...")
			if err := testutils.CallMake(exec.Command("make", "install")); err != nil {
				log.Printf("Failed to install CRDs: %s", err)
				return ctx, err
			}
			return ctx, nil
		},

		// deploy the KIM controller
		func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
			log.Println("Deploying KIM controller...")
			if err := testutils.CallMake(exec.Command("make", "deploy", fmt.Sprintf("IMG=%s", dockerImage))); err != nil {
				log.Printf("Failed to deploy KIM on k3d cluster: %s", err)
				return ctx, err
			}
			return ctx, nil
		},

		// wait for KIM controller to be ready
		func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
			log.Println("Waiting for KIM deployment to be available...")
			client := cfg.Client()
			if err := wait.For(
				conditions.New(client.Resources()).DeploymentAvailable("infrastructure-manager-infrastructure-manager", "kcp-system"),
				wait.WithTimeout(setupTimeout),
				wait.WithInterval(setupInterval),
			); err != nil {
				log.Printf("Timed out while waiting for KIM deployment: %s", err)
				return ctx, err
			}

			return ctx, nil
		},
	)

	testEnv.Finish(
		envfuncs.DestroyCluster(k3dClusterName),
	)

	os.Exit(testEnv.Run(m))
}
