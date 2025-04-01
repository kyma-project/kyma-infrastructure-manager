package fsm

import (
	"context"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/auditlogs"
	"k8s.io/apimachinery/pkg/api/meta"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
	"time"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	. "github.com/onsi/ginkgo/v2" //nolint:revive
	. "github.com/onsi/gomega"    //nolint:revive
	"github.com/onsi/gomega/types"
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

	expectedAnnotations := map[string]string{"operator.kyma-project.io/existing-annotation": "true"}
	inputRuntimeWithForceAnnotation := makeInputRuntimeWithAnnotation(map[string]string{"operator.kyma-project.io/force-patch-reconciliation": "true", "operator.kyma-project.io/existing-annotation": "true"})
	inputRuntime := makeInputRuntimeWithAnnotation(map[string]string{"operator.kyma-project.io/existing-annotation": "true"})

	testFunction := buildPatchTestFunction(sFnPatchExistingShoot)
	var resNil *ctrl.Result

	DescribeTable(
		"transition graph validation for sFnPatchExistingShoot success",
		testFunction,
		Entry(
			"should transition to Pending Unknown state after successful shoot patching",
			testCtx,
			setupFakeFSMForTest(testScheme, inputRuntime),
			&systemState{instance: *inputRuntime, shoot: getTestShootForPatch()},
			haveName("sFnUpdateStatus"),
			expectedAnnotations,
			getExpectedPendingUnknownState(),
			resNil,
		),
		Entry(
			"should transition to Pending Unknown state after successful patching and remove force patch annotation",
			testCtx,
			setupFakeFSMForTest(testScheme, inputRuntimeWithForceAnnotation),
			&systemState{instance: *inputRuntimeWithForceAnnotation, shoot: getTestShootForPatch()},
			haveName("sFnUpdateStatus"),
			expectedAnnotations,
			getExpectedPendingUnknownState(),
			resNil,
		),
		Entry(
			"should transition to Pending Unknown state after successful patching when Audit Logs are mandatory and Audit Log Config can be read",
			testCtx,
			setupFakeFSMForTestWithAuditLogMandatoryAndConfig(testScheme, inputRuntime),
			&systemState{instance: *inputRuntime, shoot: getTestShootForPatch()},
			haveName("sFnUpdateStatus"),
			expectedAnnotations,
			getExpectedPendingUnknownState(),
			resNil,
		),
		Entry(
			"should transition to Failed state when Audit Logs are mandatory and Audit Log Config cannot be read",
			testCtx,
			setupFakeFSMForTestWithAuditLogMandatory(testScheme, inputRuntime),
			&systemState{instance: *inputRuntime, shoot: getTestShootForPatch()},
			haveName("sFnUpdateStatus"),
			expectedAnnotations,
			getExpectedPendingStateAuditLogError(),
			resNil,
		),
		Entry(
			"should transition to handleKubeconfig state when shoot generation is identical",
			testCtx,
			setupFakeFSMForTestKeepGeneration(testScheme, inputRuntime),
			&systemState{instance: *inputRuntime, shoot: getTestShootForPatch()},
			haveName("sFnHandleKubeconfig"),
			expectedAnnotations,
			getExpectedPendingNoChangedState(),
			resNil,
		),
	)
})

func getTestShootForPatch() *gardener.Shoot {
	return &gardener.Shoot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-shoot",
			Namespace: "garden-",
		},
		Spec: gardener.ShootSpec{
			DNS: &gardener.DNS{
				Domain: ptr.To("test-domain"),
			},
			Provider: gardener.Provider{
				Workers: fixWorkers("test-worker", "m5.xlarge", "garden-linux", "1.19.8", 1, 1, []string{"europe-west1-d"}),
			},
		},
		Status: gardener.ShootStatus{
			LastOperation: &gardener.LastOperation{
				State: gardener.LastOperationStateSucceeded,
			},
		},
	}
}

func getExpectedPendingUnknownState() imv1.RuntimeStatus {
	var result imv1.RuntimeStatus
	result.State = imv1.RuntimeStatePending
	result.ProvisioningCompleted = false

	condition := metav1.Condition{
		Type:    string(imv1.ConditionTypeRuntimeProvisioned),
		Status:  metav1.ConditionStatus("Unknown"),
		Reason:  string(imv1.ConditionReasonProcessing),
		Message: "Shoot is pending for update",
	}
	meta.SetStatusCondition(&result.Conditions, condition)
	return result
}

func getExpectedPendingNoChangedState() imv1.RuntimeStatus {
	var result imv1.RuntimeStatus
	result.State = imv1.RuntimeStatePending
	result.ProvisioningCompleted = false

	condition := metav1.Condition{
		Type:    string(imv1.ConditionTypeRuntimeProvisioned),
		Status:  metav1.ConditionStatus("True"),
		Reason:  string(imv1.ConditionReasonProcessing),
		Message: "Shoot updated without changes",
	}
	meta.SetStatusCondition(&result.Conditions, condition)
	return result
}

func getExpectedPendingStateAuditLogError() imv1.RuntimeStatus {
	var result imv1.RuntimeStatus
	result.State = imv1.RuntimeStateFailed
	result.ProvisioningCompleted = false

	condition := metav1.Condition{
		Type:    string(imv1.ConditionTypeRuntimeProvisioned),
		Status:  metav1.ConditionStatus("False"),
		Reason:  string(imv1.ConditionReasonAuditLogError),
		Message: "Failed to configure audit logs",
	}
	meta.SetStatusCondition(&result.Conditions, condition)
	return result
}

func setupFakeFSMForTest(scheme *runtime.Scheme, runtime *imv1.Runtime) *fsm {
	return must(newFakeFSM,
		withMockedMetrics(),
		withTestFinalizer,
		withFakedK8sClient(scheme, runtime),
		withFakeEventRecorder(1),
		withDefaultReconcileDuration(),
	)
}

func setupFakeFSMForTestKeepGeneration(scheme *runtime.Scheme, runtime *imv1.Runtime) *fsm {
	return must(newFakeFSM,
		withMockedMetrics(),
		withTestFinalizer,
		withFakedK8sClientKeepGeneration(scheme, runtime),
		withFakeEventRecorder(1),
		withDefaultReconcileDuration(),
	)
}

func setupFakeFSMForTestWithAuditLogMandatory(scheme *runtime.Scheme, runtime *imv1.Runtime) *fsm {
	return must(newFakeFSM,
		withMockedMetrics(),
		withTestFinalizer,
		withFakedK8sClient(scheme, runtime),
		withFakeEventRecorder(1),
		withDefaultReconcileDuration(),
		withAuditLogMandatory(true),
	)
}

func setupFakeFSMForTestWithAuditLogMandatoryAndConfig(scheme *runtime.Scheme, runtime *imv1.Runtime) *fsm {
	return must(newFakeFSM,
		withMockedMetrics(),
		withTestFinalizer,
		withFakedK8sClient(scheme, runtime),
		withFakeEventRecorder(1),
		withDefaultReconcileDuration(),
		withAuditLogMandatory(true),
		withAuditLogConfig("gcp", "region", auditlogs.AuditLogData{
			TenantID:   "test-tenant",
			ServiceURL: "http://test-auditlog-service",
			SecretName: "test-secret",
		}),
	)
}

func buildPatchTestFunction(fn stateFn) func(context.Context, *fsm, *systemState, types.GomegaMatcher, map[string]string, imv1.RuntimeStatus, *ctrl.Result) {
	return func(ctx context.Context, r *fsm, s *systemState, matchNextFnState types.GomegaMatcher, expectedAnnotations map[string]string, expectedStatus imv1.RuntimeStatus, expectedResult *ctrl.Result) {

		createErr := r.ShootClient.Create(ctx, s.shoot)
		if createErr != nil {
			return
		}

		sFn, res, err := fn(ctx, r, s)

		Expect(err).To(BeNil())
		Expect(res).To(Equal(expectedResult))

		if s.instance.Status.Conditions != nil {
			Expect(len(s.instance.Status.Conditions)).To(Equal(len(expectedStatus.Conditions)))
			for i := range s.instance.Status.Conditions {
				s.instance.Status.Conditions[i].LastTransitionTime = metav1.Time{}
				expectedStatus.Conditions[i].LastTransitionTime = metav1.Time{}
			}
		}

		Expect(s.instance.Status).To(Equal(expectedStatus))
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
