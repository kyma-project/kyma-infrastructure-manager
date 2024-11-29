package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardener_types "github.com/gardener/gardener/pkg/client/core/clientset/versioned/typed/core/v1beta1"
	gardener_oidc "github.com/gardener/oidc-webhook-authenticator/apis/authentication/v1alpha1"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener"
	"github.com/pkg/errors"
	"k8s.io/client-go/util/flowcontrol"
	"k8s.io/utils/ptr"
	"log/slog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"time"
)

func main() {
	slog.Info("Starting runtime-migrator")

	opts := zap.Options{
		Development: true,
	}
	logger := zap.New(zap.UseFlagOptions(&opts))
	logf.SetLogger(logger)

	var kubeconfigPath string
	flag.StringVar(&kubeconfigPath, "gardener-kubeconfig-path", "", "Path to the kubeconfig file.")

	var gardenerProjectName string
	flag.StringVar(&gardenerProjectName, "gardener-project-name", "", "Name of the Gardener project.")

	var shootName string
	flag.StringVar(&shootName, "shoot-name", "", "Name of the shoot.")

	flag.Parse()

	gardenerNamespace := fmt.Sprintf("garden-%s", gardenerProjectName)
	//shootInterface, err := setupGardenerShootClient(kubeconfigPath, gardenerNamespace)

	//if err != nil {
	//	slog.Error("Failed to setup Gardener shoot client", slog.Any("error", err))
	//	return
	//}

	gardenerClient, err := initGardenerClient(kubeconfigPath, gardenerNamespace, 20*time.Second, 10, 10)
	if err != nil {
		slog.Error("Failed to create Gardener client", slog.Any("error", err))
		return
	}

	var shoot v1beta1.Shoot

	gardenerClient.Get(context.Background(), client.ObjectKey{Name: shootName, Namespace: gardenerNamespace}, &shoot)

	shoot.Spec.DNS = nil
	currentExtensions := shoot.Spec.Extensions
	newExtensions := make([]v1beta1.Extension, 0)

	for _, ext := range currentExtensions {
		if ext.Type != "shoot-dns-service" && ext.Type != "shoot-oidc-service" {
			newExtensions = append(newExtensions, ext)
		}
	}

	shoot.Spec.Extensions = newExtensions

	err = gardenerClient.Patch(context.Background(), &shoot, client.Apply, &client.PatchOptions{
		FieldManager: "shoot-investigator",
		Force:        ptr.To(true),
	})

	if err != nil {
		slog.Error("Failed to patch shoot", slog.Any("error", err))
		return
	}

	//shoot, err := shootInterface.Get(context.Background(), shootName, v1.GetOptions{})
	//if err != nil {
	//	slog.Error("Failed to get shoot", slog.Any("error", err))
	//	return
	//}

	slog.Info("Shoot", slog.Any("shoot", shoot))

}

func setupGardenerShootClient(kubeconfigPath, gardenerNamespace string) (gardener_types.ShootInterface, error) {
	restConfig, err := gardener.NewRestConfigFromFile(kubeconfigPath)
	if err != nil {
		return nil, err
	}

	gardenerClientSet, err := gardener_types.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	shootClient := gardenerClientSet.Shoots(gardenerNamespace)

	return shootClient, nil
}

func initGardenerClient(kubeconfigPath string, namespace string, timeout time.Duration, rlQPS, rlBurst int) (client.Client, error) {
	restConfig, err := gardener.NewRestConfigFromFile(kubeconfigPath)
	if err != nil {
		return nil, err
	}

	restConfig.Timeout = timeout
	restConfig.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(float32(rlQPS), rlBurst)

	gardenerClient, err := client.New(restConfig, client.Options{})
	if err != nil {
		return nil, err
	}

	err = v1beta1.AddToScheme(gardenerClient.Scheme())
	if err != nil {
		return nil, errors.Wrap(err, "failed to register Gardener schema")
	}

	err = gardener_oidc.AddToScheme(gardenerClient.Scheme())
	if err != nil {
		return nil, errors.Wrap(err, "failed to register Gardener schema")
	}

	return gardenerClient, nil
}
