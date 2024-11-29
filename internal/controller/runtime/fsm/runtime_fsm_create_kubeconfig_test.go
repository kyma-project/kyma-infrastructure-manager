package fsm

import (
	"context"
	"fmt"
	"time"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/controller/metrics/mocks"
	. "github.com/onsi/ginkgo/v2" //nolint:revive
	. "github.com/onsi/gomega"    //nolint:revive
	"github.com/stretchr/testify/mock"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	util "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("KIM sFnCreateKubeconfig", func() {
	now := metav1.NewTime(time.Now())

	testCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// GIVEN

	testScheme := runtime.NewScheme()
	util.Must(imv1.AddToScheme(testScheme))

	withTestSchemeAndObjects := func(objs ...client.Object) fakeFSMOpt {
		return func(fsm *fsm) error {
			return withFakedK8sClient(testScheme, objs...)(fsm)
		}
	}

	withMockedMetrics := func() fakeFSMOpt {
		m := &mocks.Metrics{}
		m.On("SetRuntimeStates", mock.Anything).Return()
		m.On("CleanUpRuntimeGauge", mock.Anything, mock.Anything).Return()
		m.On("IncRuntimeFSMStopCounter").Return()
		return withMetrics(m)
	}

	inputRtWithLabels := makeInputRuntimeWithLabels()
	inputRtWithLabelsAndCondition := makeInputRuntimeWithLabels()

	readyCondition := metav1.Condition{
		Type:               string(imv1.ConditionTypeRuntimeKubeconfigReady),
		Status:             metav1.ConditionTrue,
		LastTransitionTime: now,
		Reason:             string(imv1.ConditionReasonGardenerCRReady),
		Message:            "Test message",
	}

	meta.SetStatusCondition(&inputRtWithLabelsAndCondition.Status.Conditions, readyCondition)

	// input
	testGardenerCRStatePending := makeGardenerClusterCRStatePending()
	testGardenerCRStateReady := makeGardenerClusterCRStateReady()

	testShoot := gardener.Shoot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-instance",
			Namespace: "default",
		},
		Spec: gardener.ShootSpec{
			DNS: &gardener.DNS{
				Domain: ptr.To("test-domain"),
			},
		},
	}

	testFunction := buildTestFunction(sFnHandleKubeconfig)

	// WHEN/THAN

	DescribeTable(
		"transition graph validation",
		testFunction,
		Entry(
			// and set Runtime state to Pending with condition type ConditionTypeRuntimeKubeconfigReady and reason ConditionReasonGardenerCRCreated
			"should create GardenCluster CR when it does not existed before",
			testCtx,
			must(newFakeFSM, withTestFinalizer, withTestSchemeAndObjects(), withMockedMetrics(), withDefaultReconcileDuration()),
			&systemState{instance: *inputRtWithLabels, shoot: &testShoot},
			testOpts{
				MatchExpectedErr: BeNil(),
				MatchNextFnState: haveName("sFnUpdateStatus"),
				//StateMatch:       []types.GomegaMatcher{}, put here matcher for created GardenerCR object
			},
		),
		Entry(
			"should remain in waiting state when GardenCluster CR exists and is not ready yet",
			testCtx,
			must(newFakeFSM, withTestFinalizer, withTestSchemeAndObjects(testGardenerCRStatePending), withMockedMetrics(), withDefaultReconcileDuration()),
			&systemState{instance: *inputRtWithLabels, shoot: &testShoot},
			testOpts{
				MatchExpectedErr: BeNil(),
				MatchNextFnState: BeNil(), // corresponds to requeueAfter(controlPlaneRequeueDuration)
			},
		),
		Entry(
			"should return sFnProcessShoot when GardenCluster CR exists and is in ready state",
			testCtx,
			must(newFakeFSM, withTestFinalizer, withTestSchemeAndObjects(testGardenerCRStateReady), withMockedMetrics(), withDefaultReconcileDuration()),
			&systemState{instance: *inputRtWithLabelsAndCondition, shoot: &testShoot},
			testOpts{
				MatchExpectedErr: BeNil(),
				MatchNextFnState: haveName("sFnConfigureOidc"),
			},
		),
		Entry(
			"should return sFnUpdateStatus when GardenCluster CR exists and is in ready state and condition is not set",
			testCtx,
			must(newFakeFSM, withTestFinalizer, withTestSchemeAndObjects(testGardenerCRStateReady), withMockedMetrics(), withDefaultReconcileDuration()),
			&systemState{instance: *inputRtWithLabels, shoot: &testShoot},
			testOpts{
				MatchExpectedErr: BeNil(),
				MatchNextFnState: haveName("sFnUpdateStatus"),
			},
		),
	)
})

func makeGardenerClusterCR() *imv1.GardenerCluster {
	return &imv1.GardenerCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "GardenerCluster",
			APIVersion: imv1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "059dbc39-fd2b-4186-b0e5-8a1bc8ede5b8",
			Namespace: "default",
			Annotations: map[string]string{
				"skr-domain": "test-domain",
			},
			Labels: map[string]string{
				imv1.LabelKymaInstanceID:      "test-instance-id",
				imv1.LabelKymaRuntimeID:       "test-runtime-id",
				imv1.LabelKymaBrokerPlanID:    "test-broker-plan-id",
				imv1.LabelKymaBrokerPlanName:  "test-broker-plan-name",
				imv1.LabelKymaGlobalAccountID: "test-global-account-id",
				imv1.LabelKymaSubaccountID:    "test-subaccount-id",
				imv1.LabelKymaName:            "test-kyma-name",

				// values from Runtime CR fields
				imv1.LabelKymaPlatformRegion: "test-platform-region",
				imv1.LabelKymaRegion:         "test-region",
				imv1.LabelKymaShootName:      "test-instance",

				// hardcoded values
				imv1.LabelKymaManagedBy: "infrastructure-manager",
				imv1.LabelKymaInternal:  "true",
			},
		},
		Spec: imv1.GardenerClusterSpec{
			Shoot: imv1.Shoot{
				Name: "test-instance",
			},
			Kubeconfig: imv1.Kubeconfig{
				Secret: imv1.Secret{
					Name:      fmt.Sprintf("kubeconfig-%s", "test-runtime-id"),
					Namespace: "default",
					Key:       "config",
				},
			},
		},
	}
}

func makeGardenerClusterCRStatePending() *imv1.GardenerCluster {
	gardenCluster := makeGardenerClusterCR()

	gardenCluster.Status = imv1.GardenerClusterStatus{
		State: imv1.State("Pending"),
	}
	return gardenCluster
}

func makeGardenerClusterCRStateReady() *imv1.GardenerCluster {
	gardenCluster := makeGardenerClusterCR()

	gardenCluster.Status = imv1.GardenerClusterStatus{
		State: imv1.State("Ready"),
	}
	return gardenCluster
}

func makeInputRuntimeWithLabels() *imv1.Runtime {
	return &imv1.Runtime{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-instance",
			Namespace: "default",
			Labels: map[string]string{
				imv1.LabelKymaRuntimeID:       "059dbc39-fd2b-4186-b0e5-8a1bc8ede5b8",
				imv1.LabelKymaInstanceID:      "test-instance",
				imv1.LabelKymaBrokerPlanID:    "broker-plan-id",
				imv1.LabelKymaGlobalAccountID: "461f6292-8085-41c8-af0c-e185f39b5e18",
				imv1.LabelKymaSubaccountID:    "c5ad84ae-3d1b-4592-bee1-f022661f7b30",
				imv1.LabelKymaRegion:          "region",
				imv1.LabelKymaBrokerPlanName:  "aws",
				imv1.LabelKymaName:            "caadafae-1234-1234-1234-123456789abc",
			},
		},
		Status: imv1.RuntimeStatus{
			State: imv1.RuntimeStatePending,
		},
	}
}
