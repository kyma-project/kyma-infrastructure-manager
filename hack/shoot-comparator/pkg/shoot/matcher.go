package shoot

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/kyma-project/infrastructure-manager/hack/shoot-comparator/pkg/runtime"

	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/kyma-project/infrastructure-manager/hack/shoot-comparator/pkg/errors"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	"github.com/onsi/gomega/types"
	"sigs.k8s.io/yaml"
)

type Matcher struct {
	toMatch    interface{}
	fails      []string
	DNSMatcher types.GomegaMatcher
}

func NewMatcherForCreate(i interface{}) types.GomegaMatcher {
	return &Matcher{
		toMatch: i,
	}
}

func NewMatcherForPatch(i interface{}) types.GomegaMatcher {
	return &Matcher{
		toMatch:    i,
		DNSMatcher: gstruct.Ignore(),
	}
}

func getShoot(i interface{}) (shoot v1beta1.Shoot, err error) {
	if i == nil {
		return v1beta1.Shoot{}, fmt.Errorf("invalid value nil")
	}

	switch v := i.(type) {
	case string:
		err = yaml.Unmarshal([]byte(v), &shoot)
		return shoot, err

	case v1beta1.Shoot:
		return v, nil

	case *v1beta1.Shoot:
		return *v, nil

	default:
		return v1beta1.Shoot{}, fmt.Errorf(`%w: %s`, errors.ErrInvalidType, reflect.TypeOf(v))
	}
}

func (m *Matcher) Match(actual interface{}) (success bool, err error) {
	shootActual, err := getShoot(actual)
	if err != nil {
		return false, err
	}

	shootToMatch, err := getShoot(m.toMatch)
	if err != nil {
		return false, err
	}

	if m.DNSMatcher == nil {
		m.DNSMatcher = newDNSMatcher(shootToMatch.Spec.DNS)
	}

	matchers := []propertyMatcher{
		{
			GomegaMatcher: gomega.BeComparableTo(shootToMatch.Name),
			actual:        shootActual.Name,
			path:          "metadata/name",
		},
		{
			GomegaMatcher: gomega.BeComparableTo(shootToMatch.Namespace),
			actual:        shootActual.Namespace,
			path:          "metadata/namespace",
		},
		{
			GomegaMatcher: gstruct.MatchElements(idExtension, gstruct.IgnoreMissing, extensions(shootToMatch.Spec.Extensions)),
			actual:        shootActual.Spec.Extensions,
			path:          "spec/extensions",
		},
		{
			GomegaMatcher: gomega.BeComparableTo(shootToMatch.Spec.CloudProfileName),
			actual:        shootActual.Spec.CloudProfileName,
			path:          "spec/cloudProfileName",
		},
		{
			GomegaMatcher: gstruct.MatchFields(gstruct.IgnoreMissing, gstruct.Fields{
				"ClusterAutoscaler":           gstruct.Ignore(),
				"KubeAPIServer":               newKubeAPIServerMatcher(shootToMatch.Spec.Kubernetes),
				"KubeControllerManager":       gstruct.Ignore(),
				"KubeScheduler":               gstruct.Ignore(),
				"KubeProxy":                   gstruct.Ignore(),
				"Kubelet":                     gstruct.Ignore(),
				"Version":                     gomega.BeComparableTo(shootToMatch.Spec.Kubernetes.Version),
				"VerticalPodAutoscaler":       gstruct.Ignore(),
				"EnableStaticTokenKubeconfig": gomega.BeComparableTo(shootToMatch.Spec.Kubernetes.EnableStaticTokenKubeconfig),
			}),
			actual: shootActual.Spec.Kubernetes,
			path:   "spec/kubernetes",
		},
		{
			GomegaMatcher: newNetworkingMatcher(shootToMatch.Spec),
			actual:        shootActual.Spec.Networking,
			path:          "spec/networking",
		},
		{
			GomegaMatcher: newMaintenanceMatcher(shootToMatch.Spec),
			actual:        shootActual.Spec.Maintenance,
			path:          "spec/maintenance",
		},
		{
			GomegaMatcher: gomega.BeComparableTo(shootToMatch.Spec.Purpose),
			actual:        shootActual.Spec.Purpose,
			path:          "spec/purpose",
		},
		{
			GomegaMatcher: gomega.BeComparableTo(shootToMatch.Spec.Region),
			actual:        shootActual.Spec.Region,
			path:          "spec/region",
		},
		{
			GomegaMatcher: gomega.BeComparableTo(shootToMatch.Spec.SecretBindingName),
			actual:        shootActual.Spec.SecretBindingName,
			path:          "spec/secretBindingName",
		},
		{
			GomegaMatcher: m.DNSMatcher,
			path:          "spec/dns",
			actual:        shootActual.Spec.DNS,
		},
		{
			GomegaMatcher: gstruct.MatchElements(
				idToleration,
				gstruct.IgnoreMissing,
				tolerations(shootToMatch.Spec.Tolerations),
			),
			actual: shootActual.Spec.Tolerations,
			path:   "spec/tolerations",
		},
		{
			GomegaMatcher: gomega.BeComparableTo(shootToMatch.Spec.ExposureClassName),
			actual:        shootActual.Spec.ExposureClassName,
			path:          "spec/exposureClassName",
		},
		{
			GomegaMatcher: gomega.BeComparableTo(shootToMatch.Spec.ControlPlane),
			actual:        shootActual.Spec.ControlPlane,
			path:          "spec/controlPlane",
		},
		{
			GomegaMatcher: NewProviderMatcher(shootToMatch.Spec.Provider, "spec/provider"),
			actual:        shootActual.Spec.Provider,
			path:          "spec/provider",
		},
		{
			// We need to verify that labels from actual object are subset of labels from toMatch object
			GomegaMatcher: gomega.SatisfyAll(mapMatchers(shootActual.Labels)...),
			actual:        shootToMatch.Labels,
			path:          "metadata/labels",
		},
		{
			GomegaMatcher: gstruct.MatchElements(
				idResource,
				gstruct.IgnoreMissing,
				resources(shootToMatch.Spec.Resources),
			),
			actual: shootActual.Spec.Resources,
			path:   "spec/resources",
		},
	}

	for _, matcher := range matchers {
		ok, err := matcher.Match(matcher.actual)
		if err != nil {
			return false, err
		}

		if !ok {
			msg := matcher.FailureMessage(matcher.actual)
			if matcher.path != "" {
				msg = fmt.Sprintf("%s: %s", matcher.path, msg)
			}
			m.fails = append(m.fails, msg)
		}
	}

	return len(m.fails) == 0, nil
}

func (m *Matcher) NegatedFailureMessage(_ interface{}) string {
	return "expected should not equal actual"
}

func (m *Matcher) FailureMessage(_ interface{}) string {
	return strings.Join(m.fails, "\n")
}

func mapMatchers(m map[string]string) []types.GomegaMatcher {
	mLen := len(m)
	if mLen == 0 {
		return []types.GomegaMatcher{
			gomega.BeEmpty(),
		}
	}

	out := make([]types.GomegaMatcher, mLen)
	index := 0
	for key, val := range m {
		matcher := gomega.HaveKeyWithValue(key, val)
		out[index] = matcher
		index++
	}
	return out
}

func val(v interface{}) string {
	if reflect.ValueOf(v).Kind() == reflect.Pointer && reflect.ValueOf(v).IsNil() {
		return ""
	}

	if reflect.ValueOf(v).Kind() == reflect.Pointer {
		return fmt.Sprintf("%v", reflect.ValueOf(v).Elem())
	}

	return fmt.Sprintf("%v", v)
}

func idToleration(v interface{}) string {
	toleration, ok := v.(v1beta1.Toleration)
	if !ok {
		panic("invalid type")
	}
	return fmt.Sprintf("%s:%s", toleration.Key, val(toleration.Value))
}

func idResource(v interface{}) string {
	res, ok := v.(v1beta1.NamedResourceReference)
	if !ok {
		panic("invalid type")
	}
	return fmt.Sprintf("%s", res.Name)
}

func tolerations(ts []v1beta1.Toleration) gstruct.Elements {
	out := map[string]types.GomegaMatcher{}
	for _, t := range ts {
		ID := idToleration(t)
		out[ID] = gstruct.MatchAllFields(gstruct.Fields{
			"Key":   gomega.BeComparableTo(t.Key),
			"Value": gomega.BeComparableTo(t.Value),
		})
	}
	return out
}

func resources(ts []v1beta1.NamedResourceReference) gstruct.Elements {
	out := map[string]types.GomegaMatcher{}
	for _, t := range ts {
		ID := idResource(t)
		out[ID] = gstruct.MatchAllFields(gstruct.Fields{
			"Name":        gomega.BeComparableTo(t.Name),
			"ResourceRef": gomega.BeComparableTo(t.ResourceRef),
		})
	}
	return out
}

func idProvider(v interface{}) string {
	provider, ok := v.(v1beta1.DNSProvider)
	if !ok {
		panic("invalid type")
	}

	return fmt.Sprintf("%s:%s:%s",
		val(provider.Type),
		val(provider.SecretName),
		val(provider.Primary))
}

func providers(ps []v1beta1.DNSProvider) gstruct.Elements {
	out := map[string]types.GomegaMatcher{}
	for _, p := range ps {
		ID := idProvider(p)
		domainsMatcher := gomega.BeNil()
		if p.Domains != nil {
			domainsMatcher = gstruct.PointTo(
				gstruct.MatchFields(
					gstruct.IgnoreMissing,
					gstruct.Fields{
						"Include": gomega.BeComparableTo(p.Domains.Include),
						"Exclude": gstruct.Ignore(),
					}))
		}

		out[ID] = gstruct.MatchFields(
			gstruct.IgnoreMissing,
			gstruct.Fields{
				"Primary":    gomega.BeComparableTo(p.Primary),
				"SecretName": gomega.Equal(p.SecretName),
				"Type":       gomega.Equal(p.Type),
				"Domains":    domainsMatcher,
				"Zones":      gstruct.Ignore(),
			})
	}

	return out
}

func newDNSMatcher(dns *v1beta1.DNS) types.GomegaMatcher {
	if dns == nil {
		return gomega.BeNil()
	}

	return gstruct.PointTo(gstruct.MatchFields(
		gstruct.IgnoreMissing,
		gstruct.Fields{
			"Domain": gomega.BeComparableTo(dns.Domain),
			"Providers": gstruct.MatchElements(
				idProvider,
				gstruct.IgnoreMissing,
				providers(dns.Providers)),
		}))
}

func newMaintenanceMatcher(spec v1beta1.ShootSpec) types.GomegaMatcher {
	if spec.Maintenance == nil {
		return gomega.BeNil()
	}

	return gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
		"AutoUpdate": gomega.BeComparableTo(spec.Maintenance.AutoUpdate),
	}))
}

func newNetworkingMatcher(spec v1beta1.ShootSpec) types.GomegaMatcher {
	if spec.Networking == nil {
		return gomega.BeNil()
	}

	return gstruct.PointTo(
		gstruct.MatchFields(
			gstruct.IgnoreMissing,
			gstruct.Fields{
				"Type":           gomega.BeComparableTo(spec.Networking.Type),
				"Nodes":          gomega.BeComparableTo(spec.Networking.Nodes),
				"Pods":           gomega.BeComparableTo(spec.Networking.Pods),
				"Services":       gomega.BeComparableTo(spec.Networking.Services),
				"ProviderConfig": gstruct.Ignore(),
				"IPFamilies":     gstruct.Ignore(),
			}))
}

func newKubeAPIServerMatcher(k v1beta1.Kubernetes) types.GomegaMatcher {
	if k.KubeAPIServer == nil {
		return gomega.BeNil()
	}

	return gstruct.PointTo(gstruct.MatchFields(
		gstruct.IgnoreMissing,
		gstruct.Fields{
			"OIDCConfig":                          newOIDCConfigMatcher(k.KubeAPIServer),
			"KubernetesConfig":                    gstruct.Ignore(),
			"AdmissionPlugins":                    gstruct.Ignore(),
			"APIAudiences":                        gstruct.Ignore(),
			"AuditConfig":                         gomega.BeComparableTo(k.KubeAPIServer.AuditConfig),
			"RuntimeConfig":                       gstruct.Ignore(),
			"ServiceAccountConfig":                gstruct.Ignore(),
			"WatchCacheSizes":                     gstruct.Ignore(),
			"Requests":                            gstruct.Ignore(),
			"EnableAnonymousAuthentication":       gstruct.Ignore(),
			"EventTTL":                            gstruct.Ignore(),
			"Logging":                             gstruct.Ignore(),
			"DefaultNotReadyTolerationSeconds":    gstruct.Ignore(),
			"DefaultUnreachableTolerationSeconds": gstruct.Ignore(),
			"EncryptionConfig":                    gstruct.Ignore(),
			"StructuredAuthentication":            gstruct.Ignore(),
			"StructuredAuthorization":             gstruct.Ignore(),
		},
	))
}

func newOIDCConfigMatcher(c *v1beta1.KubeAPIServerConfig) types.GomegaMatcher {
	if c == nil || c.OIDCConfig == nil {
		return gomega.BeNil()
	}

	return gstruct.PointTo(gstruct.MatchFields(
		gstruct.IgnoreMissing,
		gstruct.Fields{
			"CABundle":             gomega.BeComparableTo(c.OIDCConfig.CABundle),
			"ClientID":             gomega.BeComparableTo(c.OIDCConfig.ClientID),
			"GroupsClaim":          gomega.BeComparableTo(c.OIDCConfig.GroupsClaim),
			"GroupsPrefix":         gomega.BeComparableTo(c.OIDCConfig.GroupsPrefix),
			"IssuerURL":            gomega.BeComparableTo(c.OIDCConfig.IssuerURL),
			"RequiredClaims":       gomega.BeComparableTo(c.OIDCConfig.RequiredClaims),
			"SigningAlgs":          gomega.ContainElements(c.OIDCConfig.SigningAlgs),
			"UsernameClaim":        gomega.BeComparableTo(c.OIDCConfig.UsernameClaim),
			"UsernamePrefix":       gomega.BeComparableTo(c.OIDCConfig.UsernamePrefix),
			"ClientAuthentication": gstruct.Ignore(),
		},
	))
}

func idExtension(v interface{}) string {
	e, ok := v.(v1beta1.Extension)
	if !ok {
		panic("invalid type")
	}

	return e.Type
}

func extensions(es []v1beta1.Extension) gstruct.Elements {
	out := map[string]types.GomegaMatcher{}
	for _, e := range es {
		ID := idExtension(e)
		out[ID] = gstruct.MatchAllFields(gstruct.Fields{
			"Type":           gomega.BeComparableTo(e.Type),
			"ProviderConfig": runtime.NewRawExtensionMatcher(e.ProviderConfig),
			"Disabled":       gomega.BeComparableTo(e.Disabled),
		})
	}
	return out
}
