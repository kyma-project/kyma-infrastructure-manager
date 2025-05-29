package customconfig

import (
	"context"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/controller/customconfig/mocks"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"path/filepath"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"testing"
)

var (
	cfg        *rest.Config
	k8sClient  client.Client
	testEnv    *envtest.Environment
	suiteCtx   context.Context
	cancelFunc context.CancelFunc
	reconciler *CustomSKRConfigReconciler
)

func TestCustomConfigController(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Custom Config Controller Suite")
}

var _ = BeforeSuite(func() {
	logger := zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true))
	ctrl.SetLogger(logger)

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	k8sClient, err = client.New(cfg, client.Options{Scheme: runtime.NewScheme()})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	err = imv1.AddToScheme(k8sClient.Scheme())
	Expect(err).NotTo(HaveOccurred())

	err = v1.AddToScheme(k8sClient.Scheme())
	Expect(err).NotTo(HaveOccurred())

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: k8sClient.Scheme(),
		Metrics: server.Options{
			BindAddress: ":8083",
		},
	})
	Expect(err).ToNot(HaveOccurred())

	reconciler = NewCustomConfigReconciler(mgr, logger, fixRegistryCacheCreator())
	Expect(reconciler).NotTo(BeNil())
	err = reconciler.SetupWithManager(mgr, 1)
	Expect(err).To(BeNil())

	go func() {
		defer GinkgoRecover()
		suiteCtx, cancelFunc = context.WithCancel(context.Background())
		err = mgr.Start(suiteCtx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()
})

func fixRegistryCacheCreator() func(secret v1.Secret) (RegistryCache, error) {
	const clusterWithCustomConfig = "kubeconfig-cluster-1"
	const clusterWithoutCustomConfig = "kubeconfig-cluster-2"
	const secretNotManagedByKIM = "kubeconfig-cluster-3"

	callsMap := map[string]int{
		clusterWithCustomConfig:    0,
		clusterWithoutCustomConfig: 0,
		secretNotManagedByKIM:      0,
	}

	resultsMap := map[string]bool{
		clusterWithCustomConfig:    true,
		clusterWithoutCustomConfig: false,
		secretNotManagedByKIM:      true,
	}

	return func(secret v1.Secret) (RegistryCache, error) {

		if _, found := callsMap[secret.Name]; !found {
			return nil, errors.Errorf("unexpected secret name %s", secret.Name)
		}

		if callsMap[secret.Name] == 0 {
			callsMap[secret.Name]++
			return nil, errors.New("failed to get registry cache config")
		}

		registryCacheMock := &mocks.RegistryCache{}
		registryCacheMock.On("RegistryCacheConfigExists").Return(resultsMap[secret.Name], nil)

		return registryCacheMock, nil
	}
}

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancelFunc()
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
