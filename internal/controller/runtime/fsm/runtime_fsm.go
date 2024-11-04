package fsm

import (
	"context"
	"fmt"
	"io"
	"reflect"
	"runtime"
	"time"

	"github.com/go-logr/logr"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/controller/metrics"
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
type writerGetter = func(filePath string) (io.Writer, error)

// runtime reconciler specific configuration
type RCCfg struct {
	GardenerRequeueDuration     time.Duration
	ControlPlaneRequeueDuration time.Duration
	Finalizer                   string
	PVCPath                     string
	ShootNamesapace             string
	AuditLogMandatory           bool
	Metrics                     metrics.Metrics
	AuditLogging                map[string]map[string]auditlogs.AuditLogData
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
	fn             stateFn
	writerProvider writerGetter
	log            logr.Logger
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
			m.log.WithValues("result", result, "err", err, "mFnIsNill", m.fn == nil).Info(fmt.Sprintf("switching state from %s to %s", stateFnName, newStateFnName))
			if m.fn == nil || err != nil {
				break loop
			}
		}
	}

	m.log.WithValues("error", err).
		WithValues("result", result).
		Info("reconciliation done")

	if result != nil {
		return *result, err
	}

	return ctrl.Result{
		Requeue: false,
	}, err
}

func NewFsm(log logr.Logger, cfg RCCfg, k8s K8s) Fsm {
	return &fsm{
		fn:             sFnTakeSnapshot,
		writerProvider: getWriterForFilesystem,
		RCCfg:          cfg,
		log:            log,
		K8s:            k8s,
	}
}
