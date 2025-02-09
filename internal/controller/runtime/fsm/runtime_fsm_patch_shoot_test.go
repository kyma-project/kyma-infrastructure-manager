package fsm

import (
	"context"
	"time"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/controller/metrics/mocks"
	. "github.com/onsi/ginkgo/v2" //nolint:revive
	. "github.com/onsi/gomega"    //nolint:revive
	"github.com/onsi/gomega/types"
	"github.com/stretchr/testify/mock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	util "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/utils/ptr"
)

var _ = Describe("KIM sFnPatchExistingShoot", func() {

	testCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	testScheme := runtime.NewScheme()
	util.Must(imv1.AddToScheme(testScheme))
	util.Must(gardener.AddToScheme(testScheme))

	withMockedMetrics := func() fakeFSMOpt {
		m := &mocks.Metrics{}
		m.On("SetRuntimeStates", mock.Anything).Return()
		m.On("CleanUpRuntimeGauge", mock.Anything, mock.Anything).Return()
		m.On("IncRuntimeFSMStopCounter").Return()
		return withMetrics(m)
	}

	inputRtWithForceAnnotation := makeInputRuntimeWithAnnotation(map[string]string{"operator.kyma-project.io/force-patch-reconciliation": "true"})

	testShoot := gardener.Shoot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-shoot",
			Namespace: "garden-",
		},
		Spec: gardener.ShootSpec{
			DNS: &gardener.DNS{
				Domain: ptr.To("test-domain"),
			},
		},
		Status: gardener.ShootStatus{
			LastOperation: &gardener.LastOperation{
				State: gardener.LastOperationStateSucceeded,
			},
		},
	}

	testFunction := buildPatchTestFunction(sFnPatchExistingShoot)

	var expectedAnnotations map[string]string

	DescribeTable(
		"transition graph validation for sFnPatchExistingShoot",
		testFunction,
		Entry(
			"should update status after succesful patching and remove force patch annotation",
			testCtx,
			must(newFakeFSM, withMockedMetrics(), withTestFinalizer, withFakedK8sClient(testScheme, inputRtWithForceAnnotation), withFakeEventRecorder(1)),
			&systemState{instance: *inputRtWithForceAnnotation, shoot: &testShoot},
			haveName("sFnHandleKubeconfig"),
			expectedAnnotations,
		),
	)
})

func buildPatchTestFunction(fn stateFn) func(context.Context, *fsm, *systemState, types.GomegaMatcher, map[string]string) {
	return func(ctx context.Context, r *fsm, s *systemState, matchNextFnState types.GomegaMatcher, expectedAnnotations map[string]string) {

		createErr := r.ShootClient.Create(ctx, s.shoot)
		if createErr != nil {
			return
		}

		sFn, _, err := fn(ctx, r, s)

		Expect(err).To(BeNil())
		Expect(sFn).To(matchNextFnState)
		Expect(s.instance.GetAnnotations()).To(Equal(expectedAnnotations))
	}
}
