package fsm

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gardener/gardener-extension-provider-gcp/pkg/apis/gcp/v1alpha1"
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	. "github.com/onsi/ginkgo/v2" //nolint:revive
	. "github.com/onsi/gomega"    //nolint:revive
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	util "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("KIM sFnSelectShootProcessing", func() {
	testCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// GIVEN
	testScheme := runtime.NewScheme()
	util.Must(imv1.AddToScheme(testScheme))
	withTestSchemeAndObjects := func(objs ...client.Object) fakeFSMOpt {
		return func(fsm *fsm) error {
			return withFakedK8sClient(testScheme, objs...)(fsm)
		}
	}

	inputRtWithForceAnnotation := makeInputRuntimeWithAnnotation(map[string]string{"operator.kyma-project.io/force-patch-reconciliation": "true"})
	inputRtWithSuspendAnnotation := makeInputRuntimeWithAnnotation(map[string]string{"operator.kyma-project.io/suspend-patch-reconciliation": "true"})

	testShoot := gardener.Shoot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-shoot",
			Namespace: "garden-",
		},
		Spec: gardener.ShootSpec{
			DNS: &gardener.DNS{
				Domain: ptr.To("test-domain"),
			},
		},
		Status: gardener.ShootStatus{
			LastOperation: &gardener.LastOperation{
				State:          gardener.LastOperationStateSucceeded,
			},
		},
	}

	testFunction := buildTestFunction(sFnSelectShootProcessing)

	DescribeTable(
		"transition graph validation for sFnSelectShootProcessing",
		testFunction,
		Entry(
			"should switch to sFnPatchExistingShoot due to force reconciliation annotation",
			testCtx,
			must(newFakeFSM, withTestFinalizer, withTestSchemeAndObjects()),
			&systemState{instance: *inputRtWithForceAnnotation, shoot: &testShoot},
			testOpts{
				MatchExpectedErr: BeNil(),
				MatchNextFnState: haveName("sFnPatchExistingShoot"),
			},
		),
		Entry(
			"should stop due to suspend annotation",
			testCtx,
			must(newFakeFSM, withTestFinalizer, withTestSchemeAndObjects()),
			&systemState{instance: *inputRtWithSuspendAnnotation, shoot: &testShoot},
			testOpts{
				MatchExpectedErr: BeNil(),
				MatchNextFnState: BeNil(),
			},
		),
	)
})

func makeInputRuntimeWithAnnotation(annotations map[string]string) *imv1.Runtime {
	return &imv1.Runtime{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-shoot",
			Namespace: "kcp-system",
			Labels: map[string]string{
				"kyma-project.io/instance-id":         "instance-id",
				"kyma-project.io/runtime-id":          "runtime-id",
				"kyma-project.io/shoot-name":          "shoot-name",
				"kyma-project.io/region":              "region",
				"operator.kyma-project.io/kyma-name":  "kyma-name",
				"kyma-project.io/broker-plan-id":      "broker-plan-id",
				"kyma-project.io/broker-plan-name":    "broker-plan-name",
				"kyma-project.io/global-account-id":   "global-account-id",
				"kyma-project.io/subaccount-id":       "subaccount-id",
				"operator.kyma-project.io/managed-by": "managed-by",
				"operator.kyma-project.io/internal":   "false",
				"kyma-project.io/platform-region":     "platform-region",
			},
			Annotations: annotations,
		},
		Spec: imv1.RuntimeSpec{
			Shoot: imv1.RuntimeShoot{
				Name:     "test-shoot",
				Region:   "region",
				Provider: imv1.Provider{
					Type:                 "gcp",
					Workers:              fixWorkers("test-worker", "m5.xlarge", "garden-linux", "1.19.8", 1, 1, []string{"europe-west1-d"}),
					InfrastructureConfig: fixGCPInfrastructureConfig(),
					ControlPlaneConfig: fixGCPControlPlaneConfig(),
				},
			},
		},
	}
}

func fixWorkers(name, machineType, machineImageName, machineImageVersion string, min, max int32, zones []string) []gardener.Worker {
	return []gardener.Worker{
		{
			Name: name,
			Machine: gardener.Machine{
				Type: machineType,
				Image: &gardener.ShootMachineImage{
					Name:    machineImageName,
					Version: &machineImageVersion,
				},
			},
			Minimum: min,
			Maximum: max,
			Zones:   zones,
		},
	}
}

func fixGCPInfrastructureConfig() *runtime.RawExtension {
	infraConfig, _ := json.Marshal(NewGCPInfrastructureConfig())
	return &runtime.RawExtension{Raw: infraConfig}
}

func NewGCPInfrastructureConfig() v1alpha1.InfrastructureConfig {
	return v1alpha1.InfrastructureConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       "infrastructureConfig",
			APIVersion: "gcp.provider.extensions.gardener.cloud/v1alpha1",
		},
		Networks: v1alpha1.NetworkConfig{
			Worker: "10.180.0.0/16",
			Workers: "10.180.0.0/16",
		},
	}
}

func fixGCPControlPlaneConfig() *runtime.RawExtension {
	controlPlaneConfig, _ := json.Marshal(NewGCPControlPlaneConfig())
	return &runtime.RawExtension{Raw: controlPlaneConfig}
}

func NewGCPControlPlaneConfig() v1alpha1.ControlPlaneConfig {
	return v1alpha1.ControlPlaneConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ControlPlaneConfig",
			APIVersion: "gcp.provider.extensions.gardener.cloud/v1alpha1",
		},
		Zone: "europe-west1-d",
	}
}
