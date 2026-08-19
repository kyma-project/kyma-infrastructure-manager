package fsm

import (
	"context"
	"slices"
	"time"

	gardener_api "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	. "github.com/onsi/ginkgo/v2" //nolint:revive
	. "github.com/onsi/gomega"    //nolint:revive
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("sFnWaitForShootCreation marks Shoot for retry if InvalidSubnetID.NotFound error occurs", func() {
	withMockedMetrics()

	It("sets the Runtime to pending with a retryable error condition", func() {
		testRuntime := imv1.Runtime{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-runtime",
				Namespace: "kcp-system",
			},
		}

		testScheme, err := newTestScheme()
		Expect(err).ShouldNot(HaveOccurred())

		err = gardener_api.AddToScheme(testScheme)
		Expect(err).ShouldNot(HaveOccurred())

		shoot := &gardener_api.Shoot{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-shoot",
				Namespace: "default",
			},
			Status: gardener_api.ShootStatus{
				LastOperation: &gardener_api.LastOperation{
					Type:        gardener_api.LastOperationTypeCreate,
					State:       gardener_api.LastOperationStateFailed,
					Description: "something InvalidSubnetID.NotFound something",
				},
				LastErrors: []gardener_api.LastError{{
					Codes:       []gardener_api.ErrorCode{gardener_api.ErrorConfigurationProblem},
					Description: "something InvalidSubnetID.NotFound something",
				}},
			},
		}

		sFnWaitForShootCreationSetup := newSetupStateForTest(sFnWaitForShootCreation, func(s *systemState) error {
			s.shoot = shoot.DeepCopy()
			s.saveRuntimeStatus()
			return nil
		})

		expectedCondition := metav1.Condition{
			Type:    string(imv1.ConditionTypeRuntimeProvisioned),
			Status:  metav1.ConditionUnknown,
			Reason:  string(imv1.ConditionReasonShootCreationPending),
			Message: "Retryable gardener error InvalidSubnetID.NotFound during cluster provisioning",
		}

		fsm, err := newFakeFSM(
			withFakedK8sClient(testScheme, &testRuntime, shoot),
			withFn(sFnWaitForShootCreationSetup),
			withFakeEventRecorder(1),
			withMockedMetrics(),
			withDefaultReconcileDuration(),
		)
		Expect(err).ShouldNot(HaveOccurred())

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, runErr := fsm.Run(ctx, testRuntime)
		Expect(runErr).ShouldNot(HaveOccurred())

		persistedRuntime := &imv1.Runtime{}
		err = fsm.KcpClient.Get(ctx, client.ObjectKeyFromObject(&testRuntime), persistedRuntime)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(string(persistedRuntime.Status.State)).To(Equal(imv1.RuntimeStatePending))
		Expect(conditionMatches(persistedRuntime.Status.Conditions, expectedCondition)).To(BeTrue(), "expected condition not found in persisted Runtime status")

		persistedShoot := &gardener_api.Shoot{}
		err = fsm.GardenClient.Get(ctx, client.ObjectKeyFromObject(shoot), persistedShoot)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(persistedShoot.Annotations).To(HaveKeyWithValue(v1beta1constants.GardenerOperation, v1beta1constants.ShootOperationRetry), "expected retry annotation not found in persisted Shoot")
	})

	It("does not re-patch the Shoot when the retry annotation is already set", func() {
		testRuntime := imv1.Runtime{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-runtime",
				Namespace: "kcp-system",
			},
		}

		testScheme, err := newTestScheme()
		Expect(err).ShouldNot(HaveOccurred())

		err = gardener_api.AddToScheme(testScheme)
		Expect(err).ShouldNot(HaveOccurred())

		shoot := &gardener_api.Shoot{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-shoot",
				Namespace: "default",
				Annotations: map[string]string{
					v1beta1constants.GardenerOperation: v1beta1constants.ShootOperationRetry,
				},
			},
			Status: gardener_api.ShootStatus{
				LastOperation: &gardener_api.LastOperation{
					Type:        gardener_api.LastOperationTypeCreate,
					State:       gardener_api.LastOperationStateFailed,
					Description: "something InvalidSubnetID.NotFound something",
				},
				LastErrors: []gardener_api.LastError{{
					Codes:       []gardener_api.ErrorCode{gardener_api.ErrorConfigurationProblem},
					Description: "something InvalidSubnetID.NotFound something",
				}},
			},
		}

		sFnWaitForShootCreationSetup := newSetupStateForTest(sFnWaitForShootCreation, func(s *systemState) error {
			s.shoot = shoot.DeepCopy()
			s.saveRuntimeStatus()
			return nil
		})

		expectedCondition := metav1.Condition{
			Type:    string(imv1.ConditionTypeRuntimeProvisioned),
			Status:  metav1.ConditionUnknown,
			Reason:  string(imv1.ConditionReasonShootCreationPending),
			Message: "Retryable gardener error InvalidSubnetID.NotFound during cluster provisioning",
		}

		fsm, err := newFakeFSM(
			withFakedK8sClient(testScheme, &testRuntime, shoot),
			withFn(sFnWaitForShootCreationSetup),
			withFakeEventRecorder(1),
			withMockedMetrics(),
			withDefaultReconcileDuration(),
		)
		Expect(err).ShouldNot(HaveOccurred())

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		shootBefore := &gardener_api.Shoot{}
		err = fsm.GardenClient.Get(ctx, client.ObjectKeyFromObject(shoot), shootBefore)
		Expect(err).ShouldNot(HaveOccurred())

		_, runErr := fsm.Run(ctx, testRuntime)
		Expect(runErr).ShouldNot(HaveOccurred())

		persistedRuntime := &imv1.Runtime{}
		err = fsm.KcpClient.Get(ctx, client.ObjectKeyFromObject(&testRuntime), persistedRuntime)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(string(persistedRuntime.Status.State)).To(Equal(imv1.RuntimeStatePending))
		Expect(conditionMatches(persistedRuntime.Status.Conditions, expectedCondition)).To(BeTrue(), "expected condition not found in persisted Runtime status")

		persistedShoot := &gardener_api.Shoot{}
		err = fsm.GardenClient.Get(ctx, client.ObjectKeyFromObject(shoot), persistedShoot)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(persistedShoot.Annotations).To(HaveKeyWithValue(v1beta1constants.GardenerOperation, v1beta1constants.ShootOperationRetry), "retry annotation must remain set on the Shoot")
		Expect(persistedShoot.ResourceVersion).To(Equal(shootBefore.ResourceVersion), "Shoot must not be re-patched when the retry annotation is already set")
	})
})

func conditionMatches(actual []metav1.Condition, expected metav1.Condition) bool {
	return slices.ContainsFunc(actual, func(condition metav1.Condition) bool {
		return condition.Type == expected.Type &&
			condition.Status == expected.Status &&
			condition.Reason == expected.Reason &&
			condition.Message == expected.Message
	})
}
