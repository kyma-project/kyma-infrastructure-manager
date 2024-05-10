package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardener_apis "github.com/gardener/gardener/pkg/client/core/clientset/versioned/typed/core/v1beta1"
	"github.com/kyma-project/infrastructure-manager/runtime-migrator/gardener"
	"github.com/kyma-project/infrastructure-manager/runtime-migrator/model"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"os"
	//"gopkg.in/yaml.v3"
	"sigs.k8s.io/yaml"
)

func main() {
	var gardenerKubeconfigPath string
	var gardenerProjectName string
	var outputPath string

	flag.StringVar(&gardenerKubeconfigPath, "gardener-kubeconfig-path", "/gardener/kubeconfig/kubeconfig", "Kubeconfig file for Gardener cluster")
	flag.StringVar(&gardenerProjectName, "gardener-project-name", "gardener-project", "Name of the Gardener project")
	flag.StringVar(&outputPath, "output-path", "", "Path where generated yamls will be saved. Directory has to exist")
	flag.Parse()

	log.Println("gardener-kubeconfig-path:", gardenerKubeconfigPath)
	log.Println("gardener-project-name:", gardenerProjectName)
	log.Println("output-path:", outputPath)

	gardenerNamespace := fmt.Sprintf("garden-%s", gardenerProjectName)

	gardenerShootClient := setupGardenerShootClient(gardenerKubeconfigPath, gardenerNamespace)
	list, err := gardenerShootClient.List(context.Background(), metav1.ListOptions{})

	if err != nil {
		log.Fatal(err)
	}

	for _, shoot := range list.Items {

		var runtime = model.Runtime{
			TypeMeta: metav1.TypeMeta{
				Kind:       "",
				APIVersion: "",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:                       shoot.Name,
				GenerateName:               shoot.GenerateName,
				Namespace:                  shoot.Namespace,
				SelfLink:                   shoot.SelfLink,          //TODO: do we need to fill that?
				UID:                        shoot.UID,               //TODO: do we need to fill that?
				ResourceVersion:            shoot.ResourceVersion,   //TODO: do we need to fill that?
				Generation:                 shoot.Generation,        //TODO: do we need to fill that?
				CreationTimestamp:          shoot.CreationTimestamp, //TODO: do we need to fill that?
				DeletionTimestamp:          shoot.DeletionTimestamp, //TODO: do we need to fill that?
				DeletionGracePeriodSeconds: shoot.DeletionGracePeriodSeconds,
				Labels:                     shoot.Labels,
				Annotations:                shoot.Annotations,
				OwnerReferences:            shoot.OwnerReferences,
				Finalizers:                 shoot.Finalizers,
				ManagedFields:              shoot.ManagedFields, // deliberately left empty
			},
			Spec: model.RuntimeSpec{
				Name:    shoot.Name, //TODO: What to pass here? Should it be the same as ObjectMetadata.Name?
				Purpose: "",         //TODO: fixme
				Kubernetes: model.RuntimeKubernetes{
					Version: nil, //TODO: shoot.Spec.Kubernetes.Version?
					KubeAPIServer: &model.RuntimeAPIServer{
						OidcConfig: v1beta1.OIDCConfig{
							CABundle:             nil,
							ClientAuthentication: shoot.Spec.Kubernetes.KubeAPIServer.OIDCConfig.ClientAuthentication,
							ClientID:             shoot.Spec.Kubernetes.KubeAPIServer.OIDCConfig.ClientID,
							GroupsClaim:          shoot.Spec.Kubernetes.KubeAPIServer.OIDCConfig.GroupsClaim,
							GroupsPrefix:         shoot.Spec.Kubernetes.KubeAPIServer.OIDCConfig.GroupsPrefix,
							IssuerURL:            shoot.Spec.Kubernetes.KubeAPIServer.OIDCConfig.IssuerURL,
							RequiredClaims:       shoot.Spec.Kubernetes.KubeAPIServer.OIDCConfig.RequiredClaims,
							SigningAlgs:          shoot.Spec.Kubernetes.KubeAPIServer.OIDCConfig.SigningAlgs,
							UsernameClaim:        shoot.Spec.Kubernetes.KubeAPIServer.OIDCConfig.UsernameClaim,
							UsernamePrefix:       shoot.Spec.Kubernetes.KubeAPIServer.OIDCConfig.UsernamePrefix,
						},
						AdditionalOidcConfig: nil, //TODO: fixme
					},
				},
				Provider: model.RuntimeProvider{
					Type:              shoot.Spec.Provider.Type,
					Region:            "", //TODO fixme
					SecretBindingName: "", //TODO fixme
				},
				Networking: model.RuntimeSecurityNetworking{
					Filtering: model.RuntimeSecurityNetworkingFiltering{
						Ingress: model.RuntimeSecurityNetworkingFilteringIngress{
							Enabled: false, //TODO: fixme
						},
						Egress: model.RuntimeSecurityNetworkingFilteringEgress{
							Enabled: false, //TODO: fixme
						},
					},
					Administrators: nil, //TODO: fixme
				},
				Workers: nil, //TODO: fixme
			},
			Status: model.RuntimeStatus{
				State:      "",  //TODO: how to determine out status?
				Conditions: nil, //TODO: fixme shoot.Status.Conditions,
			},
		}

		//TODO: generate yaml from Runtime instead of Shoot
		shootAsYaml, err := getYamlSpec(runtime)
		writeSpecToFile(outputPath, shoot, err, shootAsYaml)
	}

}

func getYamlSpec(shoot model.Runtime) ([]byte, error) {
	shootAsYaml, err := yaml.Marshal(shoot)
	return shootAsYaml, err
}

func writeSpecToFile(outputPath string, shoot v1beta1.Shoot, err error, shootAsYaml []byte) {
	var fileName = fmt.Sprintf("%sshoot-%s.yaml", outputPath, shoot.Name)

	err = os.WriteFile(fileName, shootAsYaml, 0644)

	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%s created\n", fileName)
}

func setupGardenerShootClient(kubeconfigPath, gardenerNamespace string) gardener_apis.ShootInterface {
	//TODO: use the same client factory as in the main code
	restConfig, err := gardener.NewRestConfigFromFile(kubeconfigPath)
	if err != nil {
		return nil
	}

	gardenerClientSet, err := gardener_apis.NewForConfig(restConfig)
	if err != nil {
		return nil
	}

	shootClient := gardenerClientSet.Shoots(gardenerNamespace)

	return shootClient
}
