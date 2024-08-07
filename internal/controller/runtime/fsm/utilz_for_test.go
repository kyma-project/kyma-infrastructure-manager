package fsm

import (
	"fmt"
	"github.com/kyma-project/infrastructure-manager/internal/gardener/shoot"

	. "github.com/onsi/gomega" //nolint:revive
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type fakeFSMOpt func(*fsm) error

var (
	errFailedToCreateFakeFSM = fmt.Errorf("failed to create fake FSM")

	must = func(f func(opts ...fakeFSMOpt) (*fsm, error), opts ...fakeFSMOpt) *fsm {
		fsm, err := f(opts...)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(fsm).NotTo(BeNil())
		return fsm
	}

	withFinalizer = func(finalizer string) fakeFSMOpt {
		return func(fsm *fsm) error {
			fsm.Finalizer = finalizer
			return nil
		}
	}

	withStorageWriter = func(testWriterGetter writerGetter) fakeFSMOpt {
		return func(fsm *fsm) error {
			fsm.writerProvider = testWriterGetter
			return nil
		}
	}

	withConverterConfig = func(config shoot.ConverterConfig) fakeFSMOpt {
		return func(fsm *fsm) error {
			fsm.ConverterConfig = config
			return nil
		}
	}

	withFakedK8sClient = func(
		scheme *runtime.Scheme,
		objs ...client.Object) fakeFSMOpt {

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(objs...).
			Build()

		return func(fsm *fsm) error {
			fsm.Client = k8sClient
			return nil
		}
	}

	/* linter fix for unused code
	withMockedShootClient = func(c *gardener_mocks.ShootClient) fakeFSMOpt {
		return func(fsm *fsm) error {
			fsm.ShootClient = c
			return nil
		}
	}
	*/
)

func newFakeFSM(opts ...fakeFSMOpt) (*fsm, error) {
	fsm := fsm{}
	// apply opts
	for _, opt := range opts {
		if err := opt(&fsm); err != nil {
			return nil, fmt.Errorf(
				"%w: %s",
				errFailedToCreateFakeFSM,
				err.Error(),
			)
		}
	}
	return &fsm, nil
}
