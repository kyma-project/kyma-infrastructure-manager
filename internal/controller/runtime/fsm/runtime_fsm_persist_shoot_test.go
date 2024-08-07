package fsm

import (
	"bytes"
	"context"
	"github.com/kyma-project/infrastructure-manager/internal/gardener/shoot"
	"io"
	"time"

	"github.com/kyma-project/infrastructure-manager/internal/controller/runtime/fsm/testing"
	. "github.com/onsi/ginkgo/v2" //nolint:revive
	. "github.com/onsi/gomega"    //nolint:revive
	"sigs.k8s.io/yaml"
)

var _ = Describe("KIM sFnPersist", func() {

	var b bytes.Buffer
	testWriterGetter := func() writerGetter {
		return func(string) (io.Writer, error) {
			return &b, nil
		}
	}()

	testCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	expectedRuntime := testing.RuntimeOnlyName.DeepCopy()
	expectedRuntime.Spec.Shoot.Provider.Type = "aws"

	expectedShoot, err := convertShoot(expectedRuntime, shoot.ConverterConfig{})
	Expect(err).To(BeNil())

	shootWrite, err := yaml.Marshal(&expectedShoot)
	Expect(err).To(BeNil())

	runtimeWrite, err := yaml.Marshal(expectedRuntime)
	Expect(err).To(BeNil())

	expectedData := append(shootWrite, runtimeWrite...)
	Expect(err).ShouldNot(HaveOccurred())

	It("should persist shoot data", func() {
		next, _, err := sFnDumpShootSpec(testCtx, must(newFakeFSM, withStorageWriter(testWriterGetter), withConverterConfig(shoot.ConverterConfig{})), &systemState{shoot: &testing.ShootNoDNS, instance: *expectedRuntime})
		Expect(err).To(BeNil())
		Expect(next).To(haveName("sFnUpdateStatus"))
		Expect(b.Bytes()).To(Equal(expectedData))
	})
})
