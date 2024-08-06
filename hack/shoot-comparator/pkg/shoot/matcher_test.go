package shoot

import (
	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	. "github.com/onsi/ginkgo/v2" //nolint:revive
	. "github.com/onsi/gomega"    //nolint:revive
)

type deepCpOpts = func(*v1beta1.Shoot)

func withName(name string) deepCpOpts {
	return func(s *v1beta1.Shoot) {
		s.Name = name
	}
}

func withNamespace(namespace string) deepCpOpts {
	return func(s *v1beta1.Shoot) {
		s.Namespace = namespace
	}
}

func withLabels(labels map[string]string) deepCpOpts {
	return func(s *v1beta1.Shoot) {
		s.Labels = labels
	}
}

func withShootSpec(spec v1beta1.ShootSpec) deepCpOpts {
	return func(s *v1beta1.Shoot) {
		s.Spec = spec
	}
}

// nolint: unparam
func deepCp(s v1beta1.Shoot, opts ...deepCpOpts) v1beta1.Shoot {
	for _, opt := range opts {
		opt(&s)
	}

	return s
}

func testInvalidArgs(actual, expected interface{}) {
	matcher := NewMatcher(expected)
	_, err := matcher.Match(actual)
	Expect(err).To(HaveOccurred())
}

func testResults(actual, expected interface{}, expectedMatch bool) {
	matcher := NewMatcher(expected)
	actualMatch, err := matcher.Match(actual)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(actualMatch).Should(Equal(expectedMatch), matcher.FailureMessage(actual))
}

var _ = Describe(":: shoot matcher :: ", func() {
	var empty v1beta1.Shoot

	DescribeTable(
		"checking invalid args :: ",
		testInvalidArgs,
		Entry("when actual is nil", nil, empty),
		Entry("when expected is nil", "", nil),
	)

	DescribeTable(
		"checking results :: ",
		testResults,
		Entry(
			"should match empty and zero values",
			"",
			empty,
			true,
		),
		Entry(
			"should match copies of the same instance",
			deepCp(empty),
			deepCp(empty),
			true,
		),
		Entry(
			"should detect name difference",
			deepCp(empty, withName("test1")),
			deepCp(empty, withName("test2")),
			false,
		),
		Entry(
			"should detect namespace difference",
			deepCp(empty, withNamespace("test1")),
			deepCp(empty, withNamespace("test2")),
			false,
		),
		Entry(
			"should detect difference in labels",
			deepCp(empty, withLabels(map[string]string{"test": "me"})),
			deepCp(empty, withLabels(map[string]string{})),
			false,
		),
		Entry(
			"should detect differences in spec",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Region: "test1",
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Region: "test2",
			})),
			false,
		),
	)
})
