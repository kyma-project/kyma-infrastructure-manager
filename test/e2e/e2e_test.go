package e2e

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/kyma-project/infrastructure-manager/test/e2e/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	testTimeout        = 20 * time.Minute
	testInterval       = 20 * time.Second
	createManifestPath = "test/e2e/resources/runtimes/test-simple-provision.yaml"
	updateManifestPath = "test/e2e/resources/runtimes/test-simple-update.yaml"
)

var runtimeName string

var _ = Describe("Manager", Ordered, func() {
	var controllerPodName string

	BeforeAll(func() {
		By("installing CRDs")
		cmd := exec.Command("make", "install")
		_, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to install CRDs")

		By("deploying the KIM controller")
		cmd = exec.Command("make", "deploy", fmt.Sprintf("IMG=%s", projectImage))
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to deploy the controller-manager")

		By("get the Runtime name from the manifest")
		runtimeName, err = utils.FetchRuntimeCRName(createManifestPath)
		Expect(err).NotTo(HaveOccurred(), "Failed to get the Runtime name from the manifest")
	})

	AfterEach(func() {
		specReport := CurrentSpecReport()
		if specReport.Failed() {
			By("Fetching controller manager pod logs")
			cmd := exec.Command("kubectl", "logs", controllerPodName, "-n", namespace)
			controllerLogs, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Controller logs:\n %s", controllerLogs)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get Controller logs: %s", err)
				_, _ = fmt.Fprint(GinkgoWriter, "Describing the controller pod for more information")
				cmd = exec.Command("kubectl", "describe", "pod", controllerPodName, "-n", namespace)
				controllerDescribe, err := utils.Run(cmd)
				if err != nil {
					_, _ = fmt.Fprintf(GinkgoWriter, "Failed to describe the controller pod: %s", err)
				} else {
					_, _ = fmt.Fprintf(GinkgoWriter, "Controller pod description:\n %s", controllerDescribe)
				}
			}
		}
	})

	SetDefaultEventuallyTimeout(2 * time.Minute)
	SetDefaultEventuallyPollingInterval(time.Second)

	Context("Kyma Infrastructure Manager", func() {
		It("should run successfully", func() {
			By("validating that the infrastructure-manager pod is running as expected")
			verifyControllerUp := func(g Gomega) {
				// Get the name of the KIM controller pod
				cmd := exec.Command("kubectl", "get",
					"pods", "-l", "control-plane=infrastructure-manager",
					"-o", "go-template={{ range .items }}"+
						"{{ if not .metadata.deletionTimestamp }}"+
						"{{ .metadata.name }}"+
						"{{ \"\\n\" }}{{ end }}{{ end }}",
					"-n", namespace,
				)

				podOutput, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to retrieve infrastructure-manager pod information")
				podNames := utils.GetNonEmptyLines(podOutput)
				g.Expect(podNames).To(HaveLen(1), "expected 1 controller pod running")
				controllerPodName = podNames[0]
				g.Expect(controllerPodName).To(ContainSubstring("infrastructure-manager"))

				// Validate the pod's status
				cmd = exec.Command("kubectl", "get",
					"pods", controllerPodName, "-o", "jsonpath={.status.phase}",
					"-n", namespace,
				)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("Running"), "Incorrect infrastructure-manager pod status")
			}
			Eventually(verifyControllerUp).Should(Succeed())
		})
		It("should create an Gardener Shoot from given RuntimeCR", func() {
			By("applying the RuntimeCR manifest")
			cmd := exec.Command("kubectl", "apply", "-f", createManifestPath, "-n", namespace)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to apply the RuntimeCR manifest")
			By("waiting for the RuntimeCR to be in the 'Ready' state")
			waitForRuntimeToBeReady()
		})

		It("should update the Gardener Shoot from given RuntimeCR", func() {
			By("applying the RuntimeCR manifest")
			cmd := exec.Command("kubectl", "apply", "-f", updateManifestPath, "-n", namespace)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to update the RuntimeCR from the manifest")
			waitForRuntimeToBeInPendingState()
			waitForRuntimeToBeReady()
		})

		It("should delete the Gardener Shoot from given RuntimeCR", func() {
			By("deleting the RuntimeCR")
			cmd := exec.Command("kubectl", "delete", "-f", updateManifestPath, "-n", namespace)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to execute the RuntimeCR deletion from the manifest")
			waitForRuntimeToBeDeleted()
		})
	})
})

func waitForRuntimeToBeReady() {
	By("waiting for the RuntimeCR to be in the 'Ready' state")
	Eventually(func(g Gomega) {
		cmd := exec.Command("kubectl", "get", "runtimes", runtimeName, "-n", namespace, "-o", "jsonpath={.status.state}")
		output, err := utils.Run(cmd)
		g.Expect(err).NotTo(HaveOccurred(), "Failed to get RuntimeCR status")
		g.Expect(output).To(Equal("Ready"), "RuntimeCR is not in 'Ready' state")
	}, testTimeout, testInterval).Should(Succeed())
}

func waitForRuntimeToBeInPendingState() {
	By("waiting for the RuntimeCR to be in the 'Pending' state after update")
	Eventually(func(g Gomega) {
		cmd := exec.Command("kubectl", "get", "runtimes", runtimeName, "-n", namespace, "-o", "jsonpath={.status.state}")
		output, err := utils.Run(cmd)
		g.Expect(err).NotTo(HaveOccurred(), "Failed to get RuntimeCR status")
		g.Expect(output).To(Equal("Pending"), "RuntimeCR is not in 'Pending' state")
	}, testTimeout, testInterval).Should(Succeed())
}

func waitForRuntimeToBeDeleted() {
	By("check if the RuntimeCR was deleted")
	Eventually(func(g Gomega) {
		cmd := exec.Command("kubectl", "get", "runtimes", runtimeName, "-n", namespace)
		output, _ := utils.Run(cmd)
		g.Expect(output).To(ContainSubstring("NotFound"), "Output should contain 'NotFound' indicating the RuntimeCR was deleted")

	}, testTimeout, testInterval).Should(Succeed())
}
