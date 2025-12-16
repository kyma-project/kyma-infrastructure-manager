package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	gardener_core "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardener_security "github.com/gardener/gardener/pkg/apis/security/v1alpha1"
	coreclientset "github.com/gardener/gardener/pkg/client/core/clientset/versioned"
	securityclientset "github.com/gardener/gardener/pkg/client/security/clientset/versioned"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"log"
)

func main() {

	ctx := context.Background()

	var gardenerKubeconfigPath string
	var gardenerProjectName string
	var dryRun bool

	//Gardener related parameters:
	flag.StringVar(&gardenerKubeconfigPath, "gardener-kubeconfig-path", "/gardener/kubeconfig/kubeconfig", "Path to the kubeconfig file for accessing the Gardener cluster")
	flag.StringVar(&gardenerProjectName, "gardener-project-name", "gardener-project", "Name of the Gardener project which is used for storing Shoot definitions")
	flag.BoolVar(&dryRun, "dry-run", true, "Indicates whether to perform a dry run or actually make changes")
	flag.Parse()

	cfg, err := clientcmd.BuildConfigFromFlags("", gardenerKubeconfigPath)
	if err != nil {
		log.Fatalf("failed to build kubeconfig: %v", err)
	}

	coreGardenerClient, err := coreclientset.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("failed to create gardener client: %v", err)
	}

	securityGardenerClient, err := securityclientset.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("failed to create gardener client: %v", err)
	}

	projectNamespace := "garden-" + gardenerProjectName
	list, err := coreGardenerClient.CoreV1beta1().SecretBindings(projectNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Fatalf("failed to list SecretBindings: %v", err)
	}

	for _, secretBinding := range list.Items {
		fmt.Printf("SecretBinding: %s/%s\n", secretBinding.Namespace, secretBinding.Name)
		credentialBinding, createBindingErr := createCredentialBinding(secretBinding)

		if createBindingErr != nil {
			log.Printf("failed to create CredentialBinding for SecretBinding %s/%s: %v", secretBinding.Namespace, secretBinding.Name, createBindingErr)
			continue // proceed with next SecretBinding
		}

		if dryRun {
			fmt.Printf("Following CredentialBinding would be created: %v\n", credentialBinding)
		} else {
			_, err := securityGardenerClient.SecurityV1alpha1().CredentialsBindings(projectNamespace).Create(ctx, &credentialBinding, metav1.CreateOptions{})
			if err != nil {
				log.Printf("failed to create CredentialBinding for SecretBinding %s/%s: %v", secretBinding.Namespace, secretBinding.Name, err)
			}
			fmt.Printf("CredentialBinding %s/%s created successfully\n", credentialBinding.Namespace, credentialBinding.Name)
		}
	}
}

func createCredentialBinding(secretBinding gardener_core.SecretBinding) (gardener_security.CredentialsBinding, error) { //nolint:staticcheck
	if secretBinding.Provider != nil {
		return gardener_security.CredentialsBinding{}, errors.New("SecretBinding is missing provider type")
	}

	credentialBinding := gardener_security.CredentialsBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "security.gardener.cloud/v1alpha1",
			Kind:       "CredentialsBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        secretBinding.Name,
			Namespace:   secretBinding.Namespace,
			Labels:      secretBinding.Labels,
			Annotations: secretBinding.Annotations,
		},
		CredentialsRef: v1.ObjectReference{
			Kind:       "Secret",
			APIVersion: "v1",
			Namespace:  secretBinding.SecretRef.Namespace,
			Name:       secretBinding.SecretRef.Name,
		},
		Provider: gardener_security.CredentialsBindingProvider{
			Type: secretBinding.Provider.Type,
		},
	}

	return credentialBinding, nil
}
