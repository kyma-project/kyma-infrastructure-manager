package fsm

import (
	"context"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	fsm_testing "github.com/kyma-project/infrastructure-manager/internal/controller/runtime/fsm/testing"
	. "github.com/onsi/ginkgo/v2" //nolint:revive
	. "github.com/onsi/gomega"    //nolint:revive
	core_v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

var _ = Describe("KIM patch2", func() {
	inputRuntime := makeInputRuntimeWithAnnotation(map[string]string{"operator.kyma-project.io/existing-annotation": "true"})

	Context("When patching a shoot", func() {
		ctx := context.Background()

		It("Should successfully patch kyma-provisioning-info configmap2", func() {
			runtime := *inputRuntime.DeepCopy()
			shoot := fsm_testing.TestShootForPatch()

			scheme, schemeErr := newCreateTestScheme()
			Expect(schemeErr).To(BeNil(), "Failed to create test scheme")

			// start of fake client setup
			var fakeClient = fake.NewClientBuilder().
				WithScheme(scheme).
				WithInterceptorFuncs(interceptor.Funcs{
					Patch: fsm_testing.GetFakePatchInterceptorForConfigMap(true),
				}).
				Build()
			testFsm := &fsm{K8s: K8s{
				ShootClient: fakeClient,
				Client:      fakeClient,
			}}
			GetShootClientPatch = func(
				_ context.Context,
				_ client.Client,
				_ imv1.Runtime) (client.Client, error) {
				return fakeClient, nil
			}

			// end of fake client setup

			systemState := &systemState{
				instance: runtime,
				shoot:    shoot,
			}

			detailsConfigMap := &core_v1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ConfigMap",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kyma-provisioning-info",
					Namespace: "kyma-system",
				},
				Data: nil,
			}

			cmCreationErr := testFsm.Create(ctx, detailsConfigMap)
			Expect(cmCreationErr).To(BeNil(), "Failed to create kyma-provisioning-info configmap")

			// when
			stateFn, _, _ := sFnPatchExistingShoot(ctx, testFsm, systemState)

			// then
			Expect(stateFn.name()).To(ContainSubstring("sFnUpdateStatus"))

			var detailsCM core_v1.ConfigMap
			key := client.ObjectKey{
				Name: "kyma-provisioning-info",
				Namespace: "kyma-system",
			}
			err := fakeClient.Get(ctx, key, &detailsCM)
			Expect(err).To(BeNil())
			Expect(detailsCM.Data).To(HaveKey("details"))
			Expect(detailsCM.Data["details"]).To(Equal("globalAccountID: global-account-id\ninfrastructureConfig:\n  apiVersion: aws.provider.extensions.gardener.cloud/v1alpha1\n  kind: InfrastructureConfig\n  networks:\n    vpc:\n      cidr: 10.250.0.0/22\n    zones:\n    - internal: 10.250.0.192/26\n      name: europe-west1-d\n      public: 10.250.0.128/26\n      workers: 10.250.0.0/25\nsubaccountID: subaccount-id\nworkerPools:\n  kyma:\n    autoScalerMax: 1\n    autoScalerMin: 1\n    haZones: false\n    machineType: m5.xlarge\n    name: test-worker\n"))
		})
	})
})

