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
	path   string
	actual interface{}
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
		//	actual:      aShoot.TypeMeta,
		//},
		{
			GomegaMatcher: gomega.BeComparableTo(eShoot.Name),
			actual:        aShoot.Name,
			path:          "metadata/name",
		},
		{
			GomegaMatcher: gomega.BeComparableTo(eShoot.Namespace),
			actual:        aShoot.Namespace,
			path:          "metadata/namespace",
		},
		{
			GomegaMatcher: gomega.BeComparableTo(eShoot.Labels),
			actual:        aShoot.Labels,
			path:          "metadata/labels",
		},
		{
			GomegaMatcher: gomega.BeComparableTo(eShoot.Spec.Addons),
			actual:        aShoot.Spec.Addons,
			path:          "spec/Addons",
		},
		{
			GomegaMatcher: gomega.BeComparableTo(eShoot.Spec.CloudProfileName),
			actual:        aShoot.Spec.CloudProfileName,
			path:          "spec/CloudProfileName",
		},
		{
			GomegaMatcher: gomega.BeComparableTo(eShoot.Spec.DNS),
			actual:        aShoot.Spec.DNS,
			path:          "spec/DNS",
		},
		{
			GomegaMatcher: NewExtensionMatcher(eShoot.Spec.Extensions),
			actual:        aShoot.Spec.Extensions,
			path:          "spec/Extensions",
		},
		{
			GomegaMatcher: gomega.BeComparableTo(eShoot.Spec.Hibernation),
			actual:        aShoot.Spec.Hibernation,
			path:          "spec/Hibernation",
		},
		{
			GomegaMatcher: gomega.BeComparableTo(eShoot.Spec.Kubernetes),
			actual:        aShoot.Spec.Kubernetes,
			path:          "spec/Kubernetes",
		},
		{
			GomegaMatcher: gomega.BeComparableTo(eShoot.Spec.Networking),
			actual:        aShoot.Spec.Networking,
			path:          "spec/Networking",
		},
		{
			GomegaMatcher: gomega.BeComparableTo(eShoot.Spec.Maintenance),
			actual:        aShoot.Spec.Maintenance,
			path:          "spec/Maintenance",
		},
		{
			GomegaMatcher: gomega.BeComparableTo(eShoot.Spec.Monitoring),
			actual:        aShoot.Spec.Monitoring,
			path:          "spec/Monitoring",
		},
		{
			GomegaMatcher: gomega.BeComparableTo(eShoot.Spec.Provider),
			actual:        aShoot.Spec.Provider,
			path:          "spec/Provider",
		},

		{
			GomegaMatcher: gomega.BeComparableTo(eShoot.Spec.Purpose),
			actual:        aShoot.Spec.Purpose,
			path:          "spec/Purpose",
		},
		{
			GomegaMatcher: gomega.BeComparableTo(eShoot.Spec.Region),
			actual:        aShoot.Spec.Region,
			path:          "spec/Region",
		},
		{
			GomegaMatcher: gomega.BeComparableTo(eShoot.Spec.SecretBindingName),
			actual:        aShoot.Spec.SecretBindingName,
			path:          "spec/SecretBindingName",
		},
		{
			GomegaMatcher: gomega.BeComparableTo(eShoot.Spec.SeedName),
			actual:        aShoot.Spec.SeedName,
			path:          "spec/SeedName",
		},
		{
			GomegaMatcher: gomega.BeComparableTo(eShoot.Spec.SeedSelector),
			actual:        aShoot.Spec.SeedSelector,
			path:          "spec/SeedSelector",
		},
		{
			GomegaMatcher: gomega.BeComparableTo(eShoot.Spec.Resources),
			actual:        aShoot.Spec.Resources,
			path:          "spec/Resources",
		},
		{
			GomegaMatcher: gomega.BeComparableTo(eShoot.Spec.Tolerations),
			actual:        aShoot.Spec.Tolerations,
			path:          "spec/Tolerations",
		},
		{
			GomegaMatcher: gomega.BeComparableTo(eShoot.Spec.ExposureClassName),
			actual:        aShoot.Spec.ExposureClassName,
			path:          "spec/ExposureClassName",
		},
		{
			GomegaMatcher: gomega.BeComparableTo(eShoot.Spec.SystemComponents),
			actual:        aShoot.Spec.SystemComponents,
			path:          "spec/SystemComponents",
		},
		{
			GomegaMatcher: gomega.BeComparableTo(eShoot.Spec.ControlPlane),
			actual:        aShoot.Spec.ControlPlane,
			path:          "spec/ControlPlane",
		},
		{
			GomegaMatcher: gomega.BeComparableTo(eShoot.Spec.SchedulerName),
			actual:        aShoot.Spec.SchedulerName,
			path:          "spec/SchedulerName",
		},
		{
			GomegaMatcher: gomega.BeComparableTo(eShoot.Spec.CloudProfile),
			actual:        aShoot.Spec.CloudProfile,
			path:          "spec/CloudProfile",
		},
		{
			GomegaMatcher: gomega.BeComparableTo(eShoot.Spec.CredentialsBindingName),
			actual:        aShoot.Spec.CredentialsBindingName,
			path:          "spec/CredentialsBindingName",
		},
	} {
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
	return "actual should not equal actual"
}

func (m *Matcher) FailureMessage(_ interface{}) string {
	return strings.Join(m.fails, "\n")
}
