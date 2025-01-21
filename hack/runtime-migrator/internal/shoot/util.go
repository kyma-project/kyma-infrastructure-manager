package shoot

import (
	"context"
	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardener_types "github.com/gardener/gardener/pkg/client/core/clientset/versioned/typed/core/v1beta1"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

const runtimeIDAnnotation = "kcp.provisioner.kyma-project.io/runtime-id"
const timeoutK8sOperation = 20 * time.Second

func Fetch(ctx context.Context, shootList *v1beta1.ShootList, shootClient gardener_types.ShootInterface, runtimeID string) (*v1beta1.Shoot, error) {
	shoot := findShoot(runtimeID, shootList)
	if shoot == nil {
		return nil, errors.New("shoot was deleted or the runtimeID is incorrect")
	}

	getCtx, cancel := context.WithTimeout(ctx, timeoutK8sOperation)
	defer cancel()

	// We are fetching the shoot from the gardener to make sure the runtime didn't get deleted during the migration process
	refreshedShoot, err := shootClient.Get(getCtx, shoot.Name, v1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, errors.New("shoot was deleted")
		}

		return nil, err
	}

	return refreshedShoot, nil
}

func findShoot(runtimeID string, shootList *v1beta1.ShootList) *v1beta1.Shoot {
	for _, shoot := range shootList.Items {
		if shoot.Annotations[runtimeIDAnnotation] == runtimeID {
			return &shoot
		}
	}
	return nil
}

func IsBeingDeleted(shoot *v1beta1.Shoot) bool {
	return !shoot.DeletionTimestamp.IsZero()
}
