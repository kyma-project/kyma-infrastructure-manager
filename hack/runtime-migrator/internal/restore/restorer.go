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
	backupDir   string
	restoreCRB  bool
	restoreOIDC bool
}

func NewRestorer(backupDir string, restoreCRB, restoreOIDC bool) Restorer {
	return Restorer{
		backupDir:   backupDir,
		restoreCRB:  restoreCRB,
		restoreOIDC: restoreOIDC,
	}
}

func (r Restorer) Do(runtimeID string, shootName string) (backup.RuntimeBackup, error) {
	shootToRestore, err := r.getShootToRestore(runtimeID, fmt.Sprintf("%s-to-restore", shootName))
	if err != nil {
		return backup.RuntimeBackup{}, err
	}

	originalShoot, err := r.getShootToRestore(runtimeID, fmt.Sprintf("%s-original", shootName))
	if err != nil {
		return backup.RuntimeBackup{}, err
	}

	var crbs []rbacv1.ClusterRoleBinding

	if r.restoreCRB {
		crbsDir := path.Join(r.backupDir, fmt.Sprintf("backup/%s/crb", runtimeID))
		crbs, err = getObjectsFromToRestore[rbacv1.ClusterRoleBinding](crbsDir)
		if err != nil {
			return backup.RuntimeBackup{}, err
		}
	}

	var oidcConfig []authenticationv1alpha1.OpenIDConnect

	if r.restoreOIDC {
		oidcDir := path.Join(r.backupDir, fmt.Sprintf("backup/%s/oidc", runtimeID))
		oidcConfig, err = getObjectsFromToRestore[authenticationv1alpha1.OpenIDConnect](oidcDir)
		if err != nil {
			return backup.RuntimeBackup{}, err
		}
	}

	return backup.RuntimeBackup{
		ShootToRestore:      shootToRestore,
		OriginalShoot:       originalShoot,
		ClusterRoleBindings: crbs,
		OIDCConfig:          oidcConfig,
	}, nil
}

func (r Restorer) getShootToRestore(runtimeID string, shootName string) (v1beta1.Shoot, error) {
	shootFilePath := path.Join(r.backupDir, fmt.Sprintf("backup/%s/%s.yaml", runtimeID, shootName))

	shoot, err := restoreFromFile[v1beta1.Shoot](shootFilePath)
	if err != nil {
		return v1beta1.Shoot{}, err
	}
	shoot.Kind = "Shoot"
	shoot.APIVersion = "core.gardener.cloud/v1beta1"

	return *shoot, nil
}

func getObjectsFromToRestore[T any](dir string) ([]T, error) {
	entries, err := os.ReadDir(dir)

	if err != nil {
		return nil, err
	}

	objects := make([]T, 0)

	for _, entry := range entries {
		filePath := fmt.Sprintf("%s/%s", dir, entry.Name())

		object, err := restoreFromFile[T](filePath)
		if err != nil {
			return nil, err
		}

		objects = append(objects, *object)
	}

	return objects, nil
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
