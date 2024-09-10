package fsm

import (
	"context"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/gardener/shoot"
	gardener_shoot "github.com/kyma-project/infrastructure-manager/internal/gardener/shoot"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func sFnPatchExistingShoot(ctx context.Context, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	m.log.Info("Patch shoot state")

	updatedShoot, err := convertShoot(&s.instance, m.ConverterConfig)
	if err != nil {
		m.log.Error(err, "Failed to convert Runtime instance to shoot object, exiting with no retry")
		return updateStatePendingWithErrorAndStop(&s.instance, imv1.ConditionTypeRuntimeProvisioned, imv1.ConditionReasonConversionError, "Runtime conversion error")
	}

	m.log.Info("Shoot converted successfully", "Name", updatedShoot.Name, "Namespace", updatedShoot.Namespace)

	err = m.ShootClient.Patch(ctx, &updatedShoot, client.Apply, &client.PatchOptions{
		FieldManager: "kim",
		Force:        ptr.To(true),
	})

	if err != nil {
		m.log.Error(err, "Failed to patch shoot object, exiting with no retry")
		return updateStatePendingWithErrorAndStop(&s.instance, imv1.ConditionTypeRuntimeProvisioned, imv1.ConditionReasonProcessingErr, "Shoot patch error")
	}

	if updatedShoot.Generation == s.shoot.Generation {
		m.log.Info("Gardener shoot for runtime did not change after patch, moving to processing", "Name", s.shoot.Name, "Namespace", s.shoot.Namespace)
		return switchState(sFnApplyClusterRoleBindings)
	}

	m.log.Info("Gardener shoot for runtime patched successfully", "Name", s.shoot.Name, "Namespace", s.shoot.Namespace)

	s.shoot = updatedShoot.DeepCopy()

	s.instance.UpdateStatePending(
		imv1.ConditionTypeRuntimeProvisioned,
		imv1.ConditionReasonProcessing,
		"Unknown",
		"Shoot is pending for update",
	)

	return updateStatusAndRequeueAfter(gardenerRequeueDuration)
}

func convertShoot(instance *imv1.Runtime, cfg shoot.ConverterConfig) (gardener.Shoot, error) {
	if err := instance.ValidateRequiredLabels(); err != nil {
		return gardener.Shoot{}, err
	}

	converter := gardener_shoot.NewConverter(cfg)
	newShoot, err := converter.ToShoot(*instance)

	if err == nil {
		setObjectFields(&newShoot)
	}

	return newShoot, err
}

// workaround
func setObjectFields(shoot *gardener.Shoot) {
	shoot.Kind = "Shoot"
	shoot.APIVersion = "core.gardener.cloud/v1beta1"
	shoot.ManagedFields = nil
}

func updateStatePendingWithErrorAndStop(instance *imv1.Runtime,
	//nolint:unparam
	c imv1.RuntimeConditionType, r imv1.RuntimeConditionReason, msg string) (stateFn, *ctrl.Result, error) {
	instance.UpdateStatePending(c, r, "False", msg)
	return updateStatusAndStop()
}
