package skrdetails

import (
	"context"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	imv1_client "github.com/kyma-project/infrastructure-manager/internal/controller/runtime/fsm/client"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var GetShootClientPatch = func(ctx context.Context, cnt client.Client, runtime imv1.Runtime) (client.Client, error) {
	runtimeID := runtime.Labels[imv1.LabelKymaRuntimeID]

	secret, err := imv1_client.GetKubeconfigSecret(ctx, cnt, runtimeID, runtime.Namespace)
	if err != nil {
		return nil, err
	}

	restConfig, err := clientcmd.RESTConfigFromKubeConfig(secret.Data[imv1_client.KubeconfigSecretKey])
	if err != nil {
		return nil, err
	}

	shootClientWithAdmin, err := client.New(restConfig, client.Options{})
	if err != nil {
		return nil, err
	}

	return shootClientWithAdmin, nil
}