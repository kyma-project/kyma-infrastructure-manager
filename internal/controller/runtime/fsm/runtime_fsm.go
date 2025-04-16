package fsm

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"time"

	"github.com/go-logr/logr"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/controller/metrics"
	"github.com/kyma-project/infrastructure-manager/internal/log_level"
	"github.com/kyma-project/infrastructure-manager/pkg/config"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/auditlogs"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type stateFn func(context.Context, *fsm, *systemState) (stateFn, *ctrl.Result, error)

// runtime reconciler specific configuration
type RCCfg struct {
	GardenerRequeueDuration       time.Duration
	RequeueDurationShootCreate    time.Duration
	RequeueDurationShootDelete    time.Duration
	RequeueDurationShootReconcile time.Duration
	ControlPlaneRequeueDuration   time.Duration
	Finalizer                     string
	ShootNamesapace               string
	AuditLogMandatory             bool
	Metrics                       metrics.Metrics
	AuditLogging                  auditlogs.Configuration
	StructuredAuthEnabled         bool
	config.Config
}

func (f stateFn) String() string {
	return f.name()
}

func (f stateFn) name() string {
	name := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
	return name
}

type Watch = func(src source.Source, eventhandler handler.EventHandler, predicates ...predicate.Predicate) error

type K8s struct {
	client.Client
	record.EventRecorder
	ShootClient client.Client
}

type Fsm interface {
	Run(ctx context.Context, v imv1.Runtime) (ctrl.Result, error)
}

type fsm struct {
	fn  stateFn
	log logr.Logger
	K8s
	RCCfg
}

func (m *fsm) Run(ctx context.Context, v imv1.Runtime) (ctrl.Result, error) {
	state := systemState{instance: v}
	var err error
	var result *ctrl.Result
loop:
	for {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			break loop
		default:
			stateFnName := m.fn.name()
			m.fn, result, err = m.fn(ctx, m, &state)
			newStateFnName := m.fn.name()
			m.log.V(log_level.TRACE).WithValues("result", result, "err", err, "mFnIsNill", m.fn == nil).Info(fmt.Sprintf("switching state from %s to %s", stateFnName, newStateFnName))
			if m.fn == nil || err != nil {
				break loop
			}
		}
	}

	m.log.V(log_level.DEBUG).
		WithValues("result", result).
		Info("Reconciliation done")

	if result != nil {
		return *result, err
	}

	return ctrl.Result{
		Requeue: false,
	}, err
}

func NewFsm(log logr.Logger, cfg RCCfg, k8s K8s) Fsm {
	return &fsm{
		fn:    sFnTakeSnapshot,
		RCCfg: cfg,
		log:   log,
		K8s:   k8s,
	}
}
