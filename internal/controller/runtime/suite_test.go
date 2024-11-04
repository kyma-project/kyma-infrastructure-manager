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
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	gardener_api "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardener_oidc "github.com/gardener/oidc-webhook-authenticator/apis/authentication/v1alpha1"
	infrastructuremanagerv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/auditlogging"
	"github.com/kyma-project/infrastructure-manager/internal/controller/metrics/mocks"
	"github.com/kyma-project/infrastructure-manager/internal/controller/runtime/fsm"
	"github.com/kyma-project/infrastructure-manager/pkg/config"
	gardener_shoot "github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/auditlogs"
	. "github.com/onsi/ginkgo/v2" //nolint:revive
	. "github.com/onsi/gomega"    //nolint:revive
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	v1 "k8s.io/api/autoscaling/v1"
	v12 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	//nolint:revive
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	clienttesting "k8s.io/client-go/testing"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	cfg                       *rest.Config         //nolint:gochecknoglobals
	k8sClient                 client.Client        //nolint:gochecknoglobals
	k8sFakeClientRoleBindings client.Client        //nolint:gochecknoglobals
	gardenerTestClient        client.Client        //nolint:gochecknoglobals
	testEnv                   *envtest.Environment //nolint:gochecknoglobals
	suiteCtx                  context.Context      //nolint:gochecknoglobals
	cancelSuiteCtx            context.CancelFunc   //nolint:gochecknoglobals
	runtimeReconciler         *RuntimeReconciler   //nolint:gochecknoglobals
	customTracker             *CustomTracker       //nolint:gochecknoglobals
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
	_ = infrastructuremanagerv1.AddToScheme(clientScheme)

	// tracker will be updated with different shoot sequence for each test case
	tracker := clienttesting.NewObjectTracker(clientScheme, serializer.NewCodecFactory(clientScheme).UniversalDecoder())
	customTracker = NewCustomTracker(tracker, []*gardener_api.Shoot{}, []*gardener_api.Seed{})
	gardenerTestClient = fake.NewClientBuilder().WithScheme(clientScheme).WithObjectTracker(customTracker).Build()

	convConfig := fixConverterConfigForTests()

	mm := &mocks.Metrics{}
	mm.On("SetRuntimeStates", mock.Anything).Return()
	mm.On("IncRuntimeFSMStopCounter").Return()
	mm.On("CleanUpRuntimeGauge", mock.Anything, mock.Anything).Return()

	fsmCfg := fsm.RCCfg{
		Finalizer:                   infrastructuremanagerv1.Finalizer,
		Config:                      convConfig,
		Metrics:                     mm,
		AuditLogging:                map[string]map[string]auditlogs.AuditLogData{},
		GardenerRequeueDuration:     3 * time.Second,
		ControlPlaneRequeueDuration: 3 * time.Second,
	}

	runtimeReconciler = NewRuntimeReconciler(mgr, gardenerTestClient, logger, fsmCfg)
	Expect(runtimeReconciler).NotTo(BeNil())
	err = runtimeReconciler.SetupWithManager(mgr)
	Expect(err).To(BeNil())

	//+kubebuilder:scaffold:scheme
	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	shootClientScheme := runtime.NewScheme()
	_ = rbacv1.AddToScheme(shootClientScheme)
	err = gardener_oidc.AddToScheme(shootClientScheme)
	k8sFakeClientRoleBindings = fake.NewClientBuilder().WithScheme(shootClientScheme).Build()

	fsm.GetShootClient = func(_ context.Context, _ client.SubResourceClient, _ *gardener_api.Shoot) (client.Client, error) {
		return k8sFakeClientRoleBindings, nil
	}

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
	seeds := fixSeedsSequenceForProvisioning()
	setupGardenerClientWithSequence(shoots, seeds)
}

func setupGardenerTestClientForUpdate() {
	baseShoot := getBaseShootForTestingSequence()
	shoots := fixShootsSequenceForUpdate(&baseShoot)
	seeds := fixSeedsSequenceForUpdate()
	setupGardenerClientWithSequence(shoots, seeds)
}

func setupGardenerTestClientForDelete() {
	baseShoot := getBaseShootForTestingSequence()
	shoots := fixShootsSequenceForDelete(&baseShoot)
	setupGardenerClientWithSequence(shoots, nil)
}

func setupGardenerClientWithSequence(shoots []*gardener_api.Shoot, seeds []*gardener_api.Seed) {
	clientScheme := runtime.NewScheme()
	_ = gardener_api.AddToScheme(clientScheme)

	tracker := clienttesting.NewObjectTracker(clientScheme, serializer.NewCodecFactory(clientScheme).UniversalDecoder())
	customTracker = NewCustomTracker(tracker, shoots, seeds)
	gardenerTestClient = fake.NewClientBuilder().WithScheme(clientScheme).WithObjectTracker(customTracker).
		WithInterceptorFuncs(interceptor.Funcs{
			Create: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
				shoot, ok := obj.(*gardener_api.Shoot)
				// ommit non gardener.Shoot create calls
				if !ok {
					return client.Create(ctx, obj, opts...)
				}
				// simulate gardener behavior
				if shoot.Spec.SeedName == nil || *shoot.Spec.SeedName == "" {
					shoot.Spec.SeedName = ptr.To("test-seed")
				}
				// continue with gardener shoot creation
				return client.Create(ctx, obj, opts...)
			},
			Patch: func(ctx context.Context, clnt client.WithWatch, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
				// Apply patches are supposed to upsert, but fake client fails if the object doesn't exist,
				// Update the generation to simulate the object being updated using interceptor function.
				if patch.Type() != types.ApplyPatchType {
					return clnt.Patch(ctx, obj, patch, opts...)
				}
				shoot, ok := obj.(*gardener_api.Shoot)
				if !ok {
					return errors.New("failed to cast object to shoot")
				}
				shoot.Generation++
				return nil
			}}).Build()
	runtimeReconciler.ShootClient = gardenerTestClient
}

func getBaseShootForTestingSequence() gardener_api.Shoot {
	runtimeStub := CreateRuntimeStub("test-resource")
	infrastructureManagerConfig := fixConverterConfigForTests()
	converter := gardener_shoot.NewConverterCreate(gardener_shoot.CreateOpts{
		ConverterConfig: infrastructureManagerConfig.ConverterConfig,
	})
	convertedShoot, err := converter.ToShoot(*runtimeStub)
	if err != nil {
		panic(err)
	}
	return convertedShoot
}

func fixShootsSequenceForProvisioning(shoot *gardener_api.Shoot) []*gardener_api.Shoot {
	var missingShoot *gardener_api.Shoot
	initialisedShoot := shoot.DeepCopy()
	initialisedShoot.Spec.SeedName = ptr.To("test-seed")

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
	existingShoot := shoot.DeepCopy()
	existingShoot.Spec.SeedName = ptr.To("test-seed")

	existingShoot.Status = gardener_api.ShootStatus{
		LastOperation: &gardener_api.LastOperation{
			Type:  gardener_api.LastOperationTypeReconcile,
			State: gardener_api.LastOperationStateSucceeded,
		},
	}

	existingShoot.Spec.DNS = &gardener_api.DNS{
		Domain: ptr.To("test.domain"),
	}

	addAuditLogConfigToShoot(existingShoot)

	pendingShoot := existingShoot.DeepCopy()

	pendingShoot.ObjectMeta.Annotations["infrastructuremanager.kyma-project.io/runtime-generation"] = "2"

	pendingShoot.Status.LastOperation.State = gardener_api.LastOperationStatePending

	processingShoot := pendingShoot.DeepCopy()

	processingShoot.Status.LastOperation.State = gardener_api.LastOperationStateProcessing

	readyShoot := processingShoot.DeepCopy()

	readyShoot.Status.LastOperation.State = gardener_api.LastOperationStateSucceeded

	// processedShoot := processingShoot.DeepCopy() // will add specific data later

	return []*gardener_api.Shoot{existingShoot, pendingShoot, processingShoot, readyShoot, readyShoot}
}

func fixShootsSequenceForDelete(shoot *gardener_api.Shoot) []*gardener_api.Shoot {
	currentShoot := shoot.DeepCopy()

	currentShoot.Spec.DNS = &gardener_api.DNS{
		Domain: ptr.To("test.domain"),
	}

	currentShoot.Spec.SeedName = ptr.To("test-seed")

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

	return []*gardener_api.Shoot{currentShoot, currentShoot, currentShoot, currentShoot, pendingDeleteShoot, nil}
}

func fixSeedsSequenceForProvisioning() []*gardener_api.Seed {
	seed := &gardener_api.Seed{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-seed",
		},
		Spec: gardener_api.SeedSpec{
			Provider: gardener_api.SeedProvider{
				Type: "aws",
			},
		},
	}

	return []*gardener_api.Seed{seed}
}

func fixSeedsSequenceForUpdate() []*gardener_api.Seed {
	seed := &gardener_api.Seed{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-seed",
		},
		Spec: gardener_api.SeedSpec{
			Provider: gardener_api.SeedProvider{
				Type: "aws",
			},
		},
	}

	return []*gardener_api.Seed{seed}
}

func setupSeedObjectOnCluster(client client.Client) error {
	seed := &gardener_api.Seed{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-seed",
		},
		Spec: gardener_api.SeedSpec{
			Provider: gardener_api.SeedProvider{
				Type: "aws",
			},
		},
	}

	return client.Create(context.Background(), seed)
}

func fixConverterConfigForTests() config.Config {
	return config.Config{
		ConverterConfig: config.ConverterConfig{
			Kubernetes: config.KubernetesConfig{
				DefaultVersion: "1.29",
			},

			DNS: config.DNSConfig{
				SecretName:   "aws-route53-secret-dev",
				DomainPrefix: "dev.kyma.ondemand.com",
				ProviderType: "aws-route53",
			},
			Provider: config.ProviderConfig{
				AWS: config.AWSConfig{
					EnableIMDSv2: true,
				},
			},
			Gardener: config.GardenerConfig{
				ProjectName: "kyma-dev",
			},
			AuditLog: config.AuditLogConfig{
				PolicyConfigMapName: "policy-config-map",
				TenantConfigPath:    filepath.Join("testdata", "auditConfig.json"),
			},
		},
	}
}

func addAuditLogConfigToShoot(shoot *gardener_api.Shoot) {
	shoot.Spec.Kubernetes.KubeAPIServer.AuditConfig = &gardener_api.AuditConfig{
		AuditPolicy: &gardener_api.AuditPolicy{
			ConfigMapRef: &v12.ObjectReference{Name: "policy-config-map"},
		},
	}

	shoot.Spec.Resources = append(shoot.Spec.Resources, gardener_api.NamedResourceReference{
		Name: "auditlog-credentials",
		ResourceRef: v1.CrossVersionObjectReference{
			Kind:       "Secret",
			Name:       "auditlog-secret",
			APIVersion: "v1",
		},
	})

	const (
		extensionKind    = "AuditlogConfig"
		extensionVersion = "service.auditlog.extensions.gardener.cloud/v1alpha1"
		extensionType    = "standard"
	)

	shoot.Spec.Extensions = append(shoot.Spec.Extensions, gardener_api.Extension{
		Type: "shoot-auditlog-service",
	})

	ext := &shoot.Spec.Extensions[len(shoot.Spec.Extensions)-1]

	cfg := auditlogging.AuditlogExtensionConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       extensionKind,
			APIVersion: extensionVersion,
		},
		Type:                extensionType,
		TenantID:            "79c64792-9c1e-4c1b-9941-ef7560dd3eae",
		ServiceURL:          "https://auditlog.example.com:3001",
		SecretReferenceName: "auditlog-credentials",
	}

	ext.ProviderConfig = &runtime.RawExtension{}
	ext.ProviderConfig.Raw, _ = json.Marshal(cfg)
}
