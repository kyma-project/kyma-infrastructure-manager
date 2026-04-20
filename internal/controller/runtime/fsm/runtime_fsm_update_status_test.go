package fsm

import (
	"context"
	"errors"
	"time"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	. "github.com/onsi/ginkgo/v2" //nolint:revive
	. "github.com/onsi/gomega"    //nolint:revive
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type conflictingStatusWriter struct {
	client.SubResourceWriter
	updateCalls            *int
	conflictsBeforeSuccess int
	conflictErr            error
}

func (w *conflictingStatusWriter) Update(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error {
	*w.updateCalls++
	if *w.updateCalls <= w.conflictsBeforeSuccess {
		return w.conflictErr
	}

	return w.SubResourceWriter.Update(ctx, obj, opts...)
}

type statusWriterClient struct {
	client.Client
	statusWriter client.SubResourceWriter
}

func (c *statusWriterClient) Status() client.SubResourceWriter {
	return c.statusWriter
}

var _ = Describe("KIM sFnUpdateStatus", func() {
	buildScheme := func() *runtime.Scheme {
		scheme := runtime.NewScheme()
		utilruntime.Must(imv1.AddToScheme(scheme))
		utilruntime.Must(corev1.AddToScheme(scheme))
		return scheme
	}

	baseRuntime := func() imv1.Runtime {
		return imv1.Runtime{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-runtime",
				Namespace: "kcp-system",
			},
			Status: imv1.RuntimeStatus{
				State: imv1.RuntimeStatePending,
			},
		}
	}

	withConflictOnRuntimeUpdate := func(scheme *runtime.Scheme, updateCalls *int, conflictsBeforeSuccess int, objs ...client.Object) fakeFSMOpt {
		baseClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(objs...).
			WithStatusSubresource(objs...).Build()

		k8sClient := &statusWriterClient{
			Client: baseClient,
			statusWriter: &conflictingStatusWriter{
				SubResourceWriter:      baseClient.Status(),
				updateCalls:            updateCalls,
				conflictsBeforeSuccess: conflictsBeforeSuccess,
				conflictErr: apierrors.NewConflict(
					schema.GroupResource{Group: imv1.GroupVersion.Group, Resource: "runtimes"},
					objs[0].GetName(),
					errors.New("simulated conflict"),
				),
			},
		}

		return func(fsm *fsm) error {
			fsm.KcpClient = k8sClient
			fsm.GardenClient = baseClient
			return nil
		}
	}

	It("retries conflicts and updates status with the latest resource version", func() {
		testCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		scheme := buildScheme()
		storedRuntime := baseRuntime()
		desiredRuntime := storedRuntime.DeepCopy()
		desiredRuntime.Status.State = imv1.RuntimeStateReady

		updateCalls := 0
		testFSM := must(newFakeFSM,
			withConflictOnRuntimeUpdate(scheme, &updateCalls, 1, &storedRuntime),
			withMockedMetrics(),
		)

		systemState := &systemState{
			instance: *desiredRuntime,
			snapshot: storedRuntime.Status,
		}

		next, result, err := sFnUpdateStatus(&ctrl.Result{Requeue: true}, nil)(testCtx, testFSM, systemState)
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(BeNil())
		Expect(next).NotTo(BeNil())
		Expect(updateCalls).To(Equal(2))

		var updatedRuntime imv1.Runtime
		Expect(testFSM.KcpClient.Get(testCtx, client.ObjectKeyFromObject(&storedRuntime), &updatedRuntime)).To(Succeed())
		Expect(updatedRuntime.Status.State).To(Equal(imv1.State(imv1.RuntimeStateReady)))
	})

	It("returns an error when status update conflicts persist", func() {
		testCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		scheme := buildScheme()
		storedRuntime := baseRuntime()
		desiredRuntime := storedRuntime.DeepCopy()
		desiredRuntime.Status.State = imv1.RuntimeStateReady
		updateCalls := 0
		testFSM := must(newFakeFSM,
			withConflictOnRuntimeUpdate(scheme, &updateCalls, 100, &storedRuntime),
			withMockedMetrics(),
		)

		systemState := &systemState{
			instance: *desiredRuntime,
			snapshot: storedRuntime.Status,
		}

		next, result, err := sFnUpdateStatus(&ctrl.Result{Requeue: true}, nil)(testCtx, testFSM, systemState)
		Expect(err).To(HaveOccurred())
		Expect(apierrors.IsConflict(err)).To(BeTrue())
		Expect(next).To(BeNil())
		Expect(result).To(BeNil())
		Expect(updateCalls).To(BeNumerically(">", 1))
	})
})
