package fsm

import (
	"context"
	"fmt"
	"time"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardener_api "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/kyma-project/infrastructure-manager/internal/controller/metrics"
	"github.com/kyma-project/infrastructure-manager/pkg/config"
	. "github.com/onsi/ginkgo/v2" //nolint:revive
	. "github.com/onsi/gomega"    //nolint:revive
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

type fakeFSMOpt func(*fsm) error

const defaultControlPlaneRequeueDuration = 10 * time.Second
const defaultGardenerRequeueDuration = 15 * time.Second

var (
	errFailedToCreateFakeFSM = fmt.Errorf("failed to create fake FSM")

	must = func(f func(opts ...fakeFSMOpt) (*fsm, error), opts ...fakeFSMOpt) *fsm {
		fsm, err := f(opts...)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(fsm).NotTo(BeNil())
		return fsm
	}

	withFinalizer = func(finalizer string) fakeFSMOpt {
		return func(fsm *fsm) error {
			fsm.Finalizer = finalizer
			return nil
		}
	}

	withStorageWriter = func(testWriterGetter writerGetter) fakeFSMOpt {
		return func(fsm *fsm) error {
			fsm.writerProvider = testWriterGetter
			return nil
		}
	}

	withConverterConfig = func(config config.ConverterConfig) fakeFSMOpt {
		return func(fsm *fsm) error {
			fsm.Config.ConverterConfig = config
			return nil
		}
	}

	withMetrics = func(m metrics.Metrics) fakeFSMOpt {
		return func(fsm *fsm) error {
			fsm.Metrics = m
			return nil
		}
	}

	withDefaultReconcileDuration = func() fakeFSMOpt {
		return func(fsm *fsm) error {
			fsm.ControlPlaneRequeueDuration = defaultControlPlaneRequeueDuration
			fsm.GardenerRequeueDuration = defaultGardenerRequeueDuration
			return nil
		}
	}

	withFakedK8sClient = func(
		scheme *runtime.Scheme,
		objs ...client.Object) fakeFSMOpt {

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(objs...).
			WithStatusSubresource(objs...).
			Build()

		return func(fsm *fsm) error {
			fsm.Client = k8sClient
			fsm.ShootClient = k8sClient
			return nil
		}
	}

	withFn = func(fn stateFn) fakeFSMOpt {
		return func(fsm *fsm) error {
			fsm.fn = fn
			return nil
		}
	}

	withFakeEventRecorder = func(buffer int) fakeFSMOpt {
		return func(fsm *fsm) error {
			fsm.EventRecorder = record.NewFakeRecorder(buffer)
			return nil
		}
	}

	withAuditLogging = func(isEnabled bool, err error) fakeFSMOpt {
		return func(fsm *fsm) error {
			fsm.AuditLogging = &stubAuditLogging{
				isEnabled: isEnabled,
				err:       err,
			}
			return nil
		}
	}
)

func newFakeFSM(opts ...fakeFSMOpt) (*fsm, error) {
	fsm := fsm{
		log: zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)),
	}
	// apply opts
	for _, opt := range opts {
		if err := opt(&fsm); err != nil {
			return nil, fmt.Errorf(
				"%w: %s",
				errFailedToCreateFakeFSM,
				err.Error(),
			)
		}
	}
	return &fsm, nil
}

// stubAuditLogging - a special type to allow to test audit logging
type stubAuditLogging struct {
	isEnabled bool
	err       error
}

func (s *stubAuditLogging) Enable(ctx context.Context, shoot *gardener.Shoot) (bool, error) {
	return s.isEnabled, s.err
}

func (s *stubAuditLogging) UpdateShootClient(client client.Client) {
}

func newSetupStateForTest(sfn stateFn, opts ...func(*systemState) error) stateFn {
	return func(_ context.Context, _ *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
		for _, fn := range opts {
			if err := fn(s); err != nil {
				return nil, nil, fmt.Errorf("test state setup failed: %s", err)
			}
		}
		return sfn, nil, nil
	}
}

// sFnApplyClusterRoleBindingsStateSetup a special function to setup system state in tests
var sFnApplyClusterRoleBindingsStateSetup = newSetupStateForTest(sFnApplyClusterRoleBindings, func(s *systemState) error {

	s.shoot = &gardener_api.Shoot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-shoot",
			Namespace: "test-namespace",
		},
	}

	return nil
})
