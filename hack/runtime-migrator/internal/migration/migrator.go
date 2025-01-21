package migration

import (
	"context"
	"fmt"
	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/hack/runtime-migrator-app/internal/initialisation"
	shootutil "github.com/kyma-project/infrastructure-manager/hack/runtime-migrator-app/internal/shoot"
	"github.com/kyma-project/infrastructure-manager/pkg/config"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/kubeconfig"
	"github.com/pkg/errors"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"log/slog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"slices"
)

const (
	migratorLabel                      = "operator.kyma-project.io/created-by-migrator"
	ShootNetworkingFilterExtensionType = "shoot-networking-filter"
)

type Migrator struct {
	cfg                initialisation.Config
	converterConfig    config.ConverterConfig
	kubeconfigProvider kubeconfig.Provider
	kcpClient          client.Client
}

func NewMigrator(cfg initialisation.Config, kubeconfigProvider kubeconfig.Provider, kcpClient client.Client) Migrator {
	return Migrator{
		cfg:                cfg,
		kubeconfigProvider: kubeconfigProvider,
		kcpClient:          kcpClient,
	}
}

func (m Migrator) Do(ctx context.Context, shoot v1beta1.Shoot) (v1.Runtime, error) {
	subjects, err := processAdministrators(ctx, m.kubeconfigProvider, shoot.Name, m.cfg.IsDryRun)

	if err != nil {
		return v1.Runtime{}, err
	}

	var oidcConfig = getOidcConfig(shoot)
	var licenceType = shoot.Annotations["kcp.provisioner.kyma-project.io/licence-type"]
	labels, err := getAllRuntimeLabels(ctx, shoot, m.kcpClient)
	if err != nil {
		return v1.Runtime{}, err
	}
	var isShootNetworkFilteringEnabled = checkIfShootNetworkFilteringEnabled(shoot)

	var runtime = v1.Runtime{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Runtime",
			APIVersion: "infrastructuremanager.kyma-project.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      labels["kyma-project.io/runtime-id"],
			Namespace: "kcp-system",
			Labels:    labels,
		},
		Spec: v1.RuntimeSpec{
			Shoot: v1.RuntimeShoot{
				Name:              shoot.Name,
				Purpose:           *shoot.Spec.Purpose,
				Region:            shoot.Spec.Region,
				LicenceType:       &licenceType,
				SecretBindingName: *shoot.Spec.SecretBindingName,
				Kubernetes: v1.Kubernetes{
					Version: &shoot.Spec.Kubernetes.Version,
					KubeAPIServer: v1.APIServer{
						OidcConfig:           oidcConfig,
						AdditionalOidcConfig: &[]v1beta1.OIDCConfig{oidcConfig},
					},
				},
				Provider: v1.Provider{
					Type:                 shoot.Spec.Provider.Type,
					Workers:              shootutil.FilterOutFields(shoot.Spec.Provider.Workers),
					ControlPlaneConfig:   shoot.Spec.Provider.ControlPlaneConfig,
					InfrastructureConfig: shoot.Spec.Provider.InfrastructureConfig,
				},
				Networking: v1.Networking{
					Type:     shoot.Spec.Networking.Type,
					Pods:     *shoot.Spec.Networking.Pods,
					Nodes:    *shoot.Spec.Networking.Nodes,
					Services: *shoot.Spec.Networking.Services,
				},
			},
			Security: v1.Security{
				Administrators: subjects,
				Networking: v1.NetworkingSecurity{
					Filter: v1.Filter{
						Ingress: &v1.Ingress{
							// deliberately left empty for now, as it was a feature implemented in the Provisioner
						},
						Egress: v1.Egress{
							Enabled: isShootNetworkFilteringEnabled,
						},
					},
				},
			},
		},
	}

	controlPlane := getControlPlane(shoot)
	if controlPlane != nil {
		runtime.Spec.Shoot.ControlPlane = controlPlane
	}

	return runtime, nil
}

func processAdministrators(ctx context.Context, provider kubeconfig.Provider, shootName string, isDryRun bool) ([]string, error) {
	var clusterKubeconfig, err = provider.Fetch(ctx, shootName)
	if clusterKubeconfig == "" {
		return []string{}, err
	}

	restClientConfig, err := clientcmd.RESTConfigFromKubeConfig([]byte(clusterKubeconfig))
	if err != nil {
		return []string{}, err
	}

	clientset, err := kubernetes.NewForConfig(restClientConfig)
	if err != nil {
		return []string{}, err
	}

	clusterRoleBindings, err := clientset.RbacV1().ClusterRoleBindings().List(ctx, metav1.ListOptions{
		LabelSelector: "reconciler.kyma-project.io/managed-by=reconciler,app=kyma",
	})

	if err != nil {
		return []string{}, err
	}

	filteredCRBs := filterOnlySupportedTypesOfCRBs(clusterRoleBindings.Items)

	if len(filteredCRBs) == 0 {
		return []string{}, nil
	}

	if !isDryRun {
		err := labelsCRBsAsDeprecated(ctx, clientset, filteredCRBs)
		if err != nil {
			return []string{}, err
		}
	}

	subjects := getAdministratorsList(filteredCRBs)

	return subjects, nil
}

func filterOnlySupportedTypesOfCRBs(bindings []rbacv1.ClusterRoleBinding) []rbacv1.ClusterRoleBinding {
	return slices.DeleteFunc(bindings, func(clusterRoleBinding rbacv1.ClusterRoleBinding) bool {
		if clusterRoleBinding.RoleRef.Kind != "ClusterRole" || clusterRoleBinding.RoleRef.Name != "cluster-admin" {
			return true
		}
		// leave only cluster-admin CRBs where at least one subject is of a user type
		if slices.ContainsFunc(clusterRoleBinding.Subjects, func(subject rbacv1.Subject) bool { return subject.Kind == rbacv1.UserKind }) {
			return false
		}
		return true
	})
}

func getAdministratorsList(bindings []rbacv1.ClusterRoleBinding) []string {
	subjects := make([]string, 0)
	for _, clusterRoleBinding := range bindings {
		for _, subject := range clusterRoleBinding.Subjects {
			// We are interested only in users
			if subject.Kind == rbacv1.UserKind && !slices.Contains(subjects, subject.Name) {
				subjects = append(subjects, subject.Name)
			}
		}
	}
	return subjects
}

func labelsCRBsAsDeprecated(ctx context.Context, clientset *kubernetes.Clientset, deprecatedCRBs []rbacv1.ClusterRoleBinding) error {
	for _, clusterRoleBinding := range deprecatedCRBs {
		clusterRoleBinding.ObjectMeta.Labels["kyma-project.io/deprecation"] = "to-be-removed-soon"
		_, err := clientset.RbacV1().ClusterRoleBindings().Update(ctx, &clusterRoleBinding, metav1.UpdateOptions{})

		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failed to update ClusterRoleBinding with deprecation label %s", clusterRoleBinding.Name))
		}
		slog.Info(fmt.Sprintf("ClusterRoleBinding %s has been labeled as deprecated", clusterRoleBinding.Name))
	}
	return nil
}

func getOidcConfig(shoot v1beta1.Shoot) v1beta1.OIDCConfig {
	var oidcConfig = v1beta1.OIDCConfig{
		CABundle:             nil, // deliberately left empty
		ClientAuthentication: shoot.Spec.Kubernetes.KubeAPIServer.OIDCConfig.ClientAuthentication,
		ClientID:             shoot.Spec.Kubernetes.KubeAPIServer.OIDCConfig.ClientID,
		GroupsClaim:          shoot.Spec.Kubernetes.KubeAPIServer.OIDCConfig.GroupsClaim,
		GroupsPrefix:         shoot.Spec.Kubernetes.KubeAPIServer.OIDCConfig.GroupsPrefix,
		IssuerURL:            shoot.Spec.Kubernetes.KubeAPIServer.OIDCConfig.IssuerURL,
		RequiredClaims:       shoot.Spec.Kubernetes.KubeAPIServer.OIDCConfig.RequiredClaims,
		SigningAlgs:          shoot.Spec.Kubernetes.KubeAPIServer.OIDCConfig.SigningAlgs,
		UsernameClaim:        shoot.Spec.Kubernetes.KubeAPIServer.OIDCConfig.UsernameClaim,
		UsernamePrefix:       shoot.Spec.Kubernetes.KubeAPIServer.OIDCConfig.UsernamePrefix,
	}

	return oidcConfig
}

func getAllRuntimeLabels(ctx context.Context, shoot v1beta1.Shoot, kcpClient client.Client) (map[string]string, error) {
	enrichedRuntimeLabels := map[string]string{}
	var err error

	gardenerCluster := v1.GardenerCluster{}

	kymaID, found := shoot.Annotations["kcp.provisioner.kyma-project.io/runtime-id"]
	if !found {
		return nil, errors.New("Runtime ID not found in shoot annotations")
	}

	gardenerCRKey := types.NamespacedName{Name: kymaID, Namespace: "kcp-system"}
	getGardenerCRerr := kcpClient.Get(ctx, gardenerCRKey, &gardenerCluster)
	if getGardenerCRerr != nil {
		var errMsg = fmt.Sprintf("Failed to retrieve GardenerCluster CR for shoot %s\n", shoot.Name)
		return map[string]string{}, errors.Wrap(getGardenerCRerr, errMsg)
	}
	enrichedRuntimeLabels["kyma-project.io/broker-plan-id"] = gardenerCluster.Labels["kyma-project.io/broker-plan-id"]
	enrichedRuntimeLabels["kyma-project.io/runtime-id"] = gardenerCluster.Labels["kyma-project.io/runtime-id"]
	enrichedRuntimeLabels["kyma-project.io/subaccount-id"] = gardenerCluster.Labels["kyma-project.io/subaccount-id"]
	enrichedRuntimeLabels["kyma-project.io/broker-plan-name"] = gardenerCluster.Labels["kyma-project.io/broker-plan-name"]
	enrichedRuntimeLabels["kyma-project.io/global-account-id"] = gardenerCluster.Labels["kyma-project.io/global-account-id"]
	enrichedRuntimeLabels["kyma-project.io/instance-id"] = gardenerCluster.Labels["kyma-project.io/instance-id"]
	enrichedRuntimeLabels["kyma-project.io/region"] = gardenerCluster.Labels["kyma-project.io/region"]
	enrichedRuntimeLabels["kyma-project.io/shoot-name"] = gardenerCluster.Labels["kyma-project.io/shoot-name"]
	enrichedRuntimeLabels["operator.kyma-project.io/kyma-name"] = gardenerCluster.Labels["operator.kyma-project.io/kyma-name"]
	// The runtime CR should be controlled by the Provisioner
	enrichedRuntimeLabels["kyma-project.io/controlled-by-provisioner"] = "true"
	// add custom label for the migrator
	enrichedRuntimeLabels[migratorLabel] = "true"

	return enrichedRuntimeLabels, err
}

func getControlPlane(shoot v1beta1.Shoot) *v1beta1.ControlPlane {
	if shoot.Spec.ControlPlane != nil {
		if shoot.Spec.ControlPlane.HighAvailability != nil {
			return &v1beta1.ControlPlane{HighAvailability: &v1beta1.HighAvailability{
				FailureTolerance: v1beta1.FailureTolerance{
					Type: shoot.Spec.ControlPlane.HighAvailability.FailureTolerance.Type,
				},
			},
			}
		}
	}

	return nil
}

func checkIfShootNetworkFilteringEnabled(shoot v1beta1.Shoot) bool {
	for _, extension := range shoot.Spec.Extensions {
		if extension.Type == ShootNetworkingFilterExtensionType {
			if extension.Disabled == nil {
				return true
			}
			return !(*extension.Disabled)
		}
	}
	return false
}
