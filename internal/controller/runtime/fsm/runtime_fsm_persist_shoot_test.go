package fsm

import (
	"bytes"
	"context"
	"io"
	"time"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/kyma-project/infrastructure-manager/internal/controller/metrics/mocks"
	"github.com/kyma-project/infrastructure-manager/internal/controller/runtime/fsm/testing"
	"github.com/kyma-project/infrastructure-manager/pkg/config"
	. "github.com/onsi/ginkgo/v2" //nolint:revive
	. "github.com/onsi/gomega"    //nolint:revive
	"github.com/stretchr/testify/mock"
	"sigs.k8s.io/yaml"
)

var _ = Describe("KIM sFnPersist", func() {

	var b bytes.Buffer
	testWriterGetter := func() writerGetter {
		return func(string) (io.Writer, error) {
			return &b, nil
		}
	}()

	withMockedMetrics := func() fakeFSMOpt {
		m := &mocks.Metrics{}
		m.On("SetRuntimeStates", mock.Anything).Return()
		m.On("CleanUpRuntimeGauge", mock.Anything, mock.Anything).Return()
		m.On("IncRuntimeFSMStopCounter").Return()
		return withMetrics(m)
	}

	testCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	expectedRuntime := testing.RuntimeOnlyName.DeepCopy()
	expectedRuntime.Spec.Shoot.Provider.Type = "aws"

	It("should persist shoot data", func() {
		next, _, err := sFnDumpShootSpec(testCtx,
			must(newFakeFSM, withStorageWriter(testWriterGetter), withConverterConfig(config.ConverterConfig{}), withMockedMetrics(), withDefaultReconcileDuration()),
			&systemState{shoot: &testing.ShootNoDNS, instance: *expectedRuntime},
		)
		Expect(err).To(BeNil())
		Expect(next).To(haveName("sFnUpdateStatus"))

		var shootStored gardener.Shoot

		err = yaml.Unmarshal(b.Bytes(), &shootStored)
		Expect(err).To(BeNil())
		Expect(shootStored.ObjectMeta.CreationTimestamp).To(Not(Equal(time.Time{})))
	})
})
