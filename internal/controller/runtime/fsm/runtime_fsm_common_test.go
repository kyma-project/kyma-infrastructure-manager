package fsm

import (
	"context"
	"time"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	. "github.com/onsi/ginkgo/v2" //nolint:revive
	. "github.com/onsi/gomega"    //nolint:revive
)

// These tests guard the fix for KIM issue #1413: updateStatusAndRequeue must
// re-enqueue with a non-zero RequeueAfter taken from fsm.StatusRequeueDelay.
// Returning Result{Requeue: true} (i.e. RequeueAfter == 0) caused the next
// reconcile to read a stale resourceVersion from the informer cache and 409
// on the next status Update.
var _ = Describe("updateStatusAndRequeue", func() {

	DescribeTable("re-enqueues with the configured StatusRequeueDelay",
		func(delay time.Duration) {
			m := &fsm{RCCfg: RCCfg{StatusRequeueDelay: delay}}
			next, immediate, err := updateStatusAndRequeue(m)

			Expect(err).ShouldNot(HaveOccurred())
			Expect(immediate).Should(BeNil(),
				"helper must not produce an immediate (zero-delay) requeue")
			Expect(next).ShouldNot(BeNil())

			// The returned stateFn is the sFnUpdateStatus closure, which carries the
			// Result captured by updateStatusAndRequeue. When Status == snapshot
			// (no status diff) the closure returns the captured Result unchanged
			// without writing to the API server, so we can read it back here.
			s := &systemState{instance: imv1.Runtime{}}
			s.saveRuntimeStatus() // snapshot == instance.Status

			_, result, err := next(context.Background(), m, s)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(result).ShouldNot(BeNil())
			Expect(result.RequeueAfter).Should(Equal(delay),
				"RequeueAfter must equal m.StatusRequeueDelay")
			Expect(result.Requeue).Should(BeFalse(),
				"helper must not set Result.Requeue (zero-delay requeue)")
		},
		Entry("1s delay", 1*time.Second),
		Entry("5s delay", 5*time.Second),
		Entry("250ms delay", 250*time.Millisecond),
	)
})
