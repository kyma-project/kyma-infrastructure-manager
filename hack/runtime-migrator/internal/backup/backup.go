package backup

import (
	"context"
	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	authenticationv1alpha1 "github.com/gardener/oidc-webhook-authenticator/apis/authentication/v1alpha1"
	"github.com/kyma-project/infrastructure-manager/hack/runtime-migrator-app/internal/config"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/kubeconfig"
	rbacv1 "k8s.io/api/rbac/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Backuper struct {
	cfg                config.Config
	isDryRun           bool
	kubeconfigProvider kubeconfig.Provider
}

func NewBackuper(isDryRun bool, kubeconfigProvider kubeconfig.Provider) Backuper {
	return Backuper{
		isDryRun:           isDryRun,
		kubeconfigProvider: kubeconfigProvider,
	}
}

type RuntimeBackup struct {
	Shoot               v1beta1.Shoot
	ClusterRoleBindings []rbacv1.ClusterRoleBinding
	OIDCConfig          []authenticationv1alpha1.OpenIDConnect
}

func (b Backuper) Do(ctx context.Context, shoot v1beta1.Shoot) (RuntimeBackup, error) {
	return RuntimeBackup{
		Shoot: b.backupShoot(shoot),
	}, nil
}

func (b Backuper) backupShoot(shootFromGardener v1beta1.Shoot) v1beta1.Shoot {
	return v1beta1.Shoot{
		TypeMeta: v1.TypeMeta{
			Kind:       "Shoot",
			APIVersion: "core.gardener.cloud/v1beta1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:        shootFromGardener.Name,
			Namespace:   shootFromGardener.Namespace,
			Annotations: shootFromGardener.Annotations,
			Labels:      shootFromGardener.Labels,
		},
		Spec: v1beta1.ShootSpec{
			Purpose:           shootFromGardener.Spec.Purpose,
			Region:            shootFromGardener.Spec.Region,
			SecretBindingName: shootFromGardener.Spec.SecretBindingName,
			Networking: &v1beta1.Networking{
				Type:     shootFromGardener.Spec.Networking.Type,
				Nodes:    shootFromGardener.Spec.Networking.Nodes,
				Pods:     shootFromGardener.Spec.Networking.Pods,
				Services: shootFromGardener.Spec.Networking.Services,
			},
			ControlPlane:      shootFromGardener.Spec.ControlPlane,
			CloudProfileName:  shootFromGardener.Spec.CloudProfileName,
			DNS:               shootFromGardener.Spec.DNS,
			Extensions:        shootFromGardener.Spec.Extensions,
			ExposureClassName: shootFromGardener.Spec.ExposureClassName,
			Kubernetes: v1beta1.Kubernetes{
				EnableStaticTokenKubeconfig: shootFromGardener.Spec.Kubernetes.EnableStaticTokenKubeconfig,
				KubeAPIServer: &v1beta1.KubeAPIServerConfig{
					// TODO: consider skipping ClientAuthentication
					OIDCConfig: shootFromGardener.Spec.Kubernetes.KubeAPIServer.OIDCConfig,
				},
				Version: shootFromGardener.Spec.Kubernetes.Version,
			},
			Maintenance: &v1beta1.Maintenance{
				AutoUpdate: shootFromGardener.Spec.Maintenance.AutoUpdate,
			},
			// TODO: consider if we need to do the backup selectively (workers)
			Provider:     shootFromGardener.Spec.Provider,
			Resources:    shootFromGardener.Spec.Resources,
			SeedSelector: shootFromGardener.Spec.SeedSelector,
			// Tolerations is not specified in patch
			//Tolerations:  shootFromGardener.Spec.Tolerations,
		},
	}
}
