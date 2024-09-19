package shoot

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

type ExtensionMatcher struct {
	toMatch interface{}
	fails   []string
}

func NewExtensionMatcher(i interface{}) types.GomegaMatcher {
	return &ExtensionMatcher{
		toMatch: i,
	}
}

func (m *ExtensionMatcher) Match(actual interface{}) (success bool, err error) {
	aExtensions, err := getExtension(actual)
	if err != nil {
		return false, err
	}

	eExtensions, err := getExtension(m.toMatch)
	if err != nil {
		return false, err
	}

	sort.Sort(Extensions(aExtensions))
	sort.Sort(Extensions(eExtensions))

	return gomega.BeComparableTo(eExtensions).Match(aExtensions)
}

func getExtension(i interface{}) ([]v1beta1.Extension, error) {
	switch v := i.(type) {
	case []v1beta1.Extension:
		return v, nil

	default:
		return []v1beta1.Extension{}, fmt.Errorf(`%w: %s`, errInvalidType, reflect.TypeOf(v))
	}
}

type Extensions []v1beta1.Extension

func (e Extensions) Len() int {
	return len(e)
}

func (e Extensions) Less(i, j int) bool {
	return e[i].Type < e[j].Type
}

func (e Extensions) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

func (m *ExtensionMatcher) NegatedFailureMessage(_ interface{}) string {
	return "expected should not equal actual"
}

func (m *ExtensionMatcher) FailureMessage(_ interface{}) string {
	return strings.Join(m.fails, "\n")
}
