package main

import (
	"context"
	"fmt"
	gardener_api "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	// gardener_apis "github.com/gardener/gardener/pkg/client/core/clientset/versioned/typed/core/v1beta1"
	gardener_oidc "github.com/gardener/oidc-webhook-authenticator/apis/authentication/v1alpha1"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener"
	"github.com/pkg/errors"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

func main() {
	gardenKubeconfigPath := os.Getenv("GARDENER_KUBECONFIG_PATH")
	//azureShootPatchfile := os.Getenv("AZURE_SHOOT_PATCH_FILE")
	//awsShootPatchfile := os.Getenv("AWS_SHOOT_PATCH_FILE")
	//gcpShootPatchfile := os.Getenv("GCP_SHOOT_PATCH_FILE")

	if gardenKubeconfigPath == "" {
		fmt.Println("Provide GARDENER_KUBECONFIG_PATH variable")
		return
	}

	//awsShootPatchfile := "aws.yaml"
	gpcShootPatchfile := "gcp.yaml"

	//if azureShootPatchfile == "" {
	//	fmt.Println("Provide AZURE_SHOOT_PATCH_FILE variable")
	//	return
	//}
	//
	//if awsShootPatchfile == "" {
	//	fmt.Println("Provide AWS_SHOOT_PATCH_FILE variable")
	//	return
	//}
	//
	//if gcpShootPatchfile == "" {
	//	fmt.Println("Provide GCP_SHOOT_PATCH_FILE variable")
	//	return
	//}

	fmt.Println("All variables are set")

	typedGardenerClient, err := initGardenerClient(gardenKubeconfigPath, "garden-kyma-dev")

	if err != nil {
		fmt.Println("Error while initializing Gardener client: ", err)
		return
	}

	fmt.Println("Gardener client initialized successfully")

	usedNamespace := "garden-kyma-dev"
	fieldMangerName := "kim-tester"

	for _, tc := range map[string]struct {
		patchFilePath     string
		existingShootName string
	}{
		//"azure": {
		//	patchFilePath:     azureShootPatchfile,
		//	existingShootName: "azure-shoot",
		//},
		//"aws": {
		//	patchFilePath:     awsShootPatchfile,
		//	existingShootName: "testme",
		//},
		"gcp": {
			patchFilePath:     gpcShootPatchfile,
			existingShootName: "testgcp2",
		},
	} {
		var existingShoot gardener_api.Shoot

		err := typedGardenerClient.Get(context.Background(), types.NamespacedName{
			Name:      tc.existingShootName,
			Namespace: usedNamespace,
		}, &existingShoot)

		if err != nil {
			fmt.Println("Error getting existing shoot for patch: ", err)
			return
		}

		patchShoot, err := readShootPatchFile(tc.patchFilePath)

		if err != nil {
			fmt.Println("Error while reading patch file: ", err)
			return
		}

		err = typedGardenerClient.Patch(context.Background(), &patchShoot, client.Apply, &client.PatchOptions{
			FieldManager: fieldMangerName,
			Force:        ptr.To(true),
		})

		if err != nil {
			fmt.Println("Error while patching shoot: ", err)
		} else {
			fmt.Println("Shoot patched successfully")
		}
	}
}

func readShootPatchFile(patchfile string) (gardener_api.Shoot, error) {

	var shootStored gardener_api.Shoot

	// read the file to get bytes
	bytes, err := os.ReadFile(patchfile)

	if err != nil {
		return gardener_api.Shoot{}, errors.Wrap(err, "failed to read patch file")
	}

	err = yaml.Unmarshal(bytes, &shootStored)

	if err != nil {
		return gardener_api.Shoot{}, errors.Wrap(err, "cannot unmarshal shoot")
	}

	return shootStored, nil
}

func initGardenerClient(kubeconfigPath string, namespace string) (client.Client, error) {
	restConfig, err := gardener.NewRestConfigFromFile(kubeconfigPath)
	if err != nil {
		return nil, err
	}

	gardenerClient, err := client.New(restConfig, client.Options{})
	if err != nil {
		return nil, err
	}

	err = gardener_api.AddToScheme(gardenerClient.Scheme())
	if err != nil {
		return nil, errors.Wrap(err, "failed to register Gardener schema")
	}

	err = gardener_oidc.AddToScheme(gardenerClient.Scheme())
	if err != nil {
		return nil, errors.Wrap(err, "failed to register Gardener schema")
	}

	return gardenerClient, nil
}
