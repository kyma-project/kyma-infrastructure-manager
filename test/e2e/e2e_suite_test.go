package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/kyma-project/infrastructure-manager/test/e2e/utils"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	setupResourcesDir = "test/e2e/resources/setup"
	namespace         = "kcp-system"
)

var (
	// projectImage is the name of the image which will be build and loaded
	// with the code source changes to be tested.
	projectImage   = "kyma-infrastructure-manager:local"
	k3dClusterName = ""
)

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	_, _ = fmt.Fprintf(GinkgoWriter, "Starting Kyma Infrastructure Manager integration test suite\n")

	// Verbosity and tracing are set to true to provide detailed output during test execution [for test]
	suiteConfig, reporterConfig := GinkgoConfiguration()
	reporterConfig.Verbose = true
	reporterConfig.FullTrace = true
	suiteConfig.Timeout = time.Minute * 45

	RunSpecs(t, "e2e suite", suiteConfig, reporterConfig)
}

var _ = BeforeSuite(func() {
	By("Checking the cluster context")
	k3dKubeconfig := os.Getenv("KUBECONFIG_K3D")
	Expect(k3dKubeconfig).NotTo(BeEmpty())

	err := os.Setenv("KUBECONFIG", k3dKubeconfig)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to set KUBECONFIG environment variable")

	cmd := exec.Command("kubectl", "config", "current-context")
	k3dClusterName, err = utils.Run(cmd)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to get current kubectl context")

	k3dClusterName = strings.TrimSpace(k3dClusterName)
	k3dClusterName = strings.TrimPrefix(k3dClusterName, "k3d-")
	_, _ = fmt.Fprintf(GinkgoWriter, "Current kubectl context is %s\n", k3dClusterName)

	By("printing Kubernetes version")
	cmd = exec.Command("kubectl", "version")
	version, err := utils.Run(cmd)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to get Kubernetes version")
	_, _ = fmt.Fprintf(GinkgoWriter, "Kubernetes version: %s\n", version)

	By("building the Kyma Infrastructure Manager image")
	cmd = exec.Command("make", "docker-build-e2e", fmt.Sprintf("IMG=%s", projectImage))
	_, err = utils.Run(cmd)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to build the Kyma Infrastructure Manager image")

	By("loading the Kyma Infrastructure Manager image image on K3d")
	err = utils.LoadImageToK3SClusterWithName(projectImage, k3dClusterName)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to load the Kyma Infrastructure Manager image into K3d")

	if !utils.IsPrometheusCRDsInstalled() {
		By("Installing Prometheus CRDs")
		_, _ = fmt.Fprintf(GinkgoWriter, "Installing Prometheus...\n")
		Expect(utils.InstallPrometheusOperator()).To(Succeed(), "Failed to install Prometheus")
	} else {
		_, _ = fmt.Fprintf(GinkgoWriter, "WARNING: Prometheus is already installed. Skipping installation...\n")
	}

	By("creating manager namespace")
	cmd = exec.Command("kubectl", "create", "ns", namespace)
	_, err = utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Failed to create namespace")

	By("installing ConfigMaps and Secrets for KIM controller")
	cmd = exec.Command("kubectl", "apply", "-f", setupResourcesDir)
	_, err = utils.Run(cmd)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to apply setup resources")

	By("generating files")
	cmd = exec.Command("make", "generate")
	_, err = utils.Run(cmd)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to run make generate")

	By("generating manifests")
	cmd = exec.Command("make", "manifests")
	_, err = utils.Run(cmd)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to run make manifests")
})
