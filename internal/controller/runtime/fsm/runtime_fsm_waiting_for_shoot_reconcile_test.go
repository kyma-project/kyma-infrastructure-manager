package fsm

import (
	"context"
	"time"

	gardener_api "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/controller/metrics/mocks"
	. "github.com/onsi/ginkgo/v2" //nolint:revive
	. "github.com/onsi/gomega"    //nolint:revive
	"github.com/stretchr/testify/mock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// End-to-end regression test for KIM issue #1413. Drives sFnWaitForShootReconcile
// with a Succeeded shoot and a Runtime that does not yet carry the success
// condition. The state function calls ensureStatusConditionIsSetAndContinue,
// which mutates the Runtime status and returns updateStatusAndRequeueAfter(m.StatusRequeueDelay).
// The FSM then walks through sFnUpdateStatus → sFnEmmitEventfunc and surfaces the
// captured Result. The reconcile result must carry RequeueAfter ==
// StatusRequeueDelay (not Requeue: true / RequeueAfter: 0).
var _ = Describe("sFnWaitForShootReconcile re-enqueue delay", Label("issue-1413"), func() {

	withMockedMetrics := func() fakeFSMOpt {
		m := &mocks.Metrics{}
		m.On("SetRuntimeStates", mock.Anything).Return()
		m.On("CleanUpRuntimeGauge", mock.Anything, mock.Anything).Return()
		m.On("IncRuntimeFSMStopCounter").Return()
		return withMetrics(m)
	}

	withStatusRequeueDelay := func(d time.Duration) fakeFSMOpt {
		return func(f *fsm) error {
			f.StatusRequeueDelay = d
			return nil
		}
	}

	testRuntime := imv1.Runtime{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-runtime",
			Namespace: "default",
		},
	}

	testScheme, err := newTestScheme()
	Expect(err).ShouldNot(HaveOccurred())

	sFnWaitForShootReconcileSetup := newSetupStateForTest(sFnWaitForShootReconcile, func(s *systemState) error {
		s.shoot = &gardener_api.Shoot{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-shoot",
				Namespace: "default",
			},
			Status: gardener_api.ShootStatus{
				LastOperation: &gardener_api.LastOperation{
					Type:  gardener_api.LastOperationTypeReconcile,
					State: gardener_api.LastOperationStateSucceeded,
				},
			},
		}
		s.saveRuntimeStatus()
		return nil
	})

	DescribeTable("uses StatusRequeueDelay (not zero-delay Requeue) when re-enqueuing after a status write",
		func(delay time.Duration) {
			fsm, err := newFakeFSM(
				withFakedK8sClient(testScheme, &testRuntime),
				withFn(sFnWaitForShootReconcileSetup),
				withFakeEventRecorder(1),
				withMockedMetrics(),
				withDefaultReconcileDuration(),
				withStatusRequeueDelay(delay),
			)
			Expect(err).ShouldNot(HaveOccurred())

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			result, runErr := fsm.Run(ctx, testRuntime)
			Expect(runErr).ShouldNot(HaveOccurred())
			Expect(result.RequeueAfter).Should(Equal(delay),
				"reconcile result must re-enqueue after StatusRequeueDelay, not after 0")
		},
		Entry("1s delay", 1*time.Second),
		Entry("3s delay", 3*time.Second),
	)
})
