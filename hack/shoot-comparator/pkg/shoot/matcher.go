package shoot

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"sigs.k8s.io/yaml"
)

var (
	errInvalidType = fmt.Errorf("invalid type")
)

type Matcher struct {
	toMatch interface{}
	fails   []string
}

func NewMatcher(i interface{}) types.GomegaMatcher {
	return &Matcher{
		toMatch: i,
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
		return v1beta1.Shoot{}, fmt.Errorf(`%w: %s`, errInvalidType, reflect.TypeOf(v))
	}
}

type matcher struct {
	types.GomegaMatcher
	path     string
	expected interface{}
}

func (m *Matcher) Match(actual interface{}) (success bool, err error) {
	aShoot, err := getShoot(actual)
	if err != nil {
		return false, err
	}

	eShoot, err := getShoot(m.toMatch)
	if err != nil {
		return false, err
	}

	// Note: we define separate matchers for each field to make input more readable
	// Annotations are not matched as they are not relevant for the comparison ; both KIM, and Provisioner have different set of annotations
	for _, matcher := range []matcher{
		// We need to skip comparing type meta as Provisioner doesn't set it.
		// It is simpler to skip it than to make fix in the Provisioner.
		//{
		//	GomegaMatcher: gomega.BeComparableTo(eShoot.TypeMeta),
		//	expected:      aShoot.TypeMeta,
		//},
		{
			GomegaMatcher: gomega.BeComparableTo(eShoot.Name),
			expected:      aShoot.Name,
			path:          "metadata/name",
		},
		{
			GomegaMatcher: gomega.BeComparableTo(eShoot.Namespace),
			expected:      aShoot.Namespace,
			path:          "metadata/namespace",
		},
		{
			GomegaMatcher: gomega.BeComparableTo(eShoot.Labels),
			expected:      aShoot.Labels,
			path:          "metadata/labels",
		},
		{
			GomegaMatcher: gomega.BeComparableTo(eShoot.Spec.Addons),
			expected:      aShoot.Spec.Addons,
			path:          "spec/Addons",
		},
		{
			GomegaMatcher: gomega.BeComparableTo(eShoot.Spec.CloudProfileName),
			expected:      aShoot.Spec.CloudProfileName,
			path:          "spec/CloudProfileName",
		},
		{
			GomegaMatcher: gomega.BeComparableTo(eShoot.Spec.DNS),
			expected:      aShoot.Spec.DNS,
			path:          "spec/DNS",
		},
		{
			GomegaMatcher: NewExtensionMatcher(eShoot.Spec.Extensions),
			expected:      aShoot.Spec.Extensions,
			path:          "spec/Extensions",
		},
		{
			GomegaMatcher: gomega.BeComparableTo(eShoot.Spec.Hibernation),
			expected:      aShoot.Spec.Hibernation,
			path:          "spec/Hibernation",
		},
		{
			GomegaMatcher: gomega.BeComparableTo(eShoot.Spec.Kubernetes),
			expected:      aShoot.Spec.Kubernetes,
			path:          "spec/Kubernetes",
		},
		{
			GomegaMatcher: gomega.BeComparableTo(eShoot.Spec.Networking),
			expected:      aShoot.Spec.Networking,
			path:          "spec/Networking",
		},
		{
			GomegaMatcher: gomega.BeComparableTo(eShoot.Spec.Maintenance),
			expected:      aShoot.Spec.Maintenance,
			path:          "spec/Maintenance",
		},
		{
			GomegaMatcher: gomega.BeComparableTo(eShoot.Spec.Monitoring),
			expected:      aShoot.Spec.Monitoring,
			path:          "spec/Monitoring",
		},
		{
			GomegaMatcher: gomega.BeComparableTo(eShoot.Spec.Provider),
			expected:      aShoot.Spec.Provider,
			path:          "spec/Provider",
		},

		{
			GomegaMatcher: gomega.BeComparableTo(eShoot.Spec.Purpose),
			expected:      aShoot.Spec.Purpose,
			path:          "spec/Purpose",
		},
		{
			GomegaMatcher: gomega.BeComparableTo(eShoot.Spec.Region),
			expected:      aShoot.Spec.Region,
			path:          "spec/Region",
		},
		{
			GomegaMatcher: gomega.BeComparableTo(eShoot.Spec.SecretBindingName),
			expected:      aShoot.Spec.SecretBindingName,
			path:          "spec/SecretBindingName",
		},
		{
			GomegaMatcher: gomega.BeComparableTo(eShoot.Spec.SeedName),
			expected:      aShoot.Spec.SeedName,
			path:          "spec/SeedName",
		},
		{
			GomegaMatcher: gomega.BeComparableTo(eShoot.Spec.SeedSelector),
			expected:      aShoot.Spec.SeedSelector,
			path:          "spec/SeedSelector",
		},
		{
			GomegaMatcher: gomega.BeComparableTo(eShoot.Spec.Resources),
			expected:      aShoot.Spec.Resources,
			path:          "spec/Resources",
		},
		{
			GomegaMatcher: gomega.BeComparableTo(eShoot.Spec.Tolerations),
			expected:      aShoot.Spec.Tolerations,
			path:          "spec/Tolerations",
		},
		{
			GomegaMatcher: gomega.BeComparableTo(eShoot.Spec.ExposureClassName),
			expected:      aShoot.Spec.ExposureClassName,
			path:          "spec/ExposureClassName",
		},
		{
			GomegaMatcher: gomega.BeComparableTo(eShoot.Spec.SystemComponents),
			expected:      aShoot.Spec.SystemComponents,
			path:          "spec/SystemComponents",
		},
		{
			GomegaMatcher: gomega.BeComparableTo(eShoot.Spec.ControlPlane),
			expected:      aShoot.Spec.ControlPlane,
			path:          "spec/ControlPlane",
		},
		{
			GomegaMatcher: gomega.BeComparableTo(eShoot.Spec.SchedulerName),
			expected:      aShoot.Spec.SchedulerName,
			path:          "spec/SchedulerName",
		},
		{
			GomegaMatcher: gomega.BeComparableTo(eShoot.Spec.CloudProfile),
			expected:      aShoot.Spec.CloudProfile,
			path:          "spec/CloudProfile",
		},
		{
			GomegaMatcher: gomega.BeComparableTo(eShoot.Spec.CredentialsBindingName),
			expected:      aShoot.Spec.CredentialsBindingName,
			path:          "spec/CredentialsBindingName",
		},
	} {
		ok, err := matcher.Match(matcher.expected)
		if err != nil {
			return false, err
		}

		if !ok {
			msg := matcher.FailureMessage(matcher.expected)
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
