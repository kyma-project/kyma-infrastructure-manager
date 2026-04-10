package fsm

import (
	"context"
	"reflect"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func sFnUpdateStatus(result *ctrl.Result, err error) stateFn {
	return func(ctx context.Context, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
		if err != nil {
			m.Metrics.IncRuntimeFSMStopCounter()
		}

		// make sure there is a change in status
		if reflect.DeepEqual(s.instance.Status, s.snapshot) {
			return nil, result, err
		}

		updateErr := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
			var latestRuntime imv1.Runtime
			if getErr := m.KcpClient.Get(ctx, client.ObjectKeyFromObject(&s.instance), &latestRuntime); getErr != nil {
				return getErr
			}

			latestRuntime.Status = s.instance.Status

			if statusErr := m.KcpClient.Status().Update(ctx, &latestRuntime); statusErr != nil {
				return statusErr
			}

			s.instance = latestRuntime
			return nil
		})

		if updateErr != nil {
			if apierrors.IsConflict(updateErr) {
				m.log.Info("conflict while updating runtime status after retries", "name", s.instance.Name, "namespace", s.instance.Namespace)
			} else {
				m.log.Error(updateErr, "unable to update instance status!")
			}
			if err == nil {
				err = updateErr
			}
			return nil, nil, err
		}

		m.Metrics.SetRuntimeStates(s.instance)
		next := sFnEmmitEventfunc(nil, result, err)
		return next, nil, nil
	}
}
