package fsm

import (
	"context"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/log_level"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func sFnCreateKymaNamespace(ctx context.Context, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	kymaNsCreationErr := createKymaSystemNamespace(ctx, m, s)
	if kymaNsCreationErr != nil {
		m.log.Error(kymaNsCreationErr, "Failed to create kyma-system namespace. Scheduling for retry")
		s.instance.UpdateStatePending(
			imv1.ConditionTypeKymaSystemCreated,
			imv1.ConditionReasonKymaSystemNSError,
			"False",
			kymaNsCreationErr.Error(),
		)
		m.Metrics.IncRuntimeFSMStopCounter()
		return updateStatusAndRequeueAfter(m.ControlPlaneRequeueDuration)
	}

	m.log.V(log_level.DEBUG).Info("kyma-system namespace created/existed", "name", s.instance.Name)

	s.instance.UpdateStatePending(
		imv1.ConditionTypeKymaSystemCreated,
		imv1.ConditionReasonKymaSystemNSReady,
		"True",
		"Creation of kyma-system Namespace",
	)

	return switchState(sFnInitializeRuntimeBootstrapper)
}

func createKymaSystemNamespace(ctx context.Context, m *fsm, s *systemState) error {
	kymaSystemNs := v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kyma-system",
		},
	}

	runtimeClient, runtimeClientError := m.RuntimeClientGetter.Get(ctx, s.instance)

	if runtimeClientError != nil {
		return runtimeClientError
	}
	kymaNsCreationErr := runtimeClient.Create(ctx, &kymaSystemNs)

	if kymaNsCreationErr != nil {
		if k8serrors.IsAlreadyExists(kymaNsCreationErr) {
			// we're expecting the namespace to already exist after first reconciliation, so we can ignore this error
			return nil
		}
	}
	return kymaNsCreationErr
}
