package fsm

import (
	"context"
	fsm_testing "github.com/kyma-project/infrastructure-manager/internal/controller/runtime/fsm/testing"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/auditlogs"
	registrycachev1beta1 "github.com/kyma-project/registry-cache/api/v1beta1"
	"github.com/pkg/errors"
	core_v1 "k8s.io/api/core/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
	"time"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	//nolint:revive
	. "github.com/onsi/gomega" //nolint:revive
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	api "k8s.io/apimachinery/pkg/runtime"
	util "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/utils/ptr"
)

func TestFSMPatchShoot(t *testing.T) {
	testCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	testScheme := api.NewScheme()

	util.Must(imv1.AddToScheme(testScheme))
	util.Must(gardener.AddToScheme(testScheme))
	util.Must(core_v1.AddToScheme(testScheme))

	testSchemeWithRegistryCache := api.NewScheme()
	util.Must(imv1.AddToScheme(testSchemeWithRegistryCache))
	util.Must(gardener.AddToScheme(testSchemeWithRegistryCache))
	util.Must(core_v1.AddToScheme(testSchemeWithRegistryCache))
	util.Must(registrycachev1beta1.AddToScheme(testSchemeWithRegistryCache))

	expectedAnnotations := map[string]string{"operator.kyma-project.io/existing-annotation": "true"}
	inputRuntimeWithForceAnnotation := makeInputRuntimeWithAnnotation(map[string]string{"operator.kyma-project.io/force-patch-reconciliation": "true", "operator.kyma-project.io/existing-annotation": "true"})
	inputRuntime := makeInputRuntimeWithAnnotation(map[string]string{"operator.kyma-project.io/existing-annotation": "true"})
	inputRuntimeWithRegistryCacheEnabled := inputRuntime.DeepCopy()
	inputRuntimeWithRegistryCacheEnabled.Spec.Caching = []imv1.ImageRegistryCache{
		{
			Name:      "config1",
			Namespace: "test1",
			Config:    registrycachev1beta1.RegistryCacheConfigSpec{},
		},
	}

	RegisterTestingT(t)

	for _, entry := range []struct {
		description string
		fsm         *fsm
		systemState *systemState
		expected    outputFnState
	}{
		{
			"should transition to Pending Unknown state after successful shoot patching",
			setupFakeFSMForTest(testScheme, inputRuntime),
			&systemState{instance: *inputRuntime, shoot: fsm_testing.TestShootForPatch()},
			outputFnState{
				nextStep:    haveName("sFnUpdateStatus"),
				annotations: expectedAnnotations,
				result:      nil,
				status:      fsm_testing.PendingStatusShootPatched(),
			},
		},
		{
			"should transition to Pending Unknown state after successful patching and remove force patch annotation",
			setupFakeFSMForTest(testScheme, inputRuntimeWithForceAnnotation),
			&systemState{instance: *inputRuntimeWithForceAnnotation, shoot: fsm_testing.TestShootForPatch()},
			outputFnState{
				nextStep:    haveName("sFnUpdateStatus"),
				annotations: expectedAnnotations,
				result:      nil,
				status:      fsm_testing.PendingStatusShootPatched(),
			},
		},
		{
			"should transition to Failed state when Audit Logs are mandatory and Audit Log Config cannot be read",
			setupFakeFSMForTestWithAuditLogMandatory(testScheme, inputRuntime),
			&systemState{instance: *inputRuntime, shoot: fsm_testing.TestShootForPatch()},
			outputFnState{
				nextStep:    haveName("sFnUpdateStatus"),
				annotations: expectedAnnotations,
				result:      nil,
				status:      fsm_testing.FailedStatusAuditLogError(),
			},
		},
		{
			"should transition to Pending Unknown state after successful patching when Audit Logs are mandatory and Audit Log Config can be read",
			setupFakeFSMForTestWithAuditLogMandatoryAndConfig(testScheme, inputRuntime),
			&systemState{instance: *inputRuntime, shoot: fsm_testing.TestShootForPatch()},
			outputFnState{
				nextStep:    haveName("sFnUpdateStatus"),
				annotations: expectedAnnotations,
				result:      nil,
				status:      fsm_testing.PendingStatusShootPatched(),
			},
		},
		{
			"should transition to handleKubeconfig state when shoot generation is identical",
			setupFakeFSMForTestKeepGeneration(testScheme, inputRuntime),
			&systemState{instance: *inputRuntime, shoot: fsm_testing.TestShootForPatch()},
			outputFnState{
				nextStep:    haveName("sFnHandleKubeconfig"),
				annotations: expectedAnnotations,
				result:      nil,
				status:      fsm_testing.PendingStatusShootNoChanged(),
			},
		},
		{
			"should transition to Pending Unknown when cannot execute Patch shoot with inConflict error",
			setupFakeFSMForTestWithFailingPatchWithInConflictError(testScheme, inputRuntime),
			&systemState{instance: *inputRuntime, shoot: fsm_testing.TestShootForPatch()},
			outputFnState{
				nextStep:    haveName("sFnUpdateStatus"),
				annotations: expectedAnnotations,
				result:      nil,
				status:      fsm_testing.PendingStatusAfterConflictErr(),
			},
		},
		{
			"should transition to Pending Unknown when cannot execute Patch shoot with forbidden error",
			setupFakeFSMForTestWithFailingPatchWithForbiddenError(testScheme, inputRuntime),
			&systemState{instance: *inputRuntime, shoot: fsm_testing.TestShootForPatch()},
			outputFnState{
				nextStep:    haveName("sFnUpdateStatus"),
				annotations: expectedAnnotations,
				result:      nil,
				status:      fsm_testing.PendingStatusAfterForbiddenErr(),
			},
		},
		{
			"should transition to Failed state when cannot execute Patch shoot with any other error",
			setupFakeFSMForTestWithFailingPatchWithOtherError(testScheme, inputRuntime),
			&systemState{instance: *inputRuntime, shoot: fsm_testing.TestShootForPatch()},
			outputFnState{
				nextStep:    haveName("sFnUpdateStatus"),
				annotations: expectedAnnotations,
				result:      nil,
				status:      fsm_testing.FailedStatusPatchErr(),
			},
		},
		{
			"should transition to Pending Unknown when cannot execute Update shoot with inConflict error",
			setupFakeFSMForTestWithFailingUpdateWithInConflictError(testScheme, inputRuntime),
			&systemState{instance: *inputRuntime, shoot: fsm_testing.TestShootForUpdate()},
			outputFnState{
				nextStep:    haveName("sFnUpdateStatus"),
				annotations: expectedAnnotations,
				result:      nil,
				status:      fsm_testing.PendingStatusAfterConflictErr(),
			},
		},
		{
			"should transition to Pending Unknown when cannot execute Update shoot with forbidden error",
			setupFakeFSMForTestWithFailingUpdateWithForbiddenError(testScheme, inputRuntime),
			&systemState{instance: *inputRuntime, shoot: fsm_testing.TestShootForUpdate()},
			outputFnState{
				nextStep:    haveName("sFnUpdateStatus"),
				annotations: expectedAnnotations,
				result:      nil,
				status:      fsm_testing.PendingStatusAfterForbiddenErr(),
			},
		},
		{
			"should transition to Failed state when cannot execute Update shoot with any other error",
			setupFakeFSMForTestWithFailingUpdateWithOtherError(testScheme, inputRuntime),
			&systemState{instance: *inputRuntime, shoot: fsm_testing.TestShootForUpdate()},
			outputFnState{
				nextStep:    haveName("sFnUpdateStatus"),
				annotations: expectedAnnotations,
				result:      nil,
				status:      fsm_testing.FailedStatusUpdateError(),
			},
		},
	} {
		createErr := entry.fsm.GardenClient.Create(testCtx, entry.systemState.shoot)
		Expect(createErr).To(BeNil())

		sFn, res, err := sFnPatchExistingShoot(testCtx, entry.fsm, entry.systemState)

		Expect(err).To(BeNil())
		Expect(res).To(Equal(entry.expected.result))

		if entry.systemState.instance.Status.Conditions != nil {
			Expect(len(entry.systemState.instance.Status.Conditions)).To(Equal(len(entry.expected.status.Conditions)))
			for i := range entry.systemState.instance.Status.Conditions {
				entry.systemState.instance.Status.Conditions[i].LastTransitionTime = metav1.Time{}
				entry.expected.status.Conditions[i].LastTransitionTime = metav1.Time{}
			}
		}

		Expect(entry.systemState.instance.Status).To(Equal(entry.expected.status))
		Expect(sFn).To(entry.expected.nextStep)
		Expect(entry.systemState.instance.GetAnnotations()).To(Equal(entry.expected.annotations))
	}
}

func setupFakeFSMForTest(scheme *api.Scheme, objs ...client.Object) *fsm {
	return must(newFakeFSM,
		withMockedMetrics(),
		withTestFinalizer,
		withShootNamespace("garden-"),
		withFakedK8sClient(scheme, objs...),
		withFakeEventRecorder(1),
		withDefaultReconcileDuration(),
	)
}

func setupFakeFSMUpdatePatchForTest(scheme *api.Scheme, objs ...client.Object) *fsm {
	return must(newFakeFSM,
		withMockedMetrics(),
		withTestFinalizer,
		withShootNamespace("garden-"),
		withFakedK8sClientWithFakeUpdateAndPatch(scheme, objs...),
		withFakeEventRecorder(1),
		withDefaultReconcileDuration(),
	)
}

func setupFakeFSMForTestKeepGeneration(scheme *api.Scheme, runtime *imv1.Runtime) *fsm {
	return must(newFakeFSM,
		withMockedMetrics(),
		withShootNamespace("garden-"),
		withTestFinalizer,
		withFakedK8sClientKeepGeneration(scheme, runtime),
		withFakeEventRecorder(1),
		withDefaultReconcileDuration(),
	)
}

func setupFakeFSMForTestWithFailingPatchWithInConflictError(scheme *api.Scheme, runtime *imv1.Runtime) *fsm {
	gr := schema.GroupResource{Group: "core.gardener.cloud", Resource: "shoot"}
	err := k8s_errors.NewConflict(gr, "test-shoot", errors.New("test conflict"))

	return must(newFakeFSM,
		withMockedMetrics(),
		withShootNamespace("garden-"),
		withTestFinalizer,
		withFakedK8sClientFailPatchError(err, scheme, runtime),
		withFakeEventRecorder(1),
		withDefaultReconcileDuration(),
	)
}

func setupFakeFSMForTestWithFailingUpdateWithInConflictError(scheme *api.Scheme, runtime *imv1.Runtime) *fsm {
	gr := schema.GroupResource{Group: "core.gardener.cloud", Resource: "shoot"}
	err := k8s_errors.NewConflict(gr, "test-shoot", errors.New("test conflict"))

	return must(newFakeFSM,
		withMockedMetrics(),
		withShootNamespace("garden-"),
		withTestFinalizer,
		withFakedK8sClientFailUpdateError(err, scheme, runtime),
		withFakeEventRecorder(1),
		withDefaultReconcileDuration(),
	)
}

func setupFakeFSMForTestWithFailingPatchWithForbiddenError(scheme *api.Scheme, runtime *imv1.Runtime) *fsm {
	gr := schema.GroupResource{Group: "core.gardener.cloud", Resource: "shoot"}
	err := k8s_errors.NewForbidden(gr, "test-shoot", errors.New("test forbidden"))

	return must(newFakeFSM,
		withMockedMetrics(),
		withShootNamespace("garden-"),
		withTestFinalizer,
		withFakedK8sClientFailPatchError(err, scheme, runtime),
		withFakeEventRecorder(1),
		withDefaultReconcileDuration(),
	)
}

func setupFakeFSMForTestWithFailingUpdateWithForbiddenError(scheme *api.Scheme, runtime *imv1.Runtime) *fsm {
	gr := schema.GroupResource{Group: "core.gardener.cloud", Resource: "shoot"}
	err := k8s_errors.NewForbidden(gr, "test-shoot", errors.New("test forbidden"))

	return must(newFakeFSM,
		withMockedMetrics(),
		withShootNamespace("garden-"),
		withTestFinalizer,
		withFakedK8sClientFailUpdateError(err, scheme, runtime),
		withFakeEventRecorder(1),
		withDefaultReconcileDuration(),
	)
}

func setupFakeFSMForTestWithFailingPatchWithOtherError(scheme *api.Scheme, runtime *imv1.Runtime) *fsm {
	err := k8s_errors.NewUnauthorized("test unauthorized")

	return must(newFakeFSM,
		withMockedMetrics(),
		withShootNamespace("garden-"),
		withTestFinalizer,
		withFakedK8sClientFailPatchError(err, scheme, runtime),
		withFakeEventRecorder(1),
		withDefaultReconcileDuration(),
	)
}

func setupFakeFSMForTestWithFailingUpdateWithOtherError(scheme *api.Scheme, runtime *imv1.Runtime) *fsm {
	err := k8s_errors.NewUnauthorized("test unauthorized")

	return must(newFakeFSM,
		withMockedMetrics(),
		withShootNamespace("garden-"),
		withTestFinalizer,
		withFakedK8sClientFailUpdateError(err, scheme, runtime),
		withFakeEventRecorder(1),
		withDefaultReconcileDuration(),
	)
}

func setupFakeFSMForTestWithAuditLogMandatory(scheme *api.Scheme, runtime *imv1.Runtime) *fsm {
	return must(newFakeFSM,
		withMockedMetrics(),
		withShootNamespace("garden-"),
		withTestFinalizer,
		withFakedK8sClient(scheme, runtime),
		withFakeEventRecorder(1),
		withDefaultReconcileDuration(),
		withAuditLogMandatory(true),
	)
}

func setupFakeFSMForTestWithAuditLogMandatoryAndConfig(scheme *api.Scheme, runtime *imv1.Runtime) *fsm {
	return must(newFakeFSM,
		withMockedMetrics(),
		withShootNamespace("garden-"),
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

func Test_SFnPatchExistingShoot_CredentialsBindingPatched(t *testing.T) {
	RegisterTestingT(t)
	testCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	testScheme := api.NewScheme()
	util.Must(imv1.AddToScheme(testScheme))
	util.Must(gardener.AddToScheme(testScheme))
	util.Must(core_v1.AddToScheme(testScheme))

	// Prepare runtime and shoot
	inputRuntime := makeInputRuntimeWithAnnotation(map[string]string{"operator.kyma-project.io/existing-annotation": "true"})
	// Ensure runtime has the secret that should be propagated to CredentialsBindingName
	inputRuntime.Spec.Shoot.SecretBindingName = "runtime-secret"

	// Create FSM and enable credential binding in converter config
	f := setupFakeFSMUpdatePatchForTest(testScheme, inputRuntime)
	f.ConverterConfig.Gardener.EnableCredentialBinding = true

	// Create a shoot that has SecretBindingName set (so bindingShouldBePatched will be true)
	shoot := fsm_testing.TestShootForPatch()
	shoot.Spec.SecretBindingName = ptr.To("existing-shoot-secret") //nolint:staticcheck

	// Persist shoot into fake GardenClient
	createErr := f.GardenClient.Create(testCtx, shoot)
	Expect(createErr).To(BeNil())

	// Call the function under test
	sFn, res, err := sFnPatchExistingShoot(testCtx, f, &systemState{instance: *inputRuntime, shoot: shoot})
	Expect(err).To(BeNil())
	Expect(res).To(BeNil())

	// Fetch the Shoot from the fake GardenClient and assert it was updated with CredentialsBindingName and SecretBindingName cleared
	gotShoot := &gardener.Shoot{}
	getErr := f.GardenClient.Get(testCtx, client.ObjectKey{Name: shoot.Name, Namespace: shoot.Namespace}, gotShoot)
	Expect(getErr).To(BeNil())

	// CredentialsBindingName should be set to runtime secret and SecretBindingName should be nil
	Expect(gotShoot.Spec.CredentialsBindingName).To(Not(BeNil()))
	Expect(*gotShoot.Spec.CredentialsBindingName).To(Equal(inputRuntime.Spec.Shoot.SecretBindingName))
	Expect(gotShoot.Spec.SecretBindingName).To(BeNil()) //nolint:staticcheck

	// Next state should be update status (or other valid step); ensure no panic and a state is returned
	Expect(sFn).To(Not(BeNil()))
}
