package shoot

import (
	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	. "github.com/onsi/ginkgo/v2" //nolint:revive
	. "github.com/onsi/gomega"    //nolint:revive
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
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
	matcher := NewMatcherForCreate(expected)
	_, err := matcher.Match(actual)
	Expect(err).To(HaveOccurred())
}

func testResults(actual, expected interface{}, expectedMatch bool) {
	matcher := NewMatcherForCreate(expected)
	actualMatch, err := matcher.Match(actual)
	Expect(err).ShouldNot(HaveOccurred(), err)
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
			"should skip missing labels",
			deepCp(empty, withLabels(map[string]string{"test": "me"})),
			deepCp(empty, withLabels(map[string]string{"test": "me", "dżułel": "wuz@here"})),
			true,
		),
		Entry(
			"should detect difference in labels",
			deepCp(empty, withLabels(map[string]string{"test": "me"})),
			deepCp(empty, withLabels(map[string]string{})),
			false,
		),
		Entry(
			"should detect differences in spec/exposureClassName #1",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				ExposureClassName: ptr.To("mage"),
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				ExposureClassName: ptr.To[string]("bard"),
			})),
			false,
		),
		Entry(
			"should detect no differences in spec/exposureClassName #1",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				ExposureClassName: ptr.To[string]("mage"),
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				ExposureClassName: ptr.To[string]("mage"),
			})),
			true,
		),
		Entry(
			"should detect differences in spec/secretBindingName #1",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				SecretBindingName: ptr.To[string]("test-1"),
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				SecretBindingName: ptr.To[string]("test-2"),
			})),
			false,
		),
		Entry(
			"should detect differences in spec/secretBindingName #2",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				SecretBindingName: ptr.To[string]("test-1"),
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{})),
			false,
		),
		Entry(
			"should detect differences in spec/secretBindingName #3",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				SecretBindingName: ptr.To[string]("test-1"),
			})),
			false,
		),
		Entry(
			"should find no differences in spec #1",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				SecretBindingName: ptr.To[string]("test-1"),
				Purpose:           ptr.To[v1beta1.ShootPurpose]("test-1"),
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				SecretBindingName: ptr.To[string]("test-1"),
				Purpose:           ptr.To[v1beta1.ShootPurpose]("test-1"),
			})),
			true,
		),
		Entry(
			"should detect differences in spec/purpose #1",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Purpose: ptr.To[v1beta1.ShootPurpose]("test-1"),
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Purpose: ptr.To[v1beta1.ShootPurpose]("test-2"),
			})),
			false,
		),
		Entry(
			"should detect differences in spec/purpose #2",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Purpose: ptr.To[v1beta1.ShootPurpose]("test-1"),
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{})),
			false,
		),
		Entry(
			"should detect differences in spec/purpose #3",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Purpose: ptr.To[v1beta1.ShootPurpose]("test-1"),
			})),
			false,
		),
		Entry(
			"should detect differences in spec #1",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Region: "test1",
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Region: "test2",
			})),
			false,
		),
		Entry(
			"should detect differences in spec #2",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Region: "test1",
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				DNS:    &v1beta1.DNS{},
				Region: "test1",
			})),
			false,
		),
		Entry(
			"should detect differences in spec/exposureClassName #1",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				ExposureClassName: ptr.To[string]("mage"),
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				ExposureClassName: ptr.To[string]("bard"),
			})),
			false,
		),
		Entry(
			"should detect no differences in spec/exposureClassName #2",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				ExposureClassName: ptr.To[string]("mage"),
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				ExposureClassName: ptr.To[string]("mage"),
			})),
			true,
		),
		Entry(
			"should detect differences in spec/dns #1 (nil check)",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				DNS: &v1beta1.DNS{},
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{})),
			false,
		),
		Entry(
			"should detect differences in spec/dns #2",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				DNS: &v1beta1.DNS{
					Domain: ptr.To[string]("test"),
				},
				Region: "test1",
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				DNS: &v1beta1.DNS{
					Domain: ptr.To[string]("test2"),
				},
				Region: "test1",
			})),
			false,
		),
		Entry(
			"should detect differences in spec/dns/providers #1",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				DNS: &v1beta1.DNS{
					Domain: ptr.To[string]("test"),
					Providers: []v1beta1.DNSProvider{
						{
							Type:       ptr.To[string]("test1"),
							Primary:    ptr.To[bool](true),
							SecretName: ptr.To[string]("test"),
						},
					},
				},
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				DNS: &v1beta1.DNS{
					Domain: ptr.To[string]("test"),
				},
			})),
			false,
		),
		Entry(
			"should detect differences in spec/dns/providers #2",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				DNS: &v1beta1.DNS{
					Providers: []v1beta1.DNSProvider{
						{
							Type:       ptr.To[string]("test1"),
							Primary:    ptr.To[bool](true),
							SecretName: ptr.To[string]("test"),
							Domains: &v1beta1.DNSIncludeExclude{
								Include: []string{"1", "2"},
							},
						},
					},
				},
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				DNS: &v1beta1.DNS{
					Providers: []v1beta1.DNSProvider{
						{
							Type:       ptr.To[string]("test1"),
							Primary:    ptr.To[bool](true),
							SecretName: ptr.To[string]("test"),
						},
					},
				},
			})),
			false,
		),
		Entry(
			"should detect differences in spec/dns/providers #3",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				DNS: &v1beta1.DNS{
					Providers: []v1beta1.DNSProvider{
						{
							Type:       ptr.To[string]("test1"),
							Primary:    ptr.To[bool](true),
							SecretName: ptr.To[string]("test"),
							Domains: &v1beta1.DNSIncludeExclude{
								Include: []string{"1", "2"},
							},
						},
					},
				},
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				DNS: &v1beta1.DNS{
					Providers: []v1beta1.DNSProvider{
						{
							Type:       ptr.To[string]("test1"),
							Primary:    ptr.To[bool](true),
							SecretName: ptr.To[string]("test"),
							Domains: &v1beta1.DNSIncludeExclude{
								Include: []string{"1"},
							},
						},
					},
				},
			})),
			false,
		),
		Entry(
			"should find no differences in spec/dns/providers",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				DNS: &v1beta1.DNS{
					Domain: ptr.To[string]("test"),
					Providers: []v1beta1.DNSProvider{
						{
							Type:       ptr.To[string]("test1"),
							Primary:    ptr.To[bool](true),
							SecretName: ptr.To[string]("test"),
							Domains: &v1beta1.DNSIncludeExclude{
								Include: []string{"1"},
								Exclude: []string{"1", "2"},
							},
							Zones: &v1beta1.DNSIncludeExclude{
								Include: []string{"11"},
								Exclude: []string{"12", "13"},
							},
						},
					},
				},
				Region: "test1",
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				DNS: &v1beta1.DNS{
					Domain: ptr.To[string]("test"),
					Providers: []v1beta1.DNSProvider{
						{
							Type:       ptr.To[string]("test1"),
							Primary:    ptr.To[bool](true),
							SecretName: ptr.To[string]("test"),
							Domains: &v1beta1.DNSIncludeExclude{
								Include: []string{"1"},
								Exclude: []string{"3", "4"},
							},
						},
					},
				},
				Region: "test1",
			})),
			true,
		),
		Entry(
			"should find differences in spec/tolerantions #1",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Tolerations: []v1beta1.Toleration{
					{
						Key:   "key",
						Value: ptr.To[string]("val"),
					},
				},
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{})),
			false,
		),
		Entry(
			"should find differences in spec/tolerantions #2",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Tolerations: []v1beta1.Toleration{
					{
						Key: "key",
					},
					{
						Key: "key2",
					},
				},
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Tolerations: []v1beta1.Toleration{
					{
						Key: "key",
					},
				},
			})),
			false,
		),
		Entry(
			"should find no differences in spec/tolerantions",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Tolerations: []v1beta1.Toleration{
					{
						Key: "key",
					},
				},
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Tolerations: []v1beta1.Toleration{
					{
						Key: "key",
					},
				},
			})),
			true,
		),
		Entry(
			"should skip missing items in spec/tolerantions",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Tolerations: []v1beta1.Toleration{
					{
						Key: "key",
					},
				},
			})),
			true,
		),
		Entry(
			"should find differences in spec/maintenance #1",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Maintenance: &v1beta1.Maintenance{},
			})),
			false,
		),
		Entry(
			"should find differences in spec/maintenance #2",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Maintenance: &v1beta1.Maintenance{
					AutoUpdate: &v1beta1.MaintenanceAutoUpdate{
						KubernetesVersion:   true,
						MachineImageVersion: ptr.To[bool](true),
					},
				},
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Maintenance: &v1beta1.Maintenance{
					AutoUpdate: &v1beta1.MaintenanceAutoUpdate{
						KubernetesVersion: true,
					},
				},
			})),
			false,
		),
		Entry(
			"should no find differences in spec/maintenance",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Maintenance: &v1beta1.Maintenance{
					AutoUpdate: &v1beta1.MaintenanceAutoUpdate{
						KubernetesVersion:   true,
						MachineImageVersion: ptr.To[bool](true),
					},
					TimeWindow: &v1beta1.MaintenanceTimeWindow{},
				},
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Maintenance: &v1beta1.Maintenance{
					AutoUpdate: &v1beta1.MaintenanceAutoUpdate{
						KubernetesVersion:   true,
						MachineImageVersion: ptr.To[bool](true),
					},
					ConfineSpecUpdateRollout: ptr.To[bool](true),
				},
			})),
			true,
		),
		Entry(
			"should find differences in spec/networking",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Networking: &v1beta1.Networking{},
			})),
			false,
		),
		Entry(
			"should find differences in spec/networking #1",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Networking: &v1beta1.Networking{},
			})),
			false,
		),
		Entry(
			"should find differences in spec/networking #2",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Networking: &v1beta1.Networking{},
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{})),
			false,
		),
		Entry(
			"should find differences in spec/networking #3",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Networking: &v1beta1.Networking{
					Type:     ptr.To[string]("r-type"),
					Nodes:    ptr.To[string]("the-nodes"),
					Services: ptr.To[string]("svcs"),
					Pods:     ptr.To[string]("pods"),
				},
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Networking: &v1beta1.Networking{
					Type:     ptr.To[string]("r-type"),
					Nodes:    ptr.To[string]("the-nodes"),
					Services: ptr.To[string]("svcs"),
					Pods:     ptr.To[string]("podzorz"),
				},
			})),
			false,
		),
		Entry(
			"should skip irrelevant differences in spec/networking",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Networking: &v1beta1.Networking{
					Type:     ptr.To[string]("r-type"),
					Nodes:    ptr.To[string]("the-nodes"),
					Services: ptr.To[string]("svcs"),
					Pods:     ptr.To[string]("pods"),
					ProviderConfig: &runtime.RawExtension{
						Raw: []byte("this is"),
					},
					IPFamilies: []v1beta1.IPFamily{
						"ipFamilyGuy",
					},
				},
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Networking: &v1beta1.Networking{
					Type:     ptr.To[string]("r-type"),
					Nodes:    ptr.To[string]("the-nodes"),
					Services: ptr.To[string]("svcs"),
					Pods:     ptr.To[string]("pods"),
					ProviderConfig: &runtime.RawExtension{
						Raw: []byte("not important"),
					},
				},
			})),
			true,
		),
		Entry(
			"should match spec/kubernetes",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Kubernetes: v1beta1.Kubernetes{
					Version: "1.2.3",
				},
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Kubernetes: v1beta1.Kubernetes{
					Version: "1.2.3",
				},
			})),
			true,
		),
		Entry(
			"should find differences in spec/kubernetes #1",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Kubernetes: v1beta1.Kubernetes{
					Version: "1.2.3",
				},
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Kubernetes: v1beta1.Kubernetes{
					Version: "1.2.1",
				},
			})),
			false,
		),
		Entry(
			"should find differences in spec/kubernetes #2",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Kubernetes: v1beta1.Kubernetes{
					Version:                     "1.2.3",
					EnableStaticTokenKubeconfig: ptr.To[bool](true),
				},
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Kubernetes: v1beta1.Kubernetes{
					Version: "1.2.3",
				},
			})),
			false,
		),
		Entry(
			"should find no differences in spec/kubernetes/kubeAPIServer #1",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Kubernetes: v1beta1.Kubernetes{
					KubeAPIServer: &v1beta1.KubeAPIServerConfig{
						OIDCConfig: &v1beta1.OIDCConfig{
							CABundle:       ptr.To[string]("test"),
							ClientID:       ptr.To[string]("test"),
							GroupsClaim:    ptr.To[string]("test"),
							SigningAlgs:    []string{"1", "2", "3"},
							GroupsPrefix:   ptr.To[string]("test"),
							IssuerURL:      ptr.To[string]("test"),
							RequiredClaims: map[string]string{"test": "me"},
							UsernameClaim:  ptr.To[string]("test"),
							UsernamePrefix: ptr.To[string]("test"),
						},
					},
				},
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Kubernetes: v1beta1.Kubernetes{
					KubeAPIServer: &v1beta1.KubeAPIServerConfig{
						OIDCConfig: &v1beta1.OIDCConfig{
							CABundle:       ptr.To[string]("test"),
							ClientID:       ptr.To[string]("test"),
							GroupsClaim:    ptr.To[string]("test"),
							SigningAlgs:    []string{"3", "1", "2"},
							GroupsPrefix:   ptr.To[string]("test"),
							IssuerURL:      ptr.To[string]("test"),
							RequiredClaims: map[string]string{"test": "me"},
							UsernameClaim:  ptr.To[string]("test"),
							UsernamePrefix: ptr.To[string]("test"),
						},
					},
				},
			})),
			true,
		),
		Entry(
			"should find differences in spec/kubernetes/kubeAPIServer #1",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Kubernetes: v1beta1.Kubernetes{
					KubeAPIServer: &v1beta1.KubeAPIServerConfig{},
				},
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Kubernetes: v1beta1.Kubernetes{
					KubeAPIServer: &v1beta1.KubeAPIServerConfig{
						OIDCConfig: &v1beta1.OIDCConfig{
							CABundle: ptr.To[string]("test"),
						},
					},
				},
			})),
			false,
		),
		Entry(
			"should find differences in spec/kubernetes/kubeAPIServer #2",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Kubernetes: v1beta1.Kubernetes{
					KubeAPIServer: &v1beta1.KubeAPIServerConfig{},
				},
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Kubernetes: v1beta1.Kubernetes{
					KubeAPIServer: &v1beta1.KubeAPIServerConfig{
						OIDCConfig: &v1beta1.OIDCConfig{
							CABundle: ptr.To[string]("test"),
						},
					},
				},
			})),
			false,
		),
		Entry(
			"should detect differences in spec/kubernetes/kubeAPIServer #3",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Kubernetes: v1beta1.Kubernetes{
					KubeAPIServer: &v1beta1.KubeAPIServerConfig{
						OIDCConfig: &v1beta1.OIDCConfig{
							GroupsPrefix: ptr.To[string]("test"),
						},
					},
				},
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Kubernetes: v1beta1.Kubernetes{
					KubeAPIServer: &v1beta1.KubeAPIServerConfig{
						OIDCConfig: &v1beta1.OIDCConfig{
							GroupsPrefix: ptr.To[string]("me"),
						},
					},
				},
			})),
			false,
		),
		Entry(
			"should detect differences in spec/kubernetes/kubeAPIServer #4",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Kubernetes: v1beta1.Kubernetes{
					KubeAPIServer: &v1beta1.KubeAPIServerConfig{
						OIDCConfig: &v1beta1.OIDCConfig{
							IssuerURL: ptr.To[string]("test"),
						},
					},
				},
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Kubernetes: v1beta1.Kubernetes{
					KubeAPIServer: &v1beta1.KubeAPIServerConfig{
						OIDCConfig: &v1beta1.OIDCConfig{
							IssuerURL: ptr.To[string]("me"),
						},
					},
				},
			})),
			false,
		),
		Entry(
			"should detect differences in spec/kubernetes/kubeAPIServer #5",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Kubernetes: v1beta1.Kubernetes{
					KubeAPIServer: &v1beta1.KubeAPIServerConfig{
						OIDCConfig: &v1beta1.OIDCConfig{
							RequiredClaims: map[string]string{"im": "not"},
						},
					},
				},
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Kubernetes: v1beta1.Kubernetes{
					KubeAPIServer: &v1beta1.KubeAPIServerConfig{
						OIDCConfig: &v1beta1.OIDCConfig{
							RequiredClaims: map[string]string{"im": "here"},
						},
					},
				},
			})),
			false,
		),
		Entry(
			"should detect differences in spec/kubernetes/kubeAPIServer #6",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Kubernetes: v1beta1.Kubernetes{
					KubeAPIServer: &v1beta1.KubeAPIServerConfig{
						OIDCConfig: &v1beta1.OIDCConfig{},
					},
				},
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Kubernetes: v1beta1.Kubernetes{
					KubeAPIServer: &v1beta1.KubeAPIServerConfig{
						OIDCConfig: &v1beta1.OIDCConfig{
							RequiredClaims: map[string]string{"im": "alone"},
						},
					},
				},
			})),
			false,
		),
		Entry(
			"should detect differences in spec/kubernetes/kubeAPIServer #7",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Kubernetes: v1beta1.Kubernetes{
					KubeAPIServer: &v1beta1.KubeAPIServerConfig{
						OIDCConfig: &v1beta1.OIDCConfig{
							RequiredClaims: map[string]string{"im": "alone"},
						},
					},
				},
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Kubernetes: v1beta1.Kubernetes{
					KubeAPIServer: &v1beta1.KubeAPIServerConfig{},
				},
			})),
			false,
		),
		Entry(
			"should detect differences in spec/kubernetes/kubeAPIServer #8",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Kubernetes: v1beta1.Kubernetes{
					KubeAPIServer: &v1beta1.KubeAPIServerConfig{
						OIDCConfig: &v1beta1.OIDCConfig{
							UsernameClaim: ptr.To[string]("test me"),
						},
					},
				},
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Kubernetes: v1beta1.Kubernetes{
					KubeAPIServer: &v1beta1.KubeAPIServerConfig{},
				},
			})),
			false,
		),
		Entry(
			"should detect differences in spec/kubernetes/kubeAPIServer #9",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Kubernetes: v1beta1.Kubernetes{
					KubeAPIServer: &v1beta1.KubeAPIServerConfig{
						OIDCConfig: &v1beta1.OIDCConfig{
							UsernameClaim: ptr.To[string]("test me"),
						},
					},
				},
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Kubernetes: v1beta1.Kubernetes{
					KubeAPIServer: &v1beta1.KubeAPIServerConfig{
						OIDCConfig: &v1beta1.OIDCConfig{
							UsernameClaim: ptr.To[string]("test me ... not"),
						},
					},
				},
			})),
			false,
		),
		Entry(
			"should detect differences in spec/kubernetes/kubeAPIServer #10",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Kubernetes: v1beta1.Kubernetes{
					KubeAPIServer: &v1beta1.KubeAPIServerConfig{
						OIDCConfig: &v1beta1.OIDCConfig{
							UsernamePrefix: ptr.To[string]("a"),
						},
					},
				},
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Kubernetes: v1beta1.Kubernetes{
					KubeAPIServer: &v1beta1.KubeAPIServerConfig{
						OIDCConfig: &v1beta1.OIDCConfig{
							UsernamePrefix: ptr.To[string]("b"),
						},
					},
				},
			})),
			false,
		),
		Entry(
			"should detect differences in spec/kubernetes/kubeAPIServer #11",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Kubernetes: v1beta1.Kubernetes{
					KubeAPIServer: &v1beta1.KubeAPIServerConfig{
						OIDCConfig: &v1beta1.OIDCConfig{
							UsernamePrefix: ptr.To[string]("a"),
						},
					},
				},
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Kubernetes: v1beta1.Kubernetes{
					KubeAPIServer: &v1beta1.KubeAPIServerConfig{
						OIDCConfig: &v1beta1.OIDCConfig{},
					},
				},
			})),
			false,
		),
		Entry(
			"should detect differences in spec/kubernetes/kubeAPIServer #12",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Kubernetes: v1beta1.Kubernetes{
					KubeAPIServer: &v1beta1.KubeAPIServerConfig{
						OIDCConfig: &v1beta1.OIDCConfig{},
					},
				},
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Kubernetes: v1beta1.Kubernetes{
					KubeAPIServer: &v1beta1.KubeAPIServerConfig{
						OIDCConfig: &v1beta1.OIDCConfig{
							UsernamePrefix: ptr.To[string]("a"),
						},
					},
				},
			})),
			false,
		),
		Entry(
			"should detect differences in spec/kubernetes/kubeAPIServer #13",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Kubernetes: v1beta1.Kubernetes{
					KubeAPIServer: &v1beta1.KubeAPIServerConfig{
						OIDCConfig: &v1beta1.OIDCConfig{},
					},
				},
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Kubernetes: v1beta1.Kubernetes{
					KubeAPIServer: &v1beta1.KubeAPIServerConfig{
						OIDCConfig: &v1beta1.OIDCConfig{
							UsernamePrefix: ptr.To[string]("a"),
						},
					},
				},
			})),
			false,
		),
		Entry(
			"should find no differences in spec/extensions #1",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Extensions: []v1beta1.Extension{
					{
						Type:     "rtype",
						Disabled: ptr.To[bool](true),
						ProviderConfig: &runtime.RawExtension{
							Raw: []byte("testme"),
						},
					},
				},
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Extensions: []v1beta1.Extension{
					{
						Type:     "rtype",
						Disabled: ptr.To[bool](true),
						ProviderConfig: &runtime.RawExtension{
							Raw: []byte("testme"),
						},
					},
				},
			})),
			true,
		),
		Entry(
			"should find no differences in spec/extensions #3",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Extensions: []v1beta1.Extension{
					{
						Type: "bug-detector",
						ProviderConfig: &runtime.RawExtension{
							Raw: []byte("b"),
						},
					},
				},
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Extensions: []v1beta1.Extension{
					{
						Type: "bug-detector",
						ProviderConfig: &runtime.RawExtension{
							Raw: []byte("b"),
						},
					},
				},
			})),
			true,
		),
		Entry(
			"should find differences in spec/extensions #1",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Extensions: []v1beta1.Extension{
					{
						Type:     "rtype",
						Disabled: ptr.To[bool](true),
						ProviderConfig: &runtime.RawExtension{
							Raw: []byte("rtype"),
						},
					},
					{
						Type:     "dtype",
						Disabled: ptr.To[bool](true),
						ProviderConfig: &runtime.RawExtension{
							Raw: []byte("btype"),
						},
					},
				},
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Extensions: []v1beta1.Extension{
					{
						Type:     "rtype",
						Disabled: ptr.To[bool](true),
						ProviderConfig: &runtime.RawExtension{
							Raw: []byte("rtype"),
						},
					},
				},
			})),
			false,
		),
		Entry(
			"should skip insignificant differences in spec/extensions #1",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Extensions: []v1beta1.Extension{
					{
						Type:     "rtype",
						Disabled: ptr.To[bool](true),
						ProviderConfig: &runtime.RawExtension{
							Raw: []byte("rtype"),
						},
					},
				},
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Extensions: []v1beta1.Extension{
					{
						Type:     "rtype",
						Disabled: ptr.To[bool](true),
						ProviderConfig: &runtime.RawExtension{
							Raw: []byte("rtype"),
						},
					},
					{
						Type:     "dtype",
						Disabled: ptr.To[bool](true),
						ProviderConfig: &runtime.RawExtension{
							Raw: []byte("btype"),
						},
					},
				},
			})),
			true,
		),
		Entry(
			"should find no differences in spec/provider #1",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Provider: v1beta1.Provider{},
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Provider: v1beta1.Provider{},
			})),
			true,
		),
		Entry(
			"should find no differences in spec/provider/infrastructureConfig #1",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Provider: v1beta1.Provider{
					InfrastructureConfig: &runtime.RawExtension{
						Raw: []byte("raw stuff goes here"),
					},
				},
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Provider: v1beta1.Provider{
					InfrastructureConfig: &runtime.RawExtension{
						Raw: []byte("raw stuff goes here"),
					},
				},
			})),
			true,
		),
		Entry(
			"should find differences in spec/provider/infrastructureConfig #1",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Provider: v1beta1.Provider{
					InfrastructureConfig: &runtime.RawExtension{
						Raw: []byte("raw stuff goes here 1"),
					},
				},
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Provider: v1beta1.Provider{
					InfrastructureConfig: &runtime.RawExtension{
						Raw: []byte("raw stuff goes here 2"),
					},
				},
			})),
			false,
		),
		Entry(
			"should find no differences in spec/provider/controlPlaneConfig #1",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Provider: v1beta1.Provider{
					ControlPlaneConfig: &runtime.RawExtension{
						Raw: []byte("raw stuff goes here"),
					},
				},
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Provider: v1beta1.Provider{
					ControlPlaneConfig: &runtime.RawExtension{
						Raw: []byte("raw stuff goes here"),
					},
				},
			})),
			true,
		),
		Entry(
			"should find differences in spec/provider/controlPlaneConfig #1",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Provider: v1beta1.Provider{
					ControlPlaneConfig: &runtime.RawExtension{
						Raw: []byte("raw stuff goes here 1"),
					},
				},
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Provider: v1beta1.Provider{
					ControlPlaneConfig: &runtime.RawExtension{
						Raw: []byte("raw stuff goes here 2"),
					},
				},
			})),
			false,
		),
		Entry(
			"should find no differences in spec/provider/workerSettings #1",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Provider: v1beta1.Provider{
					WorkersSettings: &v1beta1.WorkersSettings{
						SSHAccess: &v1beta1.SSHAccess{
							Enabled: true,
						},
					},
				},
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Provider: v1beta1.Provider{
					WorkersSettings: &v1beta1.WorkersSettings{
						SSHAccess: &v1beta1.SSHAccess{
							Enabled: true,
						},
					},
				},
			})),
			true,
		),
		Entry(
			"should find differences in spec/provider/workerSettings #1",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Provider: v1beta1.Provider{
					WorkersSettings: &v1beta1.WorkersSettings{
						SSHAccess: &v1beta1.SSHAccess{
							Enabled: false,
						},
					},
				},
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Provider: v1beta1.Provider{
					WorkersSettings: &v1beta1.WorkersSettings{
						SSHAccess: &v1beta1.SSHAccess{
							Enabled: true,
						},
					},
				},
			})),
			false,
		),
		Entry(
			"should find differences in spec/provider/workerSettings #2",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Provider: v1beta1.Provider{},
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Provider: v1beta1.Provider{
					WorkersSettings: &v1beta1.WorkersSettings{
						SSHAccess: &v1beta1.SSHAccess{
							Enabled: true,
						},
					},
				},
			})),
			false,
		),
		Entry(
			"should find differences in spec/provider/workerSettings #3",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Provider: v1beta1.Provider{
					WorkersSettings: &v1beta1.WorkersSettings{
						SSHAccess: &v1beta1.SSHAccess{
							Enabled: true,
						},
					},
				},
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Provider: v1beta1.Provider{},
			})),
			false,
		),
		Entry(
			"should find differences in spec/provider/workerSettings #2",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Provider: v1beta1.Provider{
					WorkersSettings: &v1beta1.WorkersSettings{},
				},
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Provider: v1beta1.Provider{
					WorkersSettings: &v1beta1.WorkersSettings{
						SSHAccess: &v1beta1.SSHAccess{
							Enabled: true,
						},
					},
				},
			})),
			false,
		),
		Entry(
			"should find no differences in spec/provider/workerSettings #1",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Provider: v1beta1.Provider{
					WorkersSettings: &v1beta1.WorkersSettings{
						SSHAccess: &v1beta1.SSHAccess{
							Enabled: true,
						},
					},
				},
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Provider: v1beta1.Provider{
					WorkersSettings: &v1beta1.WorkersSettings{
						SSHAccess: &v1beta1.SSHAccess{
							Enabled: true,
						},
					},
				},
			})),
			true,
		),
		Entry(

			"should find differences in spec/provider/type",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Provider: v1beta1.Provider{
					Type: "rtype",
				},
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{})),
			false,
		),
		Entry(
			"should find no differences in spec/provider/type",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Provider: v1beta1.Provider{
					Type: "rtype",
				},
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Provider: v1beta1.Provider{
					Type: "rtype",
				},
			})),
			true,
		),
		Entry(
			"should find differences in spec/provider/workers #1",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Provider: v1beta1.Provider{
					Workers: []v1beta1.Worker{
						{
							Name: "iTwurkz",
							ProviderConfig: &runtime.RawExtension{
								Raw: []byte("raw stuff here 1"),
							},
						},
					},
				},
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Provider: v1beta1.Provider{
					Workers: []v1beta1.Worker{
						{
							Name: "iTwurkz",
							ProviderConfig: &runtime.RawExtension{
								Raw: []byte("raw stuff here 2"),
							},
						},
					},
				},
			})),
			false,
		),
		Entry(
			"should find differences in spec/provider/workers #2",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Provider: v1beta1.Provider{
					Workers: []v1beta1.Worker{
						{
							Name: "iTwurkz",
							Machine: v1beta1.Machine{
								Type: "rtype",
							},
							ProviderConfig: &runtime.RawExtension{
								Raw: []byte("raw stuff here 1"),
							},
						},
					},
				},
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Provider: v1beta1.Provider{
					Workers: []v1beta1.Worker{
						{
							Name: "iTwurkz",
							Machine: v1beta1.Machine{
								Type: "rtype",
								Image: &v1beta1.ShootMachineImage{
									Name:    "image1",
									Version: ptr.To[string]("123"),
								},
							},
							ProviderConfig: &runtime.RawExtension{
								Raw: []byte("raw stuff here 1"),
							},
						},
					},
				},
			})),
			false,
		),
		Entry(
			"should find differences in spec/provider/workers #3",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Provider: v1beta1.Provider{
					Workers: []v1beta1.Worker{
						{
							Name: "iTwurkz",
						},
						{
							Name: "iTwurkz2",
						},
					},
				},
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Provider: v1beta1.Provider{
					Workers: []v1beta1.Worker{
						{
							Name: "iTwurkz",
						},
					},
				},
			})),
			false,
		),
		Entry(
			"should find no differences in spec/provider/workers #1",
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Provider: v1beta1.Provider{
					Workers: []v1beta1.Worker{
						{
							Name: "iTwurkz",
							Machine: v1beta1.Machine{
								Type: "rtype",
								Image: &v1beta1.ShootMachineImage{
									Name:    "image1",
									Version: ptr.To[string]("123"),
								},
							},
							Labels: map[string]string{
								"test": "me",
							},
							ProviderConfig: &runtime.RawExtension{
								Raw: []byte("raw stuff here 1"),
							},
							Sysctls: map[string]string{
								"test": "me",
							},
							Maximum:  5,
							Minimum:  -5,
							MaxSurge: ptr.To[intstr.IntOrString](intstr.FromInt(10)),
							Volume: &v1beta1.Volume{
								VolumeSize: "big",
								Type:       ptr.To[string]("boi"),
							},
						},
					},
				},
			})),
			deepCp(empty, withShootSpec(v1beta1.ShootSpec{
				Provider: v1beta1.Provider{
					Workers: []v1beta1.Worker{
						{
							Name: "iTwurkz",
							Machine: v1beta1.Machine{
								Type: "rtype",
								Image: &v1beta1.ShootMachineImage{
									Name:    "image1",
									Version: ptr.To[string]("123"),
								},
							},
							Annotations: map[string]string{
								"test": "me",
							},
							CABundle: ptr.To[string]("testme"),
							ProviderConfig: &runtime.RawExtension{
								Raw: []byte("raw stuff here 1"),
							},
							Sysctls: map[string]string{
								"test": "me",
							},
							Maximum:  5,
							Minimum:  -5,
							MaxSurge: ptr.To[intstr.IntOrString](intstr.FromInt(10)),
							Volume: &v1beta1.Volume{
								VolumeSize: "big",
								Type:       ptr.To[string]("boi"),
							},
						},
						{
							Name: "iTwurkz2",
							Machine: v1beta1.Machine{
								Type: "rtype",
								Image: &v1beta1.ShootMachineImage{
									Name:    "image2",
									Version: ptr.To[string]("456"),
								},
							},
							ProviderConfig: &runtime.RawExtension{
								Raw: []byte("raw stuff here 2"),
							},
							Maximum:  10,
							Minimum:  -10,
							MaxSurge: ptr.To[intstr.IntOrString](intstr.FromString("100")),
						},
					},
				},
			})),
			true,
		),
	)
})
