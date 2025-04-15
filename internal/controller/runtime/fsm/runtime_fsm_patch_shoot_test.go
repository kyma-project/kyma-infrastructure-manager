package fsm

import (
	"context"
	"errors"
	fsm_testing "github.com/kyma-project/infrastructure-manager/internal/controller/runtime/fsm/testing"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/auditlogs"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/structuredauth"
	v1 "k8s.io/api/core/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
	"testing"
	"time"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	. "github.com/onsi/ginkgo/v2" //nolint:revive
	. "github.com/onsi/gomega"    //nolint:revive
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	util "k8s.io/apimachinery/pkg/util/runtime"
)

var _ = Describe("KIM sFnPatchExistingShoot", func() {

	testCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	testScheme := runtime.NewScheme()

	util.Must(imv1.AddToScheme(testScheme))
	util.Must(gardener.AddToScheme(testScheme))
	util.Must(v1.AddToScheme(testScheme))

	expectedAnnotations := map[string]string{"operator.kyma-project.io/existing-annotation": "true"}
	inputRuntimeWithForceAnnotation := makeInputRuntimeWithAnnotation(map[string]string{"operator.kyma-project.io/force-patch-reconciliation": "true", "operator.kyma-project.io/existing-annotation": "true"})
	inputRuntime := makeInputRuntimeWithAnnotation(map[string]string{"operator.kyma-project.io/existing-annotation": "true"})

	testFunction := buildPatchTestFunction(sFnPatchExistingShoot)

	DescribeTable(
		"transition graph validation for sFnPatchExistingShoot success",
		testFunction,
		Entry(
			"should transition to Pending Unknown state after successful shoot patching",
			testCtx,
			setupFakeFSMForTest(testScheme, inputRuntime),
			&systemState{instance: *inputRuntime, shoot: fsm_testing.TestShootForPatch()},
			outputFnState{
				nextStep:    haveName("sFnUpdateStatus"),
				annotations: expectedAnnotations,
				result:      nil,
				status:      fsm_testing.PendingStatusShootPatched(),
			},
		),
		Entry(
			"should transition to Pending Unknown state after successful patching and remove force patch annotation",
			testCtx,
			setupFakeFSMForTest(testScheme, inputRuntimeWithForceAnnotation),
			&systemState{instance: *inputRuntimeWithForceAnnotation, shoot: fsm_testing.TestShootForPatch()},
			outputFnState{
				nextStep:    haveName("sFnUpdateStatus"),
				annotations: expectedAnnotations,
				result:      nil,
				status:      fsm_testing.PendingStatusShootPatched(),
			},
		),
		Entry(
			"should transition to Failed state when Audit Logs are mandatory and Audit Log Config cannot be read",
			testCtx,
			setupFakeFSMForTestWithAuditLogMandatory(testScheme, inputRuntime),
			&systemState{instance: *inputRuntime, shoot: fsm_testing.TestShootForPatch()},
			outputFnState{
				nextStep:    haveName("sFnUpdateStatus"),
				annotations: expectedAnnotations,
				result:      nil,
				status:      fsm_testing.FailedStatusAuditLogError(),
			},
		),
		Entry(
			"should transition to Pending Unknown state after successful patching when Audit Logs are mandatory and Audit Log Config can be read",
			testCtx,
			setupFakeFSMForTestWithAuditLogMandatoryAndConfig(testScheme, inputRuntime),
			&systemState{instance: *inputRuntime, shoot: fsm_testing.TestShootForPatch()},
			outputFnState{
				nextStep:    haveName("sFnUpdateStatus"),
				annotations: expectedAnnotations,
				result:      nil,
				status:      fsm_testing.PendingStatusShootPatched(),
			},
		),
		Entry(
			"should transition to handleKubeconfig state when shoot generation is identical",
			testCtx,
			setupFakeFSMForTestKeepGeneration(testScheme, inputRuntime),
			&systemState{instance: *inputRuntime, shoot: fsm_testing.TestShootForPatch()},
			outputFnState{
				nextStep:    haveName("sFnHandleKubeconfig"),
				annotations: expectedAnnotations,
				result:      nil,
				status:      fsm_testing.PendingStatusShootNoChanged(),
			},
		),
		Entry(
			"should transition to Pending Unknown when cannot execute Patch shoot with inConflict error",
			testCtx,
			setupFakeFSMForTestWithFailingPatchWithInConflictError(testScheme, inputRuntime),
			&systemState{instance: *inputRuntime, shoot: fsm_testing.TestShootForPatch()},
			outputFnState{
				nextStep:    haveName("sFnUpdateStatus"),
				annotations: expectedAnnotations,
				result:      nil,
				status:      fsm_testing.PendingStatusAfterConflictErr(),
			},
		),
		Entry(
			"should transition to Pending Unknown when cannot execute Patch shoot with forbidden error",
			testCtx,
			setupFakeFSMForTestWithFailingPatchWithForbiddenError(testScheme, inputRuntime),
			&systemState{instance: *inputRuntime, shoot: fsm_testing.TestShootForPatch()},
			outputFnState{
				nextStep:    haveName("sFnUpdateStatus"),
				annotations: expectedAnnotations,
				result:      nil,
				status:      fsm_testing.PendingStatusAfterForbiddenErr(),
			},
		),
		Entry(
			"should transition to Failed state when cannot execute Patch shoot with any other error",
			testCtx,
			setupFakeFSMForTestWithFailingPatchWithOtherError(testScheme, inputRuntime),
			&systemState{instance: *inputRuntime, shoot: fsm_testing.TestShootForPatch()},
			outputFnState{
				nextStep:    haveName("sFnUpdateStatus"),
				annotations: expectedAnnotations,
				result:      nil,
				status:      fsm_testing.FailedStatusPatchErr(),
			},
		),
		Entry(
			"should transition to Pending Unknown when cannot execute Update shoot with inConflict error",
			testCtx,
			setupFakeFSMForTestWithFailingUpdateWithInConflictError(testScheme, inputRuntime),
			&systemState{instance: *inputRuntime, shoot: fsm_testing.TestShootForUpdate()},
			outputFnState{
				nextStep:    haveName("sFnUpdateStatus"),
				annotations: expectedAnnotations,
				result:      nil,
				status:      fsm_testing.PendingStatusAfterConflictErr(),
			},
		),
		Entry(
			"should transition to Pending Unknown when cannot execute Update shoot with forbidden error",
			testCtx,
			setupFakeFSMForTestWithFailingUpdateWithForbiddenError(testScheme, inputRuntime),
			&systemState{instance: *inputRuntime, shoot: fsm_testing.TestShootForUpdate()},
			outputFnState{
				nextStep:    haveName("sFnUpdateStatus"),
				annotations: expectedAnnotations,
				result:      nil,
				status:      fsm_testing.PendingStatusAfterForbiddenErr(),
			},
		),
		Entry(
			"should transition to to Failed state when cannot execute Update shoot with any other error",
			testCtx,
			setupFakeFSMForTestWithFailingUpdateWithOtherError(testScheme, inputRuntime),
			&systemState{instance: *inputRuntime, shoot: fsm_testing.TestShootForUpdate()},
			outputFnState{
				nextStep:    haveName("sFnUpdateStatus"),
				annotations: expectedAnnotations,
				result:      nil,
				status:      fsm_testing.FailedStatusUpdateError(),
			},
		),
	)

	Context("When migrating OIDC setting for existing clusters", func() {
		const ResourceName = "test-resource"
		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      ResourceName,
			Namespace: "default",
		}

		It("Should successfully nil OIDC property, and setup structured auth config", func() {
			testFunc := buildPatchTestFunction(sFnPatchExistingShoot)
			fakeFSM := setupFakeFSMForTestWithStructuredAuthEnabled(testScheme, inputRuntime)

			runtimeWithOIDC := *inputRuntime.DeepCopy()

			runtimeWithOIDC.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.ClientID = ptr.To("client-id")
			runtimeWithOIDC.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.IssuerURL = ptr.To("some.url.com")

			shootWithOIDC := fsm_testing.TestShootForUpdate().DeepCopy()

			shootWithOIDC.Spec.Kubernetes = gardener.Kubernetes{
				KubeAPIServer: &gardener.KubeAPIServerConfig{
					OIDCConfig: &gardener.OIDCConfig{
						ClientID:  ptr.To("some-old-client-id"),
						IssuerURL: ptr.To("some-old-url.com"),
					},
				},
			}

			fakeSystemState := &systemState{instance: runtimeWithOIDC, shoot: shootWithOIDC}

			outputFsmState := outputFnState{
				nextStep:    haveName("sFnUpdateStatus"),
				annotations: expectedAnnotations,
				result:      nil,
				status:      fsm_testing.PendingStatusShootPatched(),
			}

			newConfigMap := &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "structured-auth-config-" + fakeSystemState.instance.Spec.Shoot.Name,
					Namespace: fakeSystemState.shoot.Namespace,
				},
			}

			err := fakeFSM.ShootClient.Create(ctx, newConfigMap)
			Expect(err).To(BeNil())

			testFunc(ctx, fakeFSM, fakeSystemState, outputFsmState)
			shootAfterUpdate := &gardener.Shoot{}

			err = fakeFSM.ShootClient.Get(ctx, typeNamespacedName, shootAfterUpdate)

			Expect(err).To(BeNil())
			Expect(shootAfterUpdate.Spec.Kubernetes.KubeAPIServer.OIDCConfig).To(BeNil())

			var updatedConfigMap v1.ConfigMap

			err = fakeFSM.ShootClient.Get(ctx, types.NamespacedName{Name: "structured-auth-test-shoot", Namespace: typeNamespacedName.Namespace}, &updatedConfigMap)
			Expect(err).To(BeNil())

			authenticationConfigString := updatedConfigMap.Data["config.yaml"]
			var authenticationConfiguration structuredauth.AuthenticationConfiguration
			err = yaml.Unmarshal([]byte(authenticationConfigString), &authenticationConfiguration)

			Expect(err).To(BeNil())
			Expect(authenticationConfiguration.JWT).To(HaveLen(1))
			Expect(authenticationConfiguration.JWT[0].Issuer.URL).To(Equal("some.url.com"))
			Expect(authenticationConfiguration.JWT[0].Issuer.Audiences).To(Equal([]string{"client-id"}))
		})
	})
})

func setupFakeFSMForTest(scheme *runtime.Scheme, objs ...client.Object) *fsm {
	return must(newFakeFSM,
		withMockedMetrics(),
		withTestFinalizer,
		withShootNamespace("garden-"),
		withFakedK8sClient(scheme, objs...),
		withFakeEventRecorder(1),
		withDefaultReconcileDuration(),
	)
}

func setupFakeFSMForTestKeepGeneration(scheme *runtime.Scheme, runtime *imv1.Runtime) *fsm {
	return must(newFakeFSM,
		withMockedMetrics(),
		withShootNamespace("garden-"),
		withTestFinalizer,
		withFakedK8sClientKeepGeneration(scheme, runtime),
		withFakeEventRecorder(1),
		withDefaultReconcileDuration(),
	)
}

func setupFakeFSMForTestWithFailingPatchWithInConflictError(scheme *runtime.Scheme, runtime *imv1.Runtime) *fsm {
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

func setupFakeFSMForTestWithFailingUpdateWithInConflictError(scheme *runtime.Scheme, runtime *imv1.Runtime) *fsm {
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

func setupFakeFSMForTestWithFailingPatchWithForbiddenError(scheme *runtime.Scheme, runtime *imv1.Runtime) *fsm {
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

func setupFakeFSMForTestWithFailingUpdateWithForbiddenError(scheme *runtime.Scheme, runtime *imv1.Runtime) *fsm {
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

func setupFakeFSMForTestWithFailingPatchWithOtherError(scheme *runtime.Scheme, runtime *imv1.Runtime) *fsm {
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

func setupFakeFSMForTestWithFailingUpdateWithOtherError(scheme *runtime.Scheme, runtime *imv1.Runtime) *fsm {
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

func setupFakeFSMForTestWithAuditLogMandatory(scheme *runtime.Scheme, runtime *imv1.Runtime) *fsm {
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

func setupFakeFSMForTestWithStructuredAuthEnabled(scheme *runtime.Scheme, runtime *imv1.Runtime) *fsm {
	return must(newFakeFSM,
		withMockedMetrics(),
		withShootNamespace("garden-"),
		withTestFinalizer,
		withFakedK8sClient(scheme, runtime),
		withFakeEventRecorder(1),
		withDefaultReconcileDuration(),
		withStructuredAuthEnabled(true),
	)
}

func setupFakeFSMForTestWithAuditLogMandatoryAndConfig(scheme *runtime.Scheme, runtime *imv1.Runtime) *fsm {
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

func buildPatchTestFunction(fn stateFn) func(context.Context, *fsm, *systemState, outputFnState) {
	return func(ctx context.Context, r *fsm, s *systemState, expected outputFnState) {

		createErr := r.ShootClient.Create(ctx, s.shoot)
		if createErr != nil {
			return
		}

		sFn, res, err := fn(ctx, r, s)

		Expect(err).To(BeNil())
		Expect(res).To(Equal(expected.result))

		if s.instance.Status.Conditions != nil {
			Expect(len(s.instance.Status.Conditions)).To(Equal(len(expected.status.Conditions)))
			for i := range s.instance.Status.Conditions {
				s.instance.Status.Conditions[i].LastTransitionTime = metav1.Time{}
				expected.status.Conditions[i].LastTransitionTime = metav1.Time{}
			}
		}

		Expect(s.instance.Status).To(Equal(expected.status))
		Expect(sFn).To(expected.nextStep)
		Expect(s.instance.GetAnnotations()).To(Equal(expected.annotations))
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
