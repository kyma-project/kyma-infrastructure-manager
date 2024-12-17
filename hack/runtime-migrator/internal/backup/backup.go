package backup

import (
	"context"
	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	authenticationv1alpha1 "github.com/gardener/oidc-webhook-authenticator/apis/authentication/v1alpha1"
	"github.com/kyma-project/infrastructure-manager/hack/runtime-migrator-app/internal/config"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/kubeconfig"
	rbacv1 "k8s.io/api/rbac/v1"
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

func (m Backuper) Do(ctx context.Context, shoot v1beta1.Shoot) (RuntimeBackup, error) {
	return RuntimeBackup{}, nil
}
