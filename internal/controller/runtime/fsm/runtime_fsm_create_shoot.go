package fsm

import (
	"context"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"time"
)

func sFnCreateShoot(ctx context.Context, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	m.log.Info("Create shoot state")

	newShoot, err := convertShoot(&s.instance, m.ConverterConfig)
	if err != nil {
		m.log.Error(err, "Failed to convert Runtime instance to shoot object")
		return updateStatePendingWithErrorAndStop(&s.instance, imv1.ConditionTypeRuntimeProvisioned, imv1.ConditionReasonConversionError, "Runtime conversion error")
	}

	if s.instance.Annotations == nil {
		s.instance.Annotations = make(map[string]string)
	}

	if _, found := s.instance.Annotations[imv1.AnnotationRuntimeOperationStarted]; !found || s.instance.Annotations[imv1.AnnotationRuntimeOperationStarted] == "" {
		s.instance.Annotations[imv1.AnnotationRuntimeOperationStarted] = time.Now().UTC().Format(time.RFC3339)
		m.Update(ctx, &s.instance)
		return requeue()
	}

	err = m.ShootClient.Create(ctx, &newShoot)

	if err != nil {
		m.log.Error(err, "Failed to create new gardener Shoot")

		s.instance.UpdateStatePending(
			imv1.ConditionTypeRuntimeProvisioned,
			imv1.ConditionReasonGardenerError,
			"False",
			"Gardener API create error",
		)
		return updateStatusAndRequeueAfter(gardenerRequeueDuration)
	}
	m.log.Info("Gardener shoot for runtime initialised successfully", "Name", newShoot.Name, "Namespace", newShoot.Namespace)

	s.instance.UpdateStatePending(
		imv1.ConditionTypeRuntimeProvisioned,
		imv1.ConditionReasonShootCreationPending,
		"Unknown",
		"Shoot is pending",
	)

	// it will be executed only once because created shoot is executed only once
	shouldDumpShootSpec := m.PVCPath != ""
	if shouldDumpShootSpec {
		s.shoot = newShoot.DeepCopy()
		return switchState(sFnDumpShootSpec)
	}

	return updateStatusAndRequeueAfter(gardenerRequeueDuration)
}
