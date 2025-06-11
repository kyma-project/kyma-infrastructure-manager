package fsm

import (
	"context"
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	fsm_testing "github.com/kyma-project/infrastructure-manager/internal/controller/runtime/fsm/testing"
	. "github.com/onsi/ginkgo/v2" //nolint:revive
	. "github.com/onsi/gomega"    //nolint:revive
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("KIM sFnCreateShoot", func() {
	inputRuntime := makeInputRuntimeWithAnnotation(map[string]string{"operator.kyma-project.io/existing-annotation": "true"})

	Context("When creating a shoot", func() {
		ctx := context.Background()

		It("Should successfully create kyma-provisioning-info configmap", func() {
			runtime := *inputRuntime.DeepCopy()
			shoot := fsm_testing.TestShootForUpdate().DeepCopy()

			scheme, schemeErr := newCreateTestScheme()
			Expect(schemeErr).To(BeNil(), "Failed to create test scheme")

			// start of fake client setup
			var fakeClient = fake.NewClientBuilder().
				WithScheme(scheme).
				Build()
			testFsm := &fsm{K8s: K8s{
				ShootClient: fakeClient,
				Client:      fakeClient,
			}}
			GetShootClient = func(
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

			// when
			stateFn, _, _ := sFnCreateShoot(ctx, testFsm, systemState)

			// then
			Expect(stateFn.name()).To(ContainSubstring("sFnUpdateStatus"))

			var detailsCM v1.ConfigMap
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

func newCreateTestScheme() (*runtime.Scheme, error) {
	schema := runtime.NewScheme()

	for _, fn := range []func(*runtime.Scheme) error{
		gardener.AddToScheme,
		v1.AddToScheme,
	} {
		if err := fn(schema); err != nil {
			return nil, err
		}
	}
	return schema, nil
}