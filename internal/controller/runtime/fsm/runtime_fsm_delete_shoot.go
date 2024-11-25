package fsm

import (
	"context"
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func sFnDeleteShoot(ctx context.Context, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	m.log.Info("delete shoot state")

	// wait section
	if !s.shoot.GetDeletionTimestamp().IsZero() {
		m.log.Info("Waiting for shoot to be deleted", "Name", s.shoot.Name, "Namespace", s.shoot.Namespace)
		return requeueAfter(m.RCCfg.RequeueDurationShootDelete)
	}

	// action section
	if !isGardenerCloudDelConfirmationSet(s.shoot.Annotations) {
		m.log.Info("patching shoot with del-confirmation")
		// workaround for Gardener client
		setObjectFields(s.shoot)
		s.shoot.Annotations = addGardenerCloudDelConfirmation(s.shoot.Annotations)

		err := m.ShootClient.Patch(ctx, s.shoot, client.Apply, &client.PatchOptions{
			FieldManager: "kim",
			Force:        ptr.To(true),
		})

		if err != nil {
			m.log.Error(err, "unable to patch shoot:", "Name", s.shoot.Name)
			return requeue()
		}
	}

	m.log.Info("deleting shoot", "Name", s.shoot.Name, "Namespace", s.shoot.Namespace)
	err := m.ShootClient.Delete(ctx, s.shoot)
	if err != nil {
		// action error handler section
		m.log.Error(err, "Failed to delete gardener Shoot")
		s.instance.UpdateStateDeletion(
			imv1.ConditionTypeRuntimeDeprovisioned,
			imv1.ConditionReasonGardenerShootDeleted,
			"False",
			"Gardener API shoot delete error",
		)
	} else {
		s.instance.UpdateStateDeletion(
			imv1.ConditionTypeRuntimeDeprovisioned,
			imv1.ConditionReasonGardenerShootDeleted,
			"Unknown",
			"Runtime shoot deletion started",
		)
	}

	// out section
	return updateStatusAndRequeueAfter(m.RCCfg.RequeueDurationShootDelete)
}

// workaround
func setObjectFields(shoot *gardener.Shoot) {
	shoot.Kind = "Shoot"
	shoot.APIVersion = "core.gardener.cloud/v1beta1"
	shoot.ManagedFields = nil
}

func isGardenerCloudDelConfirmationSet(a map[string]string) bool {
	if len(a) == 0 {
		return false
	}
	val, found := a[imv1.AnnotationGardenerCloudDelConfirmation]
	return found && (val == "true")
}

func addGardenerCloudDelConfirmation(a map[string]string) map[string]string {
	if len(a) == 0 {
		a = map[string]string{}
	}
	a[imv1.AnnotationGardenerCloudDelConfirmation] = "true"
	return a
}
