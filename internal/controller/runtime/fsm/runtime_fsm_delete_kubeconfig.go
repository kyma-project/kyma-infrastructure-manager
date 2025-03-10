package fsm

import (
	"context"
	"fmt"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/log_level"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

func sFnDeleteKubeconfig(ctx context.Context, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	// get section
	runtimeID := s.instance.Labels[imv1.LabelKymaRuntimeID]
	var cluster imv1.GardenerCluster
	err := m.Get(ctx, types.NamespacedName{
		Namespace: s.instance.Namespace,
		Name:      runtimeID,
	}, &cluster)

	if err != nil {
		if !k8serrors.IsNotFound(err) {
			m.log.Error(err, "GardenerCluster CR read error", "Name", runtimeID)
			s.instance.UpdateStateDeletion(imv1.RuntimeStateTerminating, imv1.ConditionReasonKubernetesAPIErr, "False", err.Error())
			m.Metrics.IncRuntimeFSMStopCounter()
			return updateStatusAndStop()
		}

		// out section
		return ensureTerminatingStatusConditionAndContinue(&s.instance,
			imv1.ConditionTypeRuntimeDeprovisioned,
			imv1.ConditionReasonGardenerCRDeleted,
			"Gardener Cluster CR successfully deleted",
			sFnDeleteShoot)
	}

	// wait section
	if !cluster.DeletionTimestamp.IsZero() {
		m.log.V(log_level.DEBUG).Info("Waiting for GardenerCluster CR to be deleted", "Runtime", runtimeID, "Shoot", s.shoot.Name)
		return requeueAfter(m.RCCfg.ControlPlaneRequeueDuration)
	}

	// action section
	m.log.Info("deleting GardenerCluster CR", "Runtime", runtimeID, "Shoot", s.shoot.Name)
	err = m.Delete(ctx, &cluster)
	if err != nil {
		// action error handler section
		m.log.Error(err, "Failed to delete gardener Cluster CR")
		s.instance.UpdateStateDeletion(
			imv1.ConditionTypeRuntimeDeprovisioned,
			imv1.ConditionReasonGardenerError,
			"False",
			fmt.Sprintf("Gardener API delete error: %v", err),
		)
	} else {
		s.instance.UpdateStateDeletion(
			imv1.ConditionTypeRuntimeDeprovisioned,
			imv1.ConditionReasonGardenerCRDeleted,
			"Unknown",
			"Runtime shoot deletion started",
		)
	}

	// out succeeded section
	return updateStatusAndRequeueAfter(m.RCCfg.ControlPlaneRequeueDuration)
}
