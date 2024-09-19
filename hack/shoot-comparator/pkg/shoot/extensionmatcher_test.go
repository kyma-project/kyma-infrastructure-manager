package shoot

import (
	"fmt"

	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	. "github.com/onsi/ginkgo/v2" //nolint:revive
	. "github.com/onsi/gomega"    //nolint:revive
	"k8s.io/apimachinery/pkg/runtime"
)

var _ = Describe(":: extension matcher :: ", func() {
	emptyExt := make([]v1beta1.Extension, 0)

	DescribeTable(
		"checking invalid args :: ",
		testExtensionInvalidArgs,
		Entry("when actual is not of Extension type", v1beta1.Shoot{}, emptyExt),
		Entry("when expected is not of Extension type", emptyExt, v1beta1.Shoot{}),
		Entry("when actual is nil", nil, emptyExt),
		Entry("when expected is nil", emptyExt, nil),
	)

	dnsExtEnabled := getDNSExtension(false, true)
	dnsExtDisabled := getDNSExtension(true, false)
	networkingExtDisabled := getNetworkingExtension(true)
	certificateExtEnabled := getCertificateExtension(false, true)

	DescribeTable(
		"checking results :: ",
		testExtensionResults,
		Entry(
			"should match empty values",
			emptyExt,
			emptyExt,
			true,
		),
		Entry(
			"should match identical tables",
			[]v1beta1.Extension{dnsExtEnabled, networkingExtDisabled, certificateExtEnabled},
			[]v1beta1.Extension{networkingExtDisabled, certificateExtEnabled, dnsExtEnabled},
			true,
		),
		Entry(
			"should detect extension differs",
			[]v1beta1.Extension{dnsExtEnabled, networkingExtDisabled, certificateExtEnabled},
			[]v1beta1.Extension{dnsExtDisabled, networkingExtDisabled, certificateExtEnabled},
			false,
		),
		Entry(
			"should detect redundant extension",
			[]v1beta1.Extension{dnsExtEnabled, networkingExtDisabled, certificateExtEnabled},
			[]v1beta1.Extension{networkingExtDisabled, certificateExtEnabled},
			false,
		),
		Entry(
			"should detect missing extension",
			[]v1beta1.Extension{networkingExtDisabled, certificateExtEnabled},
			[]v1beta1.Extension{dnsExtEnabled, networkingExtDisabled, certificateExtEnabled},
			false,
		),
	)
})

func testExtensionInvalidArgs(actual, expected interface{}) {
	matcher := NewExtensionMatcher(expected)
	_, err := matcher.Match(actual)
	Expect(err).To(HaveOccurred())
}

func testExtensionResults(actual, expected interface{}, expectedMatch bool) {
	matcher := NewExtensionMatcher(expected)
	actualMatch, err := matcher.Match(actual)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(actualMatch).Should(Equal(expectedMatch), matcher.FailureMessage(actual))
}

func getDNSExtension(disabled bool, replicationEnabled bool) v1beta1.Extension {
	json := fmt.Sprintf(`{apiVersion: "service.dns.extensions.gardener.cloud/v1alpha1", kind: "DNSConfig", dnsProviderReplication: {enabled: %v}}`, replicationEnabled)

	return v1beta1.Extension{
		Type:     "shoot-dns-service",
		Disabled: &disabled,
		ProviderConfig: &runtime.RawExtension{
			Raw: []byte(json),
		},
	}
}

func getNetworkingExtension(disabled bool) v1beta1.Extension {
	return v1beta1.Extension{
		Type:     "shoot-networking-service",
		Disabled: &disabled,
	}
}

func getCertificateExtension(disabled bool, shootIssuersEnabled bool) v1beta1.Extension {
	json := fmt.Sprintf(`{apiVersion: "service.cert.extensions.gardener.cloud/v1alpha1", kind: "CertConfig", shootIssuers: {enabled: %v}}`, shootIssuersEnabled)

	return v1beta1.Extension{
		Type:     "shoot-cert-service",
		Disabled: &disabled,
		ProviderConfig: &runtime.RawExtension{
			Raw: []byte(json),
		},
	}
}
