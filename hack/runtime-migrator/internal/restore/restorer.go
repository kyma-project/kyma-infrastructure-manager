package restore

import (
	"fmt"
	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	authenticationv1alpha1 "github.com/gardener/oidc-webhook-authenticator/apis/authentication/v1alpha1"
	"github.com/kyma-project/infrastructure-manager/hack/runtime-migrator-app/internal/backup"
	rbacv1 "k8s.io/api/rbac/v1"
	"os"
	"path"
	"sigs.k8s.io/yaml"
)

type Restorer struct {
	backupDir string
}

func NewRestorer(backupDir string) Restorer {
	return Restorer{
		backupDir: backupDir,
	}
}

func (r Restorer) Do(runtimeID string, shootName string) (backup.RuntimeBackup, error) {
	shoot, err := r.getShootToRestore(runtimeID, shootName)
	if err != nil {
		return backup.RuntimeBackup{}, err
	}

	crbs, err := r.getCRBsToRestore(runtimeID)
	if err != nil {
		return backup.RuntimeBackup{}, err
	}

	oidcConfig, err := r.getOIDCConfigToRestore(runtimeID)
	if err != nil {
		return backup.RuntimeBackup{}, err
	}

	return backup.RuntimeBackup{
		ShootToRestore:      shoot,
		ClusterRoleBindings: crbs,
		OIDCConfig:          oidcConfig,
	}, nil
}

func (r Restorer) getShootToRestore(runtimeID string, shootName string) (v1beta1.Shoot, error) {
	shootFilePath := path.Join(r.backupDir, fmt.Sprintf("backup/%s/%s-to-restore.yaml", runtimeID, shootName))

	shoot, err := restoreFromFile[v1beta1.Shoot](shootFilePath)
	if err != nil {
		return v1beta1.Shoot{}, err
	}
	shoot.Kind = "Shoot"
	shoot.APIVersion = "core.gardener.cloud/v1beta1"

	return *shoot, nil
}

func (r Restorer) getCRBsToRestore(runtimeID string) ([]rbacv1.ClusterRoleBinding, error) {
	crbsDir := path.Join("%s/%s/crb", r.backupDir, runtimeID)
	entries, err := os.ReadDir(crbsDir)

	if err != nil {
		return nil, err
	}

	crbs := make([]rbacv1.ClusterRoleBinding, 0)

	for _, entry := range entries {
		crbFilePath := entry.Name()

		crb, err := restoreFromFile[rbacv1.ClusterRoleBinding](crbFilePath)
		if err != nil {
			return nil, err
		}

		crbs = append(crbs, *crb)
	}

	return crbs, nil
}

func (r Restorer) getOIDCConfigToRestore(runtimeID string) ([]authenticationv1alpha1.OpenIDConnect, error) {
	crbsDir := path.Join("%s/%s/oidc", r.backupDir, runtimeID)
	entries, err := os.ReadDir(crbsDir)

	if err != nil {
		return nil, err
	}

	crbs := make([]authenticationv1alpha1.OpenIDConnect, 0)

	for _, entry := range entries {
		crbFilePath := entry.Name()

		crb, err := restoreFromFile[authenticationv1alpha1.OpenIDConnect](crbFilePath)
		if err != nil {
			return nil, err
		}

		crbs = append(crbs, *crb)
	}

	return crbs, nil
}

func restoreFromFile[T any](filePath string) (*T, error) {
	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var obj T

	err = yaml.Unmarshal(fileBytes, &obj)
	if err != nil {
		return nil, err
	}

	return &obj, nil
}
