package fsm

import (
	"context"
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	imv1_client "github.com/kyma-project/infrastructure-manager/internal/controller/runtime/fsm/client"
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

		It("Should successfully swithc status to sFnUpdateStatus", func() {
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
			imv1_client.GetShootClient = func(
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