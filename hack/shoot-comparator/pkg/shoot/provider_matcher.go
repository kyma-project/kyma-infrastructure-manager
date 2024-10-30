package shoot

import (
	"fmt"
	"strings"

	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/kyma-project/infrastructure-manager/hack/shoot-comparator/pkg/runtime"
	"github.com/kyma-project/infrastructure-manager/hack/shoot-comparator/pkg/utilz"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	"github.com/onsi/gomega/types"
)

func NewProviderMatcher(v any, path string) types.GomegaMatcher {
	return &ProviderMatcher{
		toMatch:  v,
		rootPath: path,
	}
}

type ProviderMatcher struct {
	toMatch  interface{}
	fails    []string
	rootPath string
}

func (m *ProviderMatcher) getPath(p string) string {
	return fmt.Sprintf("%s/%s", m.rootPath, p)
}

func (m *ProviderMatcher) Match(actual interface{}) (success bool, err error) {
	providerActual, err := utilz.Get[v1beta1.Provider](actual)
	if err != nil {
		return false, err
	}

	providerToMatch, err := utilz.Get[v1beta1.Provider](m.toMatch)
	if err != nil {
		return false, err
	}

	for _, matcher := range []propertyMatcher{
		{
			path:          m.getPath("type"),
			GomegaMatcher: gomega.BeComparableTo(providerToMatch.Type),
			actual:        providerActual.Type,
		},
		{
			path: m.getPath("workers"),
			GomegaMatcher: gstruct.MatchElements(
				idWorker,
				gstruct.IgnoreMissing,
				workers(providerToMatch.Workers),
			),
			actual: providerActual.Workers,
		},
		{
			path:          m.getPath("controlPlaneConfig"),
			GomegaMatcher: runtime.NewRawExtensionMatcher(providerToMatch.ControlPlaneConfig),
			actual:        providerActual.ControlPlaneConfig,
		},
		{
			path:          m.getPath("infrastructureConfig"),
			GomegaMatcher: runtime.NewRawExtensionMatcher(providerToMatch.InfrastructureConfig),
			actual:        providerActual.InfrastructureConfig,
		},
		{
			path:          m.getPath("workerSettings"),
			GomegaMatcher: newWorkerSettingsMatcher(providerToMatch.WorkersSettings),
			actual:        providerActual.WorkersSettings,
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

func (m *ProviderMatcher) NegatedFailureMessage(_ interface{}) string {
	return "expected should not equal actual"
}

func (m *ProviderMatcher) FailureMessage(_ interface{}) string {
	return strings.Join(m.fails, "\n")
}

type propertyMatcher = struct {
	types.GomegaMatcher
	path   string
	actual interface{}
}

func idWorker(v interface{}) string {
	if v == nil {
		panic("nil value")
	}

	w, ok := v.(v1beta1.Worker)
	if !ok {
		panic("invalid type")
	}

	return w.Name
}

func workers(ws []v1beta1.Worker) gstruct.Elements {
	out := map[string]types.GomegaMatcher{}
	for _, w := range ws {
		ID := idWorker(w)
		out[ID] = gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
			"Annotations": gstruct.Ignore(),
			"CABundle":    gstruct.Ignore(),
			"CRI":         gstruct.Ignore(),
			"Kubernetes":  gstruct.Ignore(),
			"Labels":      gstruct.Ignore(),
			"Name":        gomega.BeComparableTo(w.Name),
			"Machine": gstruct.MatchFields(
				gstruct.IgnoreMissing,
				gstruct.Fields{
					"Type":         gomega.BeComparableTo(w.Machine.Type),
					"Image":        newShootMachineImageMatcher(w.Machine.Image),
					"Architecture": gstruct.Ignore(),
				}),
			"Maximum":                          gomega.BeComparableTo(w.Maximum),
			"Minimum":                          gomega.BeComparableTo(w.Minimum),
			"MaxSurge":                         gomega.BeComparableTo(w.MaxSurge),
			"MaxUnavailable":                   gomega.BeComparableTo(w.MaxUnavailable),
			"ProviderConfig":                   runtime.NewRawExtensionMatcher(w.ProviderConfig),
			"Taints":                           gstruct.Ignore(),
			"Volume":                           gomega.BeComparableTo(w.Volume),
			"DataVolumes":                      gstruct.Ignore(),
			"KubeletDataVolumeName":            gstruct.Ignore(),
			"Zones":                            gomega.ContainElements(w.Zones),
			"SystemComponents":                 gstruct.Ignore(),
			"MachineControllerManagerSettings": gstruct.Ignore(),
			"Sysctls":                          gstruct.Ignore(),
			"ClusterAutoscaler":                gstruct.Ignore(),
		})
	}
	return out
}

func newShootMachineImageMatcher(i *v1beta1.ShootMachineImage) types.GomegaMatcher {
	if i == nil {
		return gomega.BeNil()
	}

	return gstruct.PointTo(
		gstruct.MatchFields(
			gstruct.IgnoreMissing,
			gstruct.Fields{
				"Name":           gomega.BeComparableTo(i.Name),
				"Version":        gomega.BeComparableTo(i.Version),
				"ProviderConfig": gstruct.Ignore(),
			}))
}

func newWorkerSettingsMatcher(s *v1beta1.WorkersSettings) types.GomegaMatcher {
	if s == nil || s.SSHAccess == nil {
		return gomega.BeNil()
	}

	return gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
		"SSHAccess": gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
			"Enabled": gomega.BeComparableTo(s.SSHAccess.Enabled),
		})),
	}))
}
