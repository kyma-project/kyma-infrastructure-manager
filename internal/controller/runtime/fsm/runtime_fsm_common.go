package fsm

import (
	"context"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
)

// updateStatusAndRequeue writes the Runtime status and re-enqueues after
// m.StatusRequeueDelay. The delay must be > 0: controller-runtime treats
// Result{RequeueAfter: 0} as "no requeue", so a zero or negative value
// silently stalls the FSM at any state that relies on this helper.
func updateStatusAndRequeue(m *fsm) (stateFn, *ctrl.Result, error) {
	return sFnUpdateStatus(&ctrl.Result{RequeueAfter: m.StatusRequeueDelay}, nil), nil, nil
}

func updateStatusAndRequeueAfter(
	//nolint:unparam
	duration time.Duration) (stateFn, *ctrl.Result, error) {
	return sFnUpdateStatus(&ctrl.Result{RequeueAfter: duration}, nil), nil, nil
}

func updateStatusAndStop() (stateFn, *ctrl.Result, error) {
	return sFnUpdateStatus(nil, nil), nil, nil
}

func updateStatusAndStopWithError(err error) (stateFn, *ctrl.Result, error) {
	return sFnUpdateStatus(nil, err), nil, nil
}

func requeue() (stateFn, *ctrl.Result, error) {
	return nil, &ctrl.Result{Requeue: true}, nil
}

func requeueAfter(d time.Duration) (stateFn, *ctrl.Result, error) {
	return nil, &ctrl.Result{RequeueAfter: d}, nil
}

func stop() (stateFn, *ctrl.Result, error) {
	return nil, nil, nil
}

func switchState(fn stateFn) (stateFn, *ctrl.Result, error) {
	return fn, nil, nil
}

func stopWithMetrics() (stateFn, *ctrl.Result, error) {
	return func(_ context.Context, m *fsm, _ *systemState) (stateFn, *ctrl.Result, error) {
		m.Metrics.IncRuntimeFSMStopCounter()
		return stop()
	}, nil, nil
}
