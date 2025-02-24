package e2e

import (
	"context"
	"fmt"
	"log"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"strings"
	"testing"
	"time"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	testutils "github.com/kyma-project/infrastructure-manager/test/utils"
	k8s "k8s.io/apimachinery/pkg/api/errors"
	e2ek8s "sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/utils"
)

const (
	testTimeout        = 30 * time.Minute
	testInterval       = 15 * time.Second
	createManifestPath = "resources/runtimes/test-simple-provision.yaml"
	updateManifestPath = "resources/runtimes/test-simple-update.yaml"
	// if changed you need also to update the test RuntimeCR files
	runtimeName = "simple-prov"
)

func TestRuntimeCRUDOperations(t *testing.T) {
	runtimeSignal := make(chan *imv1.Runtime)

	featureProvision := features.New("Runtime provision").Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		client := cfg.Client()
		imv1.AddToScheme(client.Resources(namespace).GetScheme())

		if err := client.Resources(namespace).Watch(&imv1.RuntimeList{}).WithAddFunc(func(obj interface{}) {
			runtime := obj.(*imv1.Runtime)
			if strings.HasPrefix(runtime.Name, runtimeName) {
				runtimeSignal <- runtime
			}
		}).Start(ctx); err != nil {
			t.Fatal(err)
		}

		runtime, err := testutils.CreateRuntimeFromFile(createManifestPath)
		if err != nil {
			t.Fatalf("Failed to create runtime from file: %v", err)
		}
		if err = client.Resources().Create(ctx, runtime); err != nil {
			t.Fatalf("Failed to create runtime: %v", err)
		}

		return ctx
	}).Assess("Runtime is in Ready state after Provision operation", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		waitForRuntimeState(t, cfg, runtimeSignal, imv1.RuntimeStateReady)
		return ctx
	})

	featureUpdate := features.New("Runtime update").Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		client := cfg.Client()
		imv1.AddToScheme(client.Resources(namespace).GetScheme())
		if err := client.Resources(namespace).Watch(&imv1.RuntimeList{}).WithUpdateFunc(func(obj interface{}) {
			runtime := obj.(*imv1.Runtime)
			if strings.HasPrefix(runtime.Name, runtimeName) {
				runtimeSignal <- runtime
			}
		}).Start(ctx); err != nil {
			t.Fatal(err)
		}

		if p := utils.RunCommand(fmt.Sprintf("kubectl apply -f %s", updateManifestPath)); p.Err() != nil {
			log.Printf("Failed apply patch runtime from file using kubectl command: %s: %s", p.Err(), p.Out())
			return ctx
		}

		return ctx
	}).Assess("Runtime is in Ready state after Update operation", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		waitForRuntimeState(t, cfg, runtimeSignal, imv1.RuntimeStateReady)
		return ctx
	})

	featureDelete := features.New("Runtime delete").Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		client := cfg.Client()
		imv1.AddToScheme(client.Resources(namespace).GetScheme())
		if err := client.Resources(namespace).Watch(&imv1.RuntimeList{}).WithDeleteFunc(func(obj interface{}) {
			runtime := obj.(*imv1.Runtime)
			if strings.HasPrefix(runtime.Name, runtimeName) {
				runtimeSignal <- runtime
			}
		}).Start(ctx); err != nil {
			t.Fatal(err)
		}

		if p := utils.RunCommand(fmt.Sprintf("kubectl delete -f %s", updateManifestPath)); p.Err() != nil {
			log.Printf("Failed invoke runtime deletion from file using kubectl command: %s: %s", p.Err(), p.Out())
			return ctx
		}

		return ctx
	}).Assess("Runtime was deleted", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		select {
		case <-time.After(testTimeout):
			t.Error("Timed out waiting for runtime deletion")
		case runtime := <-runtimeSignal:
			if err := wait.For(
				func(ctx context.Context) (done bool, err error) {
					err = cfg.Client().Resources().Get(ctx, runtime.GetName(), runtime.GetNamespace(), &imv1.Runtime{})
					if k8s.IsNotFound(err) {
						return true, nil
					}
					return false, nil
				}, wait.WithTimeout(testTimeout), wait.WithInterval(testInterval),
			); err != nil {
				t.Fatal(err)
			}
		}
		return ctx
	})

	testEnv.Test(t, featureProvision.Feature(), featureUpdate.Feature(), featureDelete.Feature())
}

func waitForRuntimeState(t *testing.T, cfg *envconf.Config, runtimeSignal chan *imv1.Runtime, desiredState imv1.State) {
	select {
	case <-time.After(testTimeout):
		t.Error("Timed out waiting for runtime state")
	case runtime := <-runtimeSignal:
		wait.For(
			conditions.New(cfg.Client().Resources()).ResourceMatch(runtime, func(obj e2ek8s.Object) bool {
				r, ok := obj.(*imv1.Runtime)
				return ok && r.Status.State == desiredState
			}),
			wait.WithTimeout(testTimeout),
			wait.WithInterval(testInterval),
		)
	}
}
