package configreload

import (
	"context"
	"path/filepath"
	"testing"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	certificatesv1beta1 "k8s.io/api/certificates/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

const (
	suiteNamespace         = "kcp-system"
	aclConfigMapName       = "acl-configmap"
	kcpConfigName          = "kcp-config"
	manifestsConfigName    = "manifests-configmap"
	pullSecretName         = "pull-secret"
	clusterTrustBundleName = "test-trust-bundle"
)

var (
	cfg       *rest.Config
	k8sClient client.Client
	testEnv   *envtest.Environment
	suiteCtx  context.Context
	cancelCtx context.CancelFunc
)

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ConfigReloadWatcher Controller Suite")
}

var _ = BeforeSuite(func() {
	logger := zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true))
	ctrl.SetLogger(logger)

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:           []string{filepath.Join("..", "..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing:       true,
		BinaryAssetsDirectory:       filepath.Join("..", "..", "..", "bin", "k8s"),
		DownloadBinaryAssets:        true,
		DownloadBinaryAssetsVersion: "1.35.0",
	}

	testEnv.ControlPlane.GetAPIServer().Configure().
		Append("feature-gates", "ClusterTrustBundle=true", "ClusterTrustBundleProjection=true").
		Append("runtime-config", "certificates.k8s.io/v1beta1/clustertrustbundles=true")

	var err error
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	scheme := runtime.NewScheme()
	Expect(imv1.AddToScheme(scheme)).To(Succeed())
	Expect(corev1.AddToScheme(scheme)).To(Succeed())
	Expect(certificatesv1beta1.AddToScheme(scheme)).To(Succeed())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme})
	Expect(err).NotTo(HaveOccurred())

	ns := &corev1.Namespace{}
	ns.Name = suiteNamespace
	Expect(k8sClient.Create(context.Background(), ns)).To(Succeed())

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: "0",
		},
	})
	Expect(err).NotTo(HaveOccurred())

	watcher := &ConfigReloadWatcher{
		KcpClient: mgr.GetClient(),
		Namespace: suiteNamespace,
		ConfigMapPredicates: []ObjectUpdatedPredicate{
			{NamespacedName: types.NamespacedName{Name: aclConfigMapName, Namespace: suiteNamespace}},
			{NamespacedName: types.NamespacedName{Name: kcpConfigName, Namespace: suiteNamespace}},
			{NamespacedName: types.NamespacedName{Name: manifestsConfigName, Namespace: suiteNamespace}},
		},
		SecretPredicates: []ObjectUpdatedPredicate{
			{NamespacedName: types.NamespacedName{Name: pullSecretName, Namespace: suiteNamespace}},
		},
		ClusterTrustBundlePredicate: &ObjectUpdatedPredicate{
			NamespacedName: types.NamespacedName{Name: clusterTrustBundleName},
		},
		RuntimePredicate: func(configObject types.NamespacedName, rt imv1.Runtime) bool {
			if configObject.Name == aclConfigMapName && rt.Name == "runtime-excluded" {
				return false
			}
			return true
		},
	}
	Expect(watcher.SetupWithManager(mgr)).To(Succeed())

	suiteCtx, cancelCtx = context.WithCancel(context.Background())
	go func() {
		defer GinkgoRecover()
		Expect(mgr.Start(suiteCtx)).To(Succeed())
	}()
})

var _ = AfterSuite(func() {
	if cancelCtx != nil {
		cancelCtx()
	}
	if testEnv != nil {
		Expect(testEnv.Stop()).To(Succeed())
	}
})
