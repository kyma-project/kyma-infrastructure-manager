package main

import (
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/hack/runtime-cleanup/cmd/cleaner"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"log/slog"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func main() {
	textHandler := slog.NewTextHandler(os.Stdout, nil)
	log := slog.New(textHandler)
	log.Info("Starting runtime cleaner")

	k8sClient, err := createKubernetesClient()

	if err != nil {
		log.Error("Error during creating k8s client ", err)
		return
	}

	runtimeCleaner := cleaner.NewRuntimeCleaner(k8sClient, log)
	err = runtimeCleaner.Execute()
	if err != nil {
		log.Error("Error during running runtime cleanup ", err)
	}
	return
}

func createKubernetesClient() (client.Client, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	scheme := runtime.NewScheme()
	err = imv1.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}

	return client.New(config, client.Options{Scheme: scheme})
}
