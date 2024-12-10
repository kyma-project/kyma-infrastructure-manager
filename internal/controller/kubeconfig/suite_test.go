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

package kubeconfig

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	infrastructuremanagerv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	kubeconfig_mocks "github.com/kyma-project/infrastructure-manager/internal/controller/kubeconfig/mocks"
	metrics "github.com/kyma-project/infrastructure-manager/internal/controller/metrics"
	. "github.com/onsi/ginkgo/v2" //nolint:revive
	. "github.com/onsi/gomega"    //nolint:revive
	"github.com/pkg/errors"
	. "github.com/stretchr/testify/mock" //nolint:revive
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	cfg            *rest.Config                                              //nolint:gochecknoglobals
	k8sClient      client.Client                                             //nolint:gochecknoglobals
	testEnv        *envtest.Environment                                      //nolint:gochecknoglobals
	suiteCtx       context.Context                                           //nolint:gochecknoglobals
	cancelSuiteCtx context.CancelFunc                                        //nolint:gochecknoglobals
	anyContext     = MatchedBy(func(_ context.Context) bool { return true }) //nolint:gochecknoglobals
)

const TestMinimalRotationTimeRatio = 0.5
const TestKubeconfigValidityTime = 24 * time.Hour
const TestKubeconfigRotationPeriod = time.Duration(float64(TestKubeconfigValidityTime) * TestMinimalRotationTimeRatio)
const TestGardenerRequestTimeout = 60 * time.Second

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Kubeconfig Controller Suite")
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

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{Scheme: scheme.Scheme})
	Expect(err).ToNot(HaveOccurred())

	kubeconfigProviderMock := &kubeconfig_mocks.KubeconfigProvider{}
	setupKubeconfigProviderMock(kubeconfigProviderMock)

	metrics := metrics.NewMetrics()

	gardenerClusterController := NewGardenerClusterController(mgr, kubeconfigProviderMock, logger, TestKubeconfigRotationPeriod, TestMinimalRotationTimeRatio, TestGardenerRequestTimeout, metrics)

	Expect(gardenerClusterController).NotTo(BeNil())

	err = gardenerClusterController.SetupWithManager(mgr, 1)
	Expect(err).To(BeNil())

	Expect(gardenerClusterController).NotTo(BeNil())
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

func setupKubeconfigProviderMock(kpMock *kubeconfig_mocks.KubeconfigProvider) {
	kpMock.On("Fetch", anyContext, "shootName1").Return("kubeconfig1", nil)
	kpMock.On("Fetch", anyContext, "shootName2").Return("kubeconfig2", nil)
	kpMock.On("Fetch", anyContext, "shootName3").Return("", errors.New("this could be context deadline exceeded"))
	kpMock.On("Fetch", anyContext, "shootName6").Return("kubeconfig6", nil)
	kpMock.On("Fetch", anyContext, "shootName4").Return("kubeconfig4", nil)
	kpMock.On("Fetch", anyContext, "shootName5").Return("kubeconfig5", nil)
}

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancelSuiteCtx()
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
