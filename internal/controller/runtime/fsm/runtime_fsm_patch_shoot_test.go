package fsm

import (
	"context"
	"testing"
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
			haveName("sFnUpdateStatus"),
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

func TestWorkersAreEqual(t *testing.T) {
	tests := []struct {
		name     string
		workers1 []gardener.Worker
		workers2 []gardener.Worker
		want     bool
	}{
		{
			name: "equal workers",
			workers1: []gardener.Worker{
				{Name: "worker1", Minimum: 1, Maximum: 3},
				{Name: "worker2", Minimum: 3, Maximum: 10},
			},
			workers2: []gardener.Worker{
				{Name: "worker1", Minimum: 1, Maximum: 3},
				{Name: "worker2", Minimum: 3, Maximum: 10},
			},
			want: true,
		},
		{
			name: "equal workers #2 - zones",
			workers1: []gardener.Worker{
				{Name: "worker1", Zones: []string{"zone1", "zone2"}},
			},
			workers2: []gardener.Worker{
				{Name: "worker1", Zones: []string{"zone1", "zone2"}},
			},
			want: true,
		},
		{
			name: "equal workers #3 - CRI",
			workers1: []gardener.Worker{
				{Name: "worker1", CRI: &gardener.CRI{Name: "runtime", ContainerRuntimes: []gardener.ContainerRuntime{
					{Type: "docker"},
				}}},
			},
			workers2: []gardener.Worker{
				{Name: "worker1", CRI: &gardener.CRI{Name: "runtime", ContainerRuntimes: []gardener.ContainerRuntime{
					{Type: "docker"},
				}}},
			},
			want: true,
		},
		{
			name:     "empty workers",
			workers1: []gardener.Worker{},
			workers2: []gardener.Worker{},
			want:     true,
		},
		{
			name: "different workers - name",
			workers1: []gardener.Worker{
				{Name: "worker1", Minimum: 1, Maximum: 3},
			},
			workers2: []gardener.Worker{
				{Name: "worker2", Minimum: 1, Maximum: 3},
			},
			want: false,
		},
		{
			name: "different workers #2 - minmax",
			workers1: []gardener.Worker{
				{Name: "worker1", Minimum: 1, Maximum: 3},
			},
			workers2: []gardener.Worker{
				{Name: "worker1", Minimum: 1, Maximum: 1},
			},
			want: false,
		},
		{
			name: "different workers #3 - zones",
			workers1: []gardener.Worker{
				{Name: "worker1", Minimum: 1, Maximum: 3, Zones: []string{"zone1", "zone2"}},
			},
			workers2: []gardener.Worker{
				{Name: "worker1", Minimum: 1, Maximum: 3, Zones: []string{"zone1", "zone3"}},
			},
			want: false,
		},
		{
			name: "different workers #4 - CRI",
			workers1: []gardener.Worker{
				{Name: "worker1", CRI: &gardener.CRI{Name: "runtime", ContainerRuntimes: []gardener.ContainerRuntime{
					{Type: "docker"},
				}}},
			},
			workers2: []gardener.Worker{
				{Name: "worker1", CRI: &gardener.CRI{Name: "runtime", ContainerRuntimes: []gardener.ContainerRuntime{
					{Type: "containerd"},
				}}},
			},
			want: false,
		},
		{
			name: "different number of workers",
			workers1: []gardener.Worker{
				{Name: "worker1", Minimum: 1, Maximum: 3},
			},
			workers2: []gardener.Worker{
				{Name: "worker1", Minimum: 1, Maximum: 3},
				{Name: "worker1", Minimum: 1, Maximum: 3},
			},
			want: false,
		},
		{
			name:     "one workers collection is empty",
			workers1: []gardener.Worker{},
			workers2: []gardener.Worker{
				{Name: "worker1", Minimum: 1, Maximum: 3},
				{Name: "worker1", Minimum: 1, Maximum: 3},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := workersAreEqual(tt.workers1, tt.workers2); got != tt.want {
				t.Errorf("workersAreEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}
