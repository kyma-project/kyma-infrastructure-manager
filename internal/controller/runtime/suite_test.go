/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package runtime

import (
	"context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"path/filepath"
	"testing"
	"time"

	gardener_api "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	infrastructuremanagerv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/controller/runtime/fsm"
	gardener_shoot "github.com/kyma-project/infrastructure-manager/internal/gardener/shoot"
	. "github.com/onsi/ginkgo/v2" //nolint:revive
	. "github.com/onsi/gomega"    //nolint:revive
	//nolint:revive
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	clienttesting "k8s.io/client-go/testing"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	cfg                *rest.Config         //nolint:gochecknoglobals
	k8sClient          client.Client        //nolint:gochecknoglobals
	gardenerTestClient client.Client        //nolint:gochecknoglobals
	testEnv            *envtest.Environment //nolint:gochecknoglobals
	suiteCtx           context.Context      //nolint:gochecknoglobals
	cancelSuiteCtx     context.CancelFunc   //nolint:gochecknoglobals
	runtimeReconciler  *RuntimeReconciler   //nolint:gochecknoglobals
	customTracker      *CustomTracker       //nolint:gochecknoglobals
)

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Runtime Controller Suite")
}

var _ = BeforeSuite(func() {
	logger := zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true))
	logf.SetLogger(logger)

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = infrastructuremanagerv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Metrics: metricsserver.Options{
			BindAddress: ":8083",
		},
		Scheme: scheme.Scheme})
	Expect(err).ToNot(HaveOccurred())

	clientScheme := runtime.NewScheme()
	_ = gardener_api.AddToScheme(clientScheme)

	infrastructuremanagerv1.AddToScheme(clientScheme)

	// tracker will be updated with different shoot sequence for each test case
	tracker := clienttesting.NewObjectTracker(clientScheme, serializer.NewCodecFactory(clientScheme).UniversalDecoder())
	customTracker = NewCustomTracker(tracker, []*gardener_api.Shoot{})
	gardenerTestClient = fake.NewClientBuilder().WithScheme(clientScheme).WithObjectTracker(customTracker).Build()

	runtimeReconciler = NewRuntimeReconciler(mgr, gardenerTestClient, logger, fsm.RCCfg{Finalizer: infrastructuremanagerv1.Finalizer})
	Expect(runtimeReconciler).NotTo(BeNil())
	err = runtimeReconciler.SetupWithManager(mgr)
	Expect(err).To(BeNil())

	//+kubebuilder:scaffold:scheme
	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	go func() {
		defer GinkgoRecover()
		suiteCtx, cancelSuiteCtx = context.WithCancel(context.Background())

		err = mgr.Start(suiteCtx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancelSuiteCtx()
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

func setupGardenerTestClientForProvisioning() {
	baseShoot := getBaseShootForTestingSequence()
	shoots := fixShootsSequenceForProvisioning(&baseShoot)
	setupShootClientWithSequence(shoots)
}

func setupGardenerTestClientForUpdate() {
	baseShoot := getBaseShootForTestingSequence()
	shoots := fixShootsSequenceForUpdate(&baseShoot)
	setupShootClientWithSequence(shoots)
}

func setupGardenerTestClientForDelete() {
	baseShoot := getBaseShootForTestingSequence()
	shoots := fixShootsSequenceForDelete(&baseShoot)
	setupShootClientWithSequence(shoots)
}

func setupShootClientWithSequence(shoots []*gardener_api.Shoot) {
	clientScheme := runtime.NewScheme()
	_ = gardener_api.AddToScheme(clientScheme)

	tracker := clienttesting.NewObjectTracker(clientScheme, serializer.NewCodecFactory(clientScheme).UniversalDecoder())
	customTracker = NewCustomTracker(tracker, shoots)
	gardenerTestClient = fake.NewClientBuilder().WithScheme(clientScheme).WithObjectTracker(customTracker).Build()

	runtimeReconciler.UpdateShootClient(gardenerTestClient)
}

func getBaseShootForTestingSequence() gardener_api.Shoot {
	runtimeStub := CreateRuntimeStub("test-resource")
	converterConfig := fixConverterConfigForTests()
	converter := gardener_shoot.NewConverter(converterConfig)
	convertedShoot, err := converter.ToShoot(*runtimeStub)
	if err != nil {
		panic(err)
	}
	return convertedShoot
}

func fixShootsSequenceForProvisioning(shoot *gardener_api.Shoot) []*gardener_api.Shoot {
	var missingShoot *gardener_api.Shoot
	initialisedShoot := shoot.DeepCopy()

	dnsShoot := initialisedShoot.DeepCopy()

	dnsShoot.Spec.DNS = &gardener_api.DNS{
		Domain: ptr.To("test.domain"),
	}

	pendingShoot := dnsShoot.DeepCopy()

	pendingShoot.Status = gardener_api.ShootStatus{
		LastOperation: &gardener_api.LastOperation{
			Type:  gardener_api.LastOperationTypeCreate,
			State: gardener_api.LastOperationStatePending,
		},
	}

	processingShoot := pendingShoot.DeepCopy()

	processingShoot.Status.LastOperation.State = gardener_api.LastOperationStateProcessing

	readyShoot := processingShoot.DeepCopy()

	readyShoot.Status.LastOperation.State = gardener_api.LastOperationStateSucceeded

	// processedShoot := processingShoot.DeepCopy() // will add specific data later

	return []*gardener_api.Shoot{missingShoot, missingShoot, missingShoot, initialisedShoot, dnsShoot, pendingShoot, processingShoot, readyShoot, readyShoot, readyShoot, readyShoot}
}

func fixShootsSequenceForUpdate(shoot *gardener_api.Shoot) []*gardener_api.Shoot {
	pendingShoot := shoot.DeepCopy()

	pendingShoot.Spec.DNS = &gardener_api.DNS{
		Domain: ptr.To("test.domain"),
	}

	pendingShoot.Status = gardener_api.ShootStatus{
		LastOperation: &gardener_api.LastOperation{
			Type:  gardener_api.LastOperationTypeReconcile,
			State: gardener_api.LastOperationStatePending,
		},
	}

	processingShoot := pendingShoot.DeepCopy()

	processingShoot.Status.LastOperation.State = gardener_api.LastOperationStateProcessing

	readyShoot := processingShoot.DeepCopy()

	readyShoot.Status.LastOperation.State = gardener_api.LastOperationStateSucceeded

	// processedShoot := processingShoot.DeepCopy() // will add specific data later

	return []*gardener_api.Shoot{pendingShoot, processingShoot, readyShoot, readyShoot}
}

func fixShootsSequenceForDelete(shoot *gardener_api.Shoot) []*gardener_api.Shoot {
	currentShoot := shoot.DeepCopy()

	currentShoot.Spec.DNS = &gardener_api.DNS{
		Domain: ptr.To("test.domain"),
	}

	// To workaround limitation that apply patches are not supported in the fake client.
	// We need to set the annotation manually.  https://github.com/kubernetes/kubernetes/issues/115598
	currentShoot.Annotations = map[string]string{
		"confirmation.gardener.cloud/deletion": "true",
	}

	currentShoot.Status = gardener_api.ShootStatus{
		LastOperation: &gardener_api.LastOperation{
			Type:  gardener_api.LastOperationTypeCreate,
			State: gardener_api.LastOperationStateSucceeded,
		},
	}

	pendingDeleteShoot := currentShoot.DeepCopy()

	pendingDeleteShoot.SetDeletionTimestamp(&metav1.Time{Time: time.Now()})
	pendingDeleteShoot.Status.LastOperation.Type = gardener_api.LastOperationTypeDelete
	pendingDeleteShoot.Status.LastOperation.State = gardener_api.LastOperationStatePending

	return []*gardener_api.Shoot{currentShoot, currentShoot, currentShoot, pendingDeleteShoot, nil}
}

func fixConverterConfigForTests() gardener_shoot.ConverterConfig {
	return gardener_shoot.ConverterConfig{
		Kubernetes: gardener_shoot.KubernetesConfig{
			DefaultVersion: "1.29", //nolint:godox TODO: Should be parametrised
		},

		DNS: gardener_shoot.DNSConfig{
			SecretName:   "aws-route53-secret-dev",
			DomainPrefix: "dev.kyma.ondemand.com",
			ProviderType: "aws-route53",
		},
		Provider: gardener_shoot.ProviderConfig{
			AWS: gardener_shoot.AWSConfig{
				EnableIMDSv2: true, //nolint:godox TODO: Should be parametrised
			},
		},
		Gardener: gardener_shoot.GardenerConfig{
			ProjectName: "kyma-dev", //nolint:godox TODO: should be parametrised
		},
	}
}
