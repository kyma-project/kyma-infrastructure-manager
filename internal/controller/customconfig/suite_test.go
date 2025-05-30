package customconfig

import (
	"context"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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

	reconciler = NewCustomConfigReconciler(mgr, logger, fixMockedRegistryCache())
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

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancelFunc()
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
