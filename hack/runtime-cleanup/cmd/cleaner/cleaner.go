package cleaner

import (
	"context"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

func Execute() error {

	k8sClient, err := createKubernetesClient()

	if err != nil {
		return err
	}

	err = removeOldRuntimes(k8sClient)

	return nil
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

func removeOldRuntimes(client client.Client) error {
	runtimes := &imv1.RuntimeList{}
	if err := client.List(context.Background(), runtimes); err != nil {
		return err
	}

	for _, runtimeObj := range runtimes.Items {
		if runtimeObj.CreationTimestamp.Add(24*time.Hour).Before(time.Now()) && isControlledByKIM(runtimeObj) {
			err := client.Delete(context.Background(), &runtimeObj)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func isControlledByKIM(runtimeObj imv1.Runtime) bool {
	return runtimeObj.Labels["kyma-project.io/controlled-by-provisioner"] == "false"
}
