package fsm

import (
	"context"
	"fmt"
	"testing"
	"time"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	fsm_testing "github.com/kyma-project/infrastructure-manager/internal/controller/runtime/fsm/testing"
	"github.com/kyma-project/infrastructure-manager/pkg/auditlog"
	auditlogmocks "github.com/kyma-project/infrastructure-manager/pkg/auditlog/mocks"
	registrycachev1beta1 "github.com/kyma-project/registry-cache/api/v1beta1"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	core_v1 "k8s.io/api/core/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	api "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	util "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	//nolint:revive
	. "github.com/onsi/gomega" //nolint:revive
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
		if entry.systemState.instance.Status.ShootLastOperation != nil {
			entry.systemState.instance.Status.ShootLastOperation = &gardener.LastOperation{
				LastUpdateTime: metav1.NewTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
			}
			entry.expected.status.ShootLastOperation = &gardener.LastOperation{
				LastUpdateTime: metav1.NewTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
			}
		}

		Expect(entry.systemState.instance.Status).To(Equal(entry.expected.status))
		Expect(sFn).To(entry.expected.nextStep)
		Expect(entry.systemState.instance.GetAnnotations()).To(Equal(entry.expected.annotations))
	}
}

func setupFakeFSMForTest(scheme *api.Scheme, objs ...client.Object) *fsm {
	mockProvider := newMockAuditLogDataProvider(auditlog.AuditLogData{
		TenantID:   "test-tenant",
		ServiceURL: "http://test-service",
		SecretName: "test-secret",
	})
	return must(newFakeFSM,
		withMockedMetrics(),
		withTestFinalizer,
		withShootNamespace("garden-"),
		withFakedK8sClient(scheme, objs...),
		withFakeEventRecorder(1),
		withDefaultReconcileDuration(),
		withAuditLogDataProvider(mockProvider),
	)
}

func setupFakeFSMUpdatePatchForTest(scheme *api.Scheme, objs ...client.Object) *fsm {
	mockProvider := newMockAuditLogDataProvider(auditlog.AuditLogData{
		TenantID:   "test-tenant",
		ServiceURL: "http://test-service",
		SecretName: "test-secret",
	})
	return must(newFakeFSM,
		withMockedMetrics(),
		withTestFinalizer,
		withShootNamespace("garden-"),
		withFakedK8sClientWithFakeUpdateAndPatch(scheme, objs...),
		withFakeEventRecorder(1),
		withDefaultReconcileDuration(),
		withAuditLogDataProvider(mockProvider),
	)
}

func setupFakeFSMForTestKeepGeneration(scheme *api.Scheme, runtime *imv1.Runtime) *fsm {
	mockProvider := newMockAuditLogDataProvider(auditlog.AuditLogData{
		TenantID:   "test-tenant",
		ServiceURL: "http://test-service",
		SecretName: "test-secret",
	})
	return must(newFakeFSM,
		withMockedMetrics(),
		withShootNamespace("garden-"),
		withTestFinalizer,
		withFakedK8sClientKeepGeneration(scheme, runtime),
		withFakeEventRecorder(1),
		withDefaultReconcileDuration(),
		withAuditLogDataProvider(mockProvider),
	)
}

func setupFakeFSMForTestWithFailingPatchWithInConflictError(scheme *api.Scheme, runtime *imv1.Runtime) *fsm {
	gr := schema.GroupResource{Group: "core.gardener.cloud", Resource: "shoot"}
	err := k8s_errors.NewConflict(gr, "test-shoot", errors.New("test conflict"))

	mockProvider := newMockAuditLogDataProvider(auditlog.AuditLogData{
		TenantID:   "test-tenant",
		ServiceURL: "http://test-service",
		SecretName: "test-secret",
	})
	return must(newFakeFSM,
		withMockedMetrics(),
		withShootNamespace("garden-"),
		withTestFinalizer,
		withFakedK8sClientFailPatchError(err, scheme, runtime),
		withFakeEventRecorder(1),
		withDefaultReconcileDuration(),
		withAuditLogDataProvider(mockProvider),
	)
}

func setupFakeFSMForTestWithFailingUpdateWithInConflictError(scheme *api.Scheme, runtime *imv1.Runtime) *fsm {
	gr := schema.GroupResource{Group: "core.gardener.cloud", Resource: "shoot"}
	err := k8s_errors.NewConflict(gr, "test-shoot", errors.New("test conflict"))

	mockProvider := newMockAuditLogDataProvider(auditlog.AuditLogData{
		TenantID:   "test-tenant",
		ServiceURL: "http://test-service",
		SecretName: "test-secret",
	})
	return must(newFakeFSM,
		withMockedMetrics(),
		withShootNamespace("garden-"),
		withTestFinalizer,
		withFakedK8sClientFailUpdateError(err, scheme, runtime),
		withFakeEventRecorder(1),
		withDefaultReconcileDuration(),
		withAuditLogDataProvider(mockProvider),
	)
}

func setupFakeFSMForTestWithFailingPatchWithForbiddenError(scheme *api.Scheme, runtime *imv1.Runtime) *fsm {
	gr := schema.GroupResource{Group: "core.gardener.cloud", Resource: "shoot"}
	err := k8s_errors.NewForbidden(gr, "test-shoot", errors.New("test forbidden"))

	mockProvider := newMockAuditLogDataProvider(auditlog.AuditLogData{
		TenantID:   "test-tenant",
		ServiceURL: "http://test-service",
		SecretName: "test-secret",
	})
	return must(newFakeFSM,
		withMockedMetrics(),
		withShootNamespace("garden-"),
		withTestFinalizer,
		withFakedK8sClientFailPatchError(err, scheme, runtime),
		withFakeEventRecorder(1),
		withDefaultReconcileDuration(),
		withAuditLogDataProvider(mockProvider),
	)
}

func setupFakeFSMForTestWithFailingUpdateWithForbiddenError(scheme *api.Scheme, runtime *imv1.Runtime) *fsm {
	gr := schema.GroupResource{Group: "core.gardener.cloud", Resource: "shoot"}
	err := k8s_errors.NewForbidden(gr, "test-shoot", errors.New("test forbidden"))

	mockProvider := newMockAuditLogDataProvider(auditlog.AuditLogData{
		TenantID:   "test-tenant",
		ServiceURL: "http://test-service",
		SecretName: "test-secret",
	})
	return must(newFakeFSM,
		withMockedMetrics(),
		withShootNamespace("garden-"),
		withTestFinalizer,
		withFakedK8sClientFailUpdateError(err, scheme, runtime),
		withFakeEventRecorder(1),
		withDefaultReconcileDuration(),
		withAuditLogDataProvider(mockProvider),
	)
}

func setupFakeFSMForTestWithFailingPatchWithOtherError(scheme *api.Scheme, runtime *imv1.Runtime) *fsm {
	err := k8s_errors.NewUnauthorized("test unauthorized")

	mockProvider := newMockAuditLogDataProvider(auditlog.AuditLogData{
		TenantID:   "test-tenant",
		ServiceURL: "http://test-service",
		SecretName: "test-secret",
	})
	return must(newFakeFSM,
		withMockedMetrics(),
		withShootNamespace("garden-"),
		withTestFinalizer,
		withFakedK8sClientFailPatchError(err, scheme, runtime),
		withFakeEventRecorder(1),
		withDefaultReconcileDuration(),
		withAuditLogDataProvider(mockProvider),
	)
}

func setupFakeFSMForTestWithFailingUpdateWithOtherError(scheme *api.Scheme, runtime *imv1.Runtime) *fsm {
	err := k8s_errors.NewUnauthorized("test unauthorized")

	mockProvider := newMockAuditLogDataProvider(auditlog.AuditLogData{
		TenantID:   "test-tenant",
		ServiceURL: "http://test-service",
		SecretName: "test-secret",
	})
	return must(newFakeFSM,
		withMockedMetrics(),
		withShootNamespace("garden-"),
		withTestFinalizer,
		withFakedK8sClientFailUpdateError(err, scheme, runtime),
		withFakeEventRecorder(1),
		withDefaultReconcileDuration(),
		withAuditLogDataProvider(mockProvider),
	)
}

func setupFakeFSMForTestWithAuditLogMandatory(scheme *api.Scheme, runtime *imv1.Runtime) *fsm {
	mockProvider := newMockAuditLogDataProviderWithError()
	return must(newFakeFSM,
		withMockedMetrics(),
		withShootNamespace("garden-"),
		withTestFinalizer,
		withFakedK8sClient(scheme, runtime),
		withFakeEventRecorder(1),
		withDefaultReconcileDuration(),
		withAuditLogMandatory(true),
		withAuditLogDataProvider(mockProvider),
	)
}

func setupFakeFSMForTestWithAuditLogMandatoryAndConfig(scheme *api.Scheme, runtime *imv1.Runtime) *fsm {
	mockProvider := newMockAuditLogDataProvider(auditlog.AuditLogData{
		TenantID:   "test-tenant",
		ServiceURL: "http://test-auditlog-service",
		SecretName: "test-secret",
	})
	return must(newFakeFSM,
		withMockedMetrics(),
		withShootNamespace("garden-"),
		withTestFinalizer,
		withFakedK8sClient(scheme, runtime),
		withFakeEventRecorder(1),
		withDefaultReconcileDuration(),
		withAuditLogMandatory(true),
		withAuditLogDataProvider(mockProvider),
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

// newMockAuditLogDataProvider creates a mock DataProvider that returns the given data
func newMockAuditLogDataProvider(data auditlog.AuditLogData) *auditlogmocks.DataProvider {
	mockProvider := &auditlogmocks.DataProvider{}
	mockProvider.On("ReserveAuditLog", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockProvider.On("GetDedicatedAuditLogData", mock.Anything, mock.Anything, mock.Anything).Return(data, nil)
	mockProvider.On("GetSharedAuditLogData", mock.Anything, mock.Anything, mock.Anything).Return(data, nil)
	mockProvider.On("ReleaseDedicated", mock.Anything, mock.Anything).Return(nil)
	return mockProvider
}

// newMockAuditLogDataProviderWithError creates a mock DataProvider that returns errors
func newMockAuditLogDataProviderWithError() *auditlogmocks.DataProvider {
	mockProvider := &auditlogmocks.DataProvider{}
	mockProvider.On("ReserveAuditLog", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("mock audit log reservation error"))
	mockProvider.On("GetDedicatedAuditLogData", mock.Anything, mock.Anything, mock.Anything).Return(auditlog.AuditLogData{}, fmt.Errorf("mock audit log error"))
	mockProvider.On("GetSharedAuditLogData", mock.Anything, mock.Anything, mock.Anything).Return(auditlog.AuditLogData{}, fmt.Errorf("mock audit log error"))
	mockProvider.On("ReleaseDedicated", mock.Anything, mock.Anything).Return(nil)
	return mockProvider
}

// Tests for dedicated audit logging upgrade scenarios
func TestSFnPatchExistingShoot_DedicatedAuditLogUpgrade(t *testing.T) {
	testCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	testScheme := api.NewScheme()
	util.Must(imv1.AddToScheme(testScheme))
	util.Must(gardener.AddToScheme(testScheme))
	util.Must(core_v1.AddToScheme(testScheme))

	RegisterTestingT(t)

	t.Run("should claim AuditLogCR when upgrading to dedicated and no existing claim", func(t *testing.T) {
		// Given: Runtime with auditLogAccessEnabled=true, no existing claim
		inputRuntime := makeInputRuntimeWithDedicatedAuditLog(true)
		mockProvider := &auditlogmocks.DataProvider{}

		// GetDedicatedAuditLogData returns error (no existing claim)
		mockProvider.On("GetDedicatedAuditLogData", mock.Anything, mock.Anything, false).
			Return(auditlog.AuditLogData{}, fmt.Errorf("no AuditLogCR found"))
		// ClaimAuditLog succeeds and returns dedicated config
		mockProvider.On("ClaimAuditLog", mock.Anything, mock.Anything, mock.Anything).
			Return(auditlog.AuditLogData{
				TenantID:   "dedicated-tenant",
				ServiceURL: "http://dedicated-service",
				SecretName: "dedicated-secret",
			}, nil)

		f := must(newFakeFSM,
			withMockedMetrics(),
			withShootNamespace("garden-"),
			withTestFinalizer,
			withFakedK8sClient(testScheme, inputRuntime),
			withFakeEventRecorder(1),
			withDefaultReconcileDuration(),
			withDedicatedAuditLoggingEnabled(true),
			withAuditLogDataProvider(mockProvider),
		)

		shoot := fsm_testing.TestShootForPatch()
		createErr := f.GardenClient.Create(testCtx, shoot)
		Expect(createErr).To(BeNil())

		// When
		state := &systemState{instance: *inputRuntime, shoot: shoot}
		sFn, _, err := sFnPatchExistingShoot(testCtx, f, state)

		// Then
		Expect(err).To(BeNil())
		Expect(sFn).To(haveName("sFnUpdateStatus"))
		mockProvider.AssertCalled(t, "ClaimAuditLog", mock.Anything, "region", mock.Anything)

		// Verify condition is set to Unknown (configuring shoot)
		condition := findCondition(state.instance.Status.Conditions, imv1.ConditionTypeCustomAuditLogConfigured)
		Expect(condition).NotTo(BeNil())
		Expect(condition.Status).To(Equal(metav1.ConditionUnknown))
	})

	t.Run("should use existing dedicated config when already claimed", func(t *testing.T) {
		// Given: Runtime with auditLogAccessEnabled=true, existing claim
		inputRuntime := makeInputRuntimeWithDedicatedAuditLog(true)
		mockProvider := &auditlogmocks.DataProvider{}

		// GetDedicatedAuditLogData returns existing data (already claimed)
		mockProvider.On("GetDedicatedAuditLogData", mock.Anything, mock.Anything, false).
			Return(auditlog.AuditLogData{
				TenantID:   "dedicated-tenant",
				ServiceURL: "http://dedicated-service",
				SecretName: "dedicated-secret",
			}, nil)

		f := must(newFakeFSM,
			withMockedMetrics(),
			withShootNamespace("garden-"),
			withTestFinalizer,
			withFakedK8sClient(testScheme, inputRuntime),
			withFakeEventRecorder(1),
			withDefaultReconcileDuration(),
			withDedicatedAuditLoggingEnabled(true),
			withAuditLogDataProvider(mockProvider),
		)

		shoot := fsm_testing.TestShootForPatch()
		createErr := f.GardenClient.Create(testCtx, shoot)
		Expect(createErr).To(BeNil())

		// When
		sFn, _, err := sFnPatchExistingShoot(testCtx, f, &systemState{instance: *inputRuntime, shoot: shoot})

		// Then
		Expect(err).To(BeNil())
		Expect(sFn).To(haveName("sFnUpdateStatus"))
		mockProvider.AssertNotCalled(t, "ClaimAuditLog", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("should fail when pool exhausted and dedicated logging requested", func(t *testing.T) {
		// Given: Runtime with auditLogAccessEnabled=true, no pool capacity
		inputRuntime := makeInputRuntimeWithDedicatedAuditLog(true)
		mockProvider := &auditlogmocks.DataProvider{}

		// GetDedicatedAuditLogData returns error (no existing claim)
		mockProvider.On("GetDedicatedAuditLogData", mock.Anything, mock.Anything, false).
			Return(auditlog.AuditLogData{}, fmt.Errorf("no AuditLogCR found"))
		// ClaimAuditLog fails (pool exhausted)
		mockProvider.On("ClaimAuditLog", mock.Anything, mock.Anything, mock.Anything).
			Return(auditlog.AuditLogData{}, fmt.Errorf("no available AuditLogCR in the pool"))

		f := must(newFakeFSM,
			withMockedMetrics(),
			withShootNamespace("garden-"),
			withTestFinalizer,
			withFakedK8sClient(testScheme, inputRuntime),
			withFakeEventRecorder(1),
			withDefaultReconcileDuration(),
			withDedicatedAuditLoggingEnabled(true),
			withAuditLogDataProvider(mockProvider),
		)

		shoot := fsm_testing.TestShootForPatch()
		createErr := f.GardenClient.Create(testCtx, shoot)
		Expect(createErr).To(BeNil())

		// When
		state := &systemState{instance: *inputRuntime, shoot: shoot}
		sFn, _, err := sFnPatchExistingShoot(testCtx, f, state)

		// Then
		Expect(err).To(BeNil())
		Expect(sFn).To(haveName("sFnUpdateStatus"))
		Expect(string(state.instance.Status.State)).To(Equal(imv1.RuntimeStateFailed))
	})

	t.Run("should ignore downgrade attempt when dedicated logging already configured (irreversibility)", func(t *testing.T) {
		// Given: Runtime with auditLogAccessEnabled=false, but existing dedicated claim
		inputRuntime := makeInputRuntimeWithDedicatedAuditLog(false) // User trying to disable
		mockProvider := &auditlogmocks.DataProvider{}

		// GetDedicatedAuditLogData returns existing data (dedicated already configured)
		mockProvider.On("GetDedicatedAuditLogData", mock.Anything, mock.Anything, false).
			Return(auditlog.AuditLogData{
				TenantID:   "dedicated-tenant",
				ServiceURL: "http://dedicated-service",
				SecretName: "dedicated-secret",
			}, nil)

		f := must(newFakeFSM,
			withMockedMetrics(),
			withShootNamespace("garden-"),
			withTestFinalizer,
			withFakedK8sClient(testScheme, inputRuntime),
			withFakeEventRecorder(1),
			withDefaultReconcileDuration(),
			withDedicatedAuditLoggingEnabled(true),
			withAuditLogDataProvider(mockProvider),
		)

		shoot := fsm_testing.TestShootForPatch()
		createErr := f.GardenClient.Create(testCtx, shoot)
		Expect(createErr).To(BeNil())

		// When
		sFn, _, err := sFnPatchExistingShoot(testCtx, f, &systemState{instance: *inputRuntime, shoot: shoot})

		// Then: Should continue with dedicated config (downgrade ignored)
		Expect(err).To(BeNil())
		Expect(sFn).To(haveName("sFnUpdateStatus"))
		// Should NOT call GetSharedAuditLogData or ClaimAuditLog since we're using existing dedicated
		mockProvider.AssertNotCalled(t, "GetSharedAuditLogData", mock.Anything, mock.Anything, mock.Anything)
		mockProvider.AssertNotCalled(t, "ClaimAuditLog", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("should use shared config when no previous dedicated config exists and auditLogAccessEnabled is false", func(t *testing.T) {
		// Given: Runtime with auditLogAccessEnabled=false, no existing dedicated claim
		inputRuntime := makeInputRuntimeWithDedicatedAuditLog(false)
		mockProvider := &auditlogmocks.DataProvider{}

		// GetDedicatedAuditLogData returns error (no existing dedicated config)
		mockProvider.On("GetDedicatedAuditLogData", mock.Anything, mock.Anything, false).
			Return(auditlog.AuditLogData{}, fmt.Errorf("no AuditLogCR found"))
		// GetSharedAuditLogData returns shared config
		mockProvider.On("GetSharedAuditLogData", mock.Anything, mock.Anything, mock.Anything).
			Return(auditlog.AuditLogData{
				TenantID:   "shared-tenant",
				ServiceURL: "http://shared-service",
				SecretName: "shared-secret",
			}, nil)

		f := must(newFakeFSM,
			withMockedMetrics(),
			withShootNamespace("garden-"),
			withTestFinalizer,
			withFakedK8sClient(testScheme, inputRuntime),
			withFakeEventRecorder(1),
			withDefaultReconcileDuration(),
			withDedicatedAuditLoggingEnabled(true),
			withAuditLogDataProvider(mockProvider),
		)

		shoot := fsm_testing.TestShootForPatch()
		createErr := f.GardenClient.Create(testCtx, shoot)
		Expect(createErr).To(BeNil())

		// When
		sFn, _, err := sFnPatchExistingShoot(testCtx, f, &systemState{instance: *inputRuntime, shoot: shoot})

		// Then
		Expect(err).To(BeNil())
		Expect(sFn).To(haveName("sFnUpdateStatus"))
		mockProvider.AssertCalled(t, "GetSharedAuditLogData", mock.Anything, mock.Anything, mock.Anything)
		mockProvider.AssertNotCalled(t, "ClaimAuditLog", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("should use shared config when global dedicated feature is disabled", func(t *testing.T) {
		// Given: Runtime with auditLogAccessEnabled=true, but global feature disabled
		inputRuntime := makeInputRuntimeWithDedicatedAuditLog(true)
		mockProvider := &auditlogmocks.DataProvider{}

		// GetSharedAuditLogData returns shared config
		mockProvider.On("GetSharedAuditLogData", mock.Anything, mock.Anything, mock.Anything).
			Return(auditlog.AuditLogData{
				TenantID:   "shared-tenant",
				ServiceURL: "http://shared-service",
				SecretName: "shared-secret",
			}, nil)

		f := must(newFakeFSM,
			withMockedMetrics(),
			withShootNamespace("garden-"),
			withTestFinalizer,
			withFakedK8sClient(testScheme, inputRuntime),
			withFakeEventRecorder(1),
			withDefaultReconcileDuration(),
			withDedicatedAuditLoggingEnabled(false), // Global feature disabled
			withAuditLogDataProvider(mockProvider),
		)

		shoot := fsm_testing.TestShootForPatch()
		createErr := f.GardenClient.Create(testCtx, shoot)
		Expect(createErr).To(BeNil())

		// When
		sFn, _, err := sFnPatchExistingShoot(testCtx, f, &systemState{instance: *inputRuntime, shoot: shoot})

		// Then
		Expect(err).To(BeNil())
		Expect(sFn).To(haveName("sFnUpdateStatus"))
		mockProvider.AssertCalled(t, "GetSharedAuditLogData", mock.Anything, mock.Anything, mock.Anything)
		// Should NOT check for dedicated or reserve
		mockProvider.AssertNotCalled(t, "GetDedicatedAuditLogData", mock.Anything, mock.Anything, mock.Anything)
		mockProvider.AssertNotCalled(t, "ReserveAuditLog", mock.Anything, mock.Anything, mock.Anything)
	})
}

// makeInputRuntimeWithDedicatedAuditLog creates a runtime with auditLogAccessEnabled set
func makeInputRuntimeWithDedicatedAuditLog(enabled bool) *imv1.Runtime {
	runtime := makeInputRuntimeWithAnnotation(map[string]string{"operator.kyma-project.io/existing-annotation": "true"})
	runtime.Spec.AuditLogAccessEnabled = ptr.To(enabled)
	return runtime
}

// findCondition finds a condition by type in the conditions slice
func findCondition(conditions []metav1.Condition, conditionType imv1.RuntimeConditionType) *metav1.Condition {
	for i := range conditions {
		if conditions[i].Type == string(conditionType) {
			return &conditions[i]
		}
	}
	return nil
}
