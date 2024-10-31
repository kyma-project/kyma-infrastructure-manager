package fsm

import (
	"context"
	"fmt"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/config"
	gardener_shoot "github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot"
	ctrl "sigs.k8s.io/controller-runtime"
)

func sFnCreateShoot(ctx context.Context, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	m.log.Info("Create shoot state")

	newShoot, err := convertCreate(&s.instance, m.ConverterConfig)
	if err != nil {
		m.log.Error(err, "Failed to convert Runtime instance to shoot object")
		m.Metrics.IncRuntimeFSMStopCounter()
		return updateStatePendingWithErrorAndStop(
			&s.instance,
			imv1.ConditionTypeRuntimeProvisioned,
			imv1.ConditionReasonConversionError,
			"Runtime conversion error")
	}

	err = m.ShootClient.Create(ctx, &newShoot)

	if err != nil {
		m.log.Error(err, "Failed to create new gardener Shoot")

		s.instance.UpdateStatePending(
			imv1.ConditionTypeRuntimeProvisioned,
			imv1.ConditionReasonGardenerError,
			"False",
			fmt.Sprintf("Gardener API create error: %v", err),
		)
		return updateStatusAndRequeueAfter(m.RCCfg.GardenerRequeueDuration)
	}

	m.log.Info(
		"Gardener shoot for runtime initialised successfully",
		"Name", newShoot.Name,
		"Namespace", newShoot.Namespace,
	)

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

	return updateStatusAndRequeueAfter(m.RCCfg.GardenerRequeueDuration)
}

func convertCreate(instance *imv1.Runtime, cfg config.ConverterConfig) (gardener.Shoot, error) {
	if err := instance.ValidateRequiredLabels(); err != nil {
		return gardener.Shoot{}, err
	}

	converter := gardener_shoot.NewConverterCreate(cfg)
	newShoot, err := converter.ToShoot(*instance)
	if err != nil {
		setObjectFields(&newShoot)
		return newShoot, err
	}

	return newShoot, nil
}
