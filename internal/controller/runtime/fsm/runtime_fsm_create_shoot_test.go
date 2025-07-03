package fsm

import (
	"context"
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	fsm_testing "github.com/kyma-project/infrastructure-manager/internal/controller/runtime/fsm/testing"
	. "github.com/onsi/ginkgo/v2" //nolint:revive
	. "github.com/onsi/gomega"    //nolint:revive
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("KIM sFnCreateShoot", func() {
	inputRuntime := makeInputRuntimeWithAnnotation(map[string]string{"operator.kyma-project.io/existing-annotation": "true"})

	Context("When creating a shoot", func() {
		ctx := context.Background()

		It("Should successfully switch status to sFnUpdateStatus", func() {
			runtime := *inputRuntime.DeepCopy()
			shoot := fsm_testing.TestShootForPatch().DeepCopy()

			scheme, schemeErr := newCreateTestScheme()
			Expect(schemeErr).To(BeNil(), "Failed to create test scheme")

			// start of fake client setup
			var fakeClient = fake.NewClientBuilder().
				WithScheme(scheme).
				Build()
			testFsm := &fsm{K8s: K8s{
				SeedClient: fakeClient,
				KcpClient:  fakeClient,
			}}

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
