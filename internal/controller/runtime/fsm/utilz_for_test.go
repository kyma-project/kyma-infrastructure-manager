package fsm

import (
	"context"
	"fmt"
	gardener_api "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/controller/metrics"
	metrics_mocks "github.com/kyma-project/infrastructure-manager/internal/controller/metrics/mocks"
	fsm_mocks "github.com/kyma-project/infrastructure-manager/internal/controller/runtime/fsm/mocks"
	fsm_testing "github.com/kyma-project/infrastructure-manager/internal/controller/runtime/fsm/testing"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/auditlogs"
	. "github.com/onsi/ginkgo/v2" //nolint:revive
	. "github.com/onsi/gomega"    //nolint:revive
	"github.com/onsi/gomega/types"
	"github.com/stretchr/testify/mock"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"time"
)

type fakeFSMOpt func(*fsm) error

const defaultControlPlaneRequeueDuration = 10 * time.Second
const defaultGardenerRequeueDuration = 15 * time.Second

type outputFnState struct {
	nextStep    types.GomegaMatcher
	annotations map[string]string
	result      *ctrl.Result
	status      imv1.RuntimeStatus
}

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

	withShootNamespace = func(ns string) fakeFSMOpt {
		return func(fsm *fsm) error {
			fsm.ShootNamesapace = ns
			return nil
		}
	}

	withTestFinalizer = withFinalizer("test-me-plz")

	withMockedMetrics = func() fakeFSMOpt {
		m := &metrics_mocks.Metrics{}
		m.On("SetRuntimeStates", mock.Anything).Return()
		m.On("CleanUpRuntimeGauge", mock.Anything, mock.Anything).Return()
		m.On("IncRuntimeFSMStopCounter").Return()
		return withMetrics(m)
	}

	withAuditLogMandatory = func(isMandatory bool) fakeFSMOpt {
		return func(fsm *fsm) error {
			fsm.AuditLogMandatory = isMandatory
			return nil
		}
	}

	withAuditLogConfig = func(provider, region string, data auditlogs.AuditLogData) fakeFSMOpt {
		return func(fsm *fsm) error {
			fsm.AuditLogging = auditlogs.Configuration{
				provider: {
					region: data,
				},
			}
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
			WithInterceptorFuncs(interceptor.Funcs{
				Patch:  fsm_testing.GetFakePatchInterceptorFn(true),
				Update: fsm_testing.GetFakeUpdateInterceptorFn(true),
			}).Build()

		runtimeClientGetter := &fsm_mocks.RuntimeClientGetter{}
		runtimeClientGetter.On("Get", mock.Anything, mock.Anything).Return(k8sClient, nil)

		return func(fsm *fsm) error {
			fsm.KcpClient = k8sClient
			fsm.GardenClient = k8sClient
			fsm.RuntimeClientGetter = runtimeClientGetter
			return nil
		}
	}

	withFailedRuntimeK8sClient = func(err error, scheme *runtime.Scheme,
		objs ...client.Object) fakeFSMOpt {

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(objs...).
			WithStatusSubresource(objs...).Build()

		runtimeClientGetter := &fsm_mocks.RuntimeClientGetter{}
		runtimeClientGetter.On("Get", mock.Anything, mock.Anything).Return(nil, err)

		return func(fsm *fsm) error {
			fsm.KcpClient = k8sClient
			fsm.GardenClient = k8sClient
			fsm.RuntimeClientGetter = runtimeClientGetter
			return nil
		}
	}

	withFakedK8sClientKeepGeneration = func(
		scheme *runtime.Scheme,
		objs ...client.Object) fakeFSMOpt {

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(objs...).
			WithStatusSubresource(objs...).
			WithInterceptorFuncs(interceptor.Funcs{
				Patch:  fsm_testing.GetFakePatchInterceptorFn(false),
				Update: fsm_testing.GetFakeUpdateInterceptorFn(false),
			}).Build()

		return func(fsm *fsm) error {
			fsm.KcpClient = k8sClient
			fsm.GardenClient = k8sClient
			return nil
		}
	}

	withFakedK8sClientWithFakeUpdateAndPatch = func(
		scheme *runtime.Scheme,
		objs ...client.Object) fakeFSMOpt {

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(objs...).
			WithStatusSubresource(objs...).Build()

		runtimeClientGetter := &fsm_mocks.RuntimeClientGetter{}
		runtimeClientGetter.On("Get", mock.Anything, mock.Anything).Return(k8sClient, nil)

		return func(fsm *fsm) error {
			fsm.KcpClient = k8sClient
			fsm.GardenClient = k8sClient
			fsm.RuntimeClientGetter = runtimeClientGetter
			return nil
		}
	}

	withFakedK8sClientFailPatchError = func(
		err *k8s_errors.StatusError,
		scheme *runtime.Scheme,
		objs ...client.Object) fakeFSMOpt {

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(objs...).
			WithStatusSubresource(objs...).
			WithInterceptorFuncs(interceptor.Funcs{
				Patch:  fsm_testing.GetFakePatchInterceptorFnError(err),
				Update: fsm_testing.GetFakeUpdateInterceptorFn(true),
			}).Build()

		return func(fsm *fsm) error {
			fsm.KcpClient = k8sClient
			fsm.GardenClient = k8sClient
			return nil
		}
	}

	withFakedK8sClientFailUpdateError = func(
		err *k8s_errors.StatusError,
		scheme *runtime.Scheme,
		objs ...client.Object) fakeFSMOpt {

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(objs...).
			WithStatusSubresource(objs...).
			WithInterceptorFuncs(interceptor.Funcs{
				Patch:  fsm_testing.GetFakePatchInterceptorFn(true),
				Update: fsm_testing.GetFakeUpdateInterceptorFnError(err),
			}).Build()

		return func(fsm *fsm) error {
			fsm.KcpClient = k8sClient
			fsm.GardenClient = k8sClient
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
