package kubeconfig

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"time"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/controller/metrics"
	. "github.com/onsi/ginkgo/v2" //nolint:revive
	. "github.com/onsi/gomega"    //nolint:revive
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Gardener Cluster controller", func() {
	Context("Secret with kubeconfig doesn't exist", func() {
		It("Should create secret, and set Ready status on CR", func() {
			kymaName := "kymaname1"
			secretName := "secret-name1"
			shootName := "shootName1"
			namespace := "default"

			By("Create GardenerCluster CR")

			gardenerClusterCR := fixGardenerClusterCR(kymaName, namespace, shootName, secretName)
			Expect(k8sClient.Create(context.Background(), &gardenerClusterCR)).To(Succeed())

			By("Wait for secret creation")
			var kubeconfigSecret corev1.Secret
			secretKey := types.NamespacedName{Name: secretName, Namespace: namespace}

			Eventually(func() bool {
				return k8sClient.Get(context.Background(), secretKey, &kubeconfigSecret) == nil
			}, time.Second*30, time.Second*3).Should(BeTrue())

			gardenerClusterKey := types.NamespacedName{Name: gardenerClusterCR.Name, Namespace: gardenerClusterCR.Namespace}
			var newGardenerCluster imv1.GardenerCluster
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), gardenerClusterKey, &newGardenerCluster)
				if err != nil {
					return false
				}

				return newGardenerCluster.Status.State == imv1.ReadyState
			}, time.Second*30, time.Second*3).Should(BeTrue())

			epochParseErr := k8sClient.Get(context.Background(), secretKey, &kubeconfigSecret)
			Expect(epochParseErr).To(BeNil())
			expectedSecret := fixNewSecret(secretName, namespace, kymaName, shootName, "kubeconfig1", "")
			Expect(kubeconfigSecret.Labels).To(Equal(expectedSecret.Labels))
			Expect(kubeconfigSecret.Data).To(Equal(expectedSecret.Data))
			lastSyncTime := kubeconfigSecret.Annotations[lastKubeconfigSyncAnnotation]
			Expect(lastSyncTime).ToNot(BeEmpty())

			metricsData := getMetricsData(kymaName)
			By("ClusterState metrics should be appended after creation")
			Expect(metricsData.gardenerClusterStatesData.reason).To(Equal(string(imv1.ConditionReasonKubeconfigSecretCreated)))
			Expect(metricsData.gardenerClusterStatesData.state).To(Equal(string(imv1.ReadyState)))
			Expect(metricsData.shootName).To(Equal(shootName))

			expectKubeconfigMetricsAreValid(metricsData, lastSyncTime, "Kubeconfig expiration metrics should be appended after creation", shootName)
		})

		It("Should delete secret", func() {
			kymaName := "kymaname2"
			secretName := "secret-name2"
			shootName := "shootName2"
			namespace := "default"

			By("Create GardenerCluster CR")

			gardenerClusterCR := fixGardenerClusterCR(kymaName, namespace, shootName, secretName)
			Expect(k8sClient.Create(context.Background(), &gardenerClusterCR)).To(Succeed())

			By("Wait for secret creation")
			var kubeconfigSecret corev1.Secret
			secretKey := types.NamespacedName{Name: secretName, Namespace: namespace}

			Eventually(func() bool {
				return k8sClient.Get(context.Background(), secretKey, &kubeconfigSecret) == nil
			}, time.Second*30, time.Second*3).Should(BeTrue())

			By("Delete Cluster CR")
			Expect(k8sClient.Delete(context.Background(), &gardenerClusterCR)).To(Succeed())

			By("Wait for secret deletion")
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), secretKey, &kubeconfigSecret)
				secretNotFound := err != nil && k8serrors.IsNotFound(err)
				return secretNotFound
			}, time.Minute*30, time.Second*3).Should(BeTrue())

			By("Metrics should be cleared after deletion")
			Eventually(func() string {
				metricsData := getMetricsData(kymaName)
				return metricsData.gardenerClusterStatesData.reason
			}, time.Second*30, time.Second*3).Should(BeEmpty())

		})

		It("Should set Error status on CR if failed to fetch kubeconfig", func() {
			kymaName := "kymaname3"
			secretName := "secret-name3"
			shootName := "shootName3"
			namespace := "default"

			gardenerClusterCR := fixGardenerClusterCR(kymaName, namespace, shootName, secretName)
			Expect(k8sClient.Create(context.Background(), &gardenerClusterCR)).To(Succeed())

			gardenerClusterKey := types.NamespacedName{Name: gardenerClusterCR.Name, Namespace: gardenerClusterCR.Namespace}
			var newGardenerCluster imv1.GardenerCluster
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), gardenerClusterKey, &newGardenerCluster)
				if err != nil {
					return false
				}

				return newGardenerCluster.Status.State == imv1.ErrorState &&
					len(newGardenerCluster.Status.Conditions) > 0 &&
					newGardenerCluster.Status.Conditions[0].Reason == string(imv1.ConditionReasonFailedToGetKubeconfig) &&
					newGardenerCluster.Status.Conditions[0].Message == "Failed to get kubeconfig. Error: this could be context deadline exceeded"
			}, time.Second*30, time.Second*3).Should(BeTrue())

			By("Metrics should contain error label")
			metricsData := getMetricsData(kymaName)
			Expect(metricsData.gardenerClusterStatesData.reason).To(Equal(string(imv1.ConditionReasonFailedToGetKubeconfig)))
			Expect(metricsData.gardenerClusterStatesData.state).To(Equal(string(imv1.ErrorState)))
			Expect(metricsData.shootName).To(Equal(shootName))
		})
	})

	Context("Secret with kubeconfig exists", func() {
		namespace := "default"
		DescribeTable("Should update secret", func(gardenerClusterCR imv1.GardenerCluster, secret corev1.Secret, expectedKubeconfig string) {
			By("Create kubeconfig secret")
			Expect(k8sClient.Create(context.Background(), &secret)).To(Succeed())

			previousTimestamp := secret.Annotations[lastKubeconfigSyncAnnotation]

			By("Create Cluster CR")
			Expect(k8sClient.Create(context.Background(), &gardenerClusterCR)).To(Succeed())

			var kubeconfigSecret corev1.Secret
			secretKey := types.NamespacedName{Name: secret.Name, Namespace: namespace}

			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), secretKey, &kubeconfigSecret)
				if err != nil {
					return false
				}

				timestampAnnotation := kubeconfigSecret.Annotations[lastKubeconfigSyncAnnotation]

				return timestampAnnotation != previousTimestamp
			}, time.Second*30, time.Second*3).Should(BeTrue())

			gardenerClusterKey := types.NamespacedName{Name: gardenerClusterCR.Name, Namespace: gardenerClusterCR.Namespace}
			var newGardenerCluster imv1.GardenerCluster

			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), gardenerClusterKey, &newGardenerCluster)
				if err != nil {
					return false
				}

				readyState := newGardenerCluster.Status.State == imv1.ReadyState
				_, forceRotationAnnotationFound := newGardenerCluster.GetAnnotations()[forceKubeconfigRotationAnnotation]

				return readyState && !forceRotationAnnotationFound
			}, time.Second*45, time.Second*3).Should(BeTrue())

			err := k8sClient.Get(context.Background(), secretKey, &kubeconfigSecret)
			Expect(err).To(BeNil())
			Expect(string(kubeconfigSecret.Data["config"])).To(Equal(expectedKubeconfig))
			lastSyncTime := kubeconfigSecret.Annotations[lastKubeconfigSyncAnnotation]
			Expect(lastSyncTime).ToNot(BeEmpty())

			metricsData := getMetricsData(gardenerClusterKey.Name)
			expectKubeconfigMetricsAreValid(metricsData, lastSyncTime, "Kubeconfig expiration metrics should be appended after creation", kubeconfigSecret.Labels[metrics.ShootNameLabel])
		},
			Entry("Rotate kubeconfig when rotation epoch passed",
				fixGardenerClusterCR("kymaname4", namespace, "shootName4", "secret-name4"),
				fixNewSecret("secret-name4", namespace, "kymaname4", "shootName4", "kubeconfig4", "2023-10-09T23:00:00Z"),
				"kubeconfig4"),
			Entry("Force rotation",
				fixGardenerClusterCRWithForceRotationAnnotation("kymaname5", namespace, "shootName5", "secret-name5"),
				fixNewSecret("secret-name5", namespace, "kymaname5", "shootName5", "kubeconfig5", time.Now().UTC().Format(time.RFC3339)),
				"kubeconfig5"),
		)

		It("Should skip rotation", func() {
			By("Create kubeconfig secret")
			secret := fixNewSecret("secret-name6", namespace, "kymaname6", "shootName6", "kubeconfig6", time.Now().UTC().Format(time.RFC3339))
			Expect(k8sClient.Create(context.Background(), &secret)).To(Succeed())

			previousTimestamp := secret.Annotations[lastKubeconfigSyncAnnotation]

			By("Create Cluster CR")
			gardenerClusterCR := fixGardenerClusterCR("kymaname6", namespace, "shootName6", "secret-name6")
			Expect(k8sClient.Create(context.Background(), &gardenerClusterCR)).To(Succeed())

			var kubeconfigSecret corev1.Secret
			secretKey := types.NamespacedName{Name: secret.Name, Namespace: namespace}

			Consistently(func() bool {
				err := k8sClient.Get(context.Background(), secretKey, &kubeconfigSecret)
				if err != nil {
					return false
				}

				timestampAnnotation := kubeconfigSecret.Annotations[lastKubeconfigSyncAnnotation]

				return timestampAnnotation == previousTimestamp
			}, time.Second*45, time.Second*3).Should(BeTrue())
		})
	})
})

func expectKubeconfigMetricsAreValid(metricsData metricsData, lastSyncTimeString, stepDescription, shootName string) {
	By(stepDescription)
	Expect(metricsData.shootName).To(Equal(shootName))

	intEpoch, epochParseErr := strconv.ParseInt(metricsData.kubeconfigExpiration.epoch, 10, 64)
	Expect(epochParseErr).To(BeNil())
	expirationMetric := time.Unix(intEpoch, 0)

	lastSyncTimeDateTime, err := time.Parse(time.RFC3339, lastSyncTimeString)
	var kubeconfigValidUntil time.Time
	if err == nil {
		kubeconfigValidUntil = lastSyncTimeDateTime.Add(TestKubeconfigValidityTime)
	}
	Expect(expirationMetric.UTC().Format(time.RFC3339)).To(Equal(kubeconfigValidUntil.Format(time.RFC3339)))

	expirationDuration, expirationDurationParseErr := strconv.ParseFloat(metricsData.kubeconfigExpiration.expirationDuration, 64)
	Expect(expirationDurationParseErr).To(BeNil())
	Expect(expirationDuration).To(Equal(TestKubeconfigValidityTime.Seconds()))
}

type gardenerClusterStatesData struct {
	reason string
	state  string
}

type kubeconfigExpiration struct {
	epoch              string
	expirationDuration string
	rotationDuration   string
}

type metricsData struct {
	gardenerClusterStatesData gardenerClusterStatesData
	kubeconfigExpiration      kubeconfigExpiration
	shootName                 string
}

func getMetricsData(runtimeID string) metricsData {
	data := metricsData{}

	stringBody, _ := getMetricsBody()
	clusterStateMetricRegex := getGardenerClusterStateMetricRegex(runtimeID)
	clusterStateMetricMatches := clusterStateMetricRegex.FindStringSubmatch(stringBody)

	if len(clusterStateMetricMatches) > 0 {
		data.gardenerClusterStatesData.reason = clusterStateMetricMatches[1]
		data.shootName = clusterStateMetricMatches[2]
		data.gardenerClusterStatesData.state = clusterStateMetricMatches[3]
	}

	kubeconfigExpirationMetricRegex := getKubeconfigExpirationMetricRegex(runtimeID)
	kubeconfigExpirationMetricMatches := kubeconfigExpirationMetricRegex.FindStringSubmatch(stringBody)
	if len(kubeconfigExpirationMetricMatches) > 0 {
		data.kubeconfigExpiration.expirationDuration = kubeconfigExpirationMetricMatches[1]
		data.kubeconfigExpiration.epoch = kubeconfigExpirationMetricMatches[2]
		data.kubeconfigExpiration.rotationDuration = kubeconfigExpirationMetricMatches[3]
		data.shootName = kubeconfigExpirationMetricMatches[4]
	}

	return data
}

// getKubeconfigExpirationMetricRegex returns regex that will find matches of kubeconfig_expiration metrics
// and capture a group for given `runtimeId` label value:
// 1) `expirationDuration` label value
// 2) `expires` label value
// 3) `rotationDuration` label value
// 4) `shoot` label value
func getKubeconfigExpirationMetricRegex(runtimeID string) *regexp.Regexp {
	regexString := fmt.Sprintf("im_kubeconfig_expiration{expirationDuration=\"(.*?)\",expires=\"(.*?)\",rotationDuration=\"(.*?)\",runtimeId=\"%v\",shoot=\"(.*?)\"", runtimeID)
	return regexp.MustCompile(regexString)
}

// getGardenerClusterStateMetricRegex returns regex that will find matches of gardener_cluster_state metrics
// and capture three groups for given `runtimeId` label value:
// 1) `reason` label value
// 2) `shoot` label value
// 3) `state` label value
func getGardenerClusterStateMetricRegex(runtimeID string) *regexp.Regexp {
	regexString := fmt.Sprintf("infrastructure_manager_im_gardener_clusters_state.*reason=\"(.*?)\",runtimeId=\"%v\",shoot=\"(.*?)\",state=\"(.*?)\"", runtimeID)
	return regexp.MustCompile(regexString)
}

func getMetricsBody() (string, error) {
	clnt := &http.Client{}
	request, err := http.NewRequestWithContext(suiteCtx, http.MethodGet, "http://localhost:8080/metrics", nil)
	if err != nil {
		return "", fmt.Errorf("request to metrics endpoint :%w", err)
	}
	response, err := clnt.Do(request)
	if err != nil {
		return "", fmt.Errorf("response from metrics endpoint :%w", err)
	}
	defer response.Body.Close()
	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("response body:%w", err)
	}
	bodyString := string(bodyBytes)

	return bodyString, nil
}

func fixNewSecret(name, namespace, kymaName, shootName, data string, lastSyncTime string) corev1.Secret {
	labels := fixSecretLabels(kymaName, shootName)
	annotations := map[string]string{lastKubeconfigSyncAnnotation: lastSyncTime}

	builder := newTestSecret(name, namespace)
	return builder.WithLabels(labels).WithAnnotations(annotations).WithData(data).ToSecret()
}

func (sb *TestSecret) WithAnnotations(annotations map[string]string) *TestSecret {
	sb.secret.Annotations = annotations

	return sb
}

func (sb *TestSecret) WithLabels(labels map[string]string) *TestSecret {
	sb.secret.Labels = labels

	return sb
}

func (sb *TestSecret) WithData(data string) *TestSecret {
	sb.secret.Data = map[string][]byte{"config": []byte(data)}

	return sb
}

func (sb *TestSecret) ToSecret() corev1.Secret {
	return sb.secret
}

func newTestSecret(name, namespace string) *TestSecret {
	return &TestSecret{
		secret: corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		},
	}
}

type TestSecret struct {
	secret corev1.Secret
}

func fixSecretLabels(kymaName, shootName string) map[string]string {
	labels := fixGardenerClusterLabels(kymaName, shootName)
	labels["operator.kyma-project.io/managed-by"] = "infrastructure-manager"
	labels["operator.kyma-project.io/cluster-name"] = kymaName
	return labels
}

func fixGardenerClusterCR(kymaName, namespace, shootName, secretName string) imv1.GardenerCluster {
	return newTestGardenerClusterCR(kymaName, namespace, shootName, secretName).
		WithLabels(fixGardenerClusterLabels(kymaName, shootName)).
		ToCluster()
}

func fixGardenerClusterCRWithForceRotationAnnotation(kymaName, namespace, shootName, secretName string) imv1.GardenerCluster {
	annotations := map[string]string{forceKubeconfigRotationAnnotation: "true"}

	return newTestGardenerClusterCR(kymaName, namespace, shootName, secretName).
		WithLabels(fixGardenerClusterLabels(kymaName, shootName)).
		WithAnnotations(annotations).
		ToCluster()
}

func newTestGardenerClusterCR(name, namespace, shootName, secretName string) *TestGardenerClusterCR {
	return &TestGardenerClusterCR{
		gardenerCluster: imv1.GardenerCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: imv1.GardenerClusterSpec{
				Shoot: imv1.Shoot{
					Name: shootName,
				},
				Kubeconfig: imv1.Kubeconfig{
					Secret: imv1.Secret{
						Name:      secretName,
						Namespace: namespace,
						Key:       "config", //nolint:godox TODO: fill it up with the actual data
					},
				},
			},
		},
	}
}

func (sb *TestGardenerClusterCR) WithLabels(labels map[string]string) *TestGardenerClusterCR {
	sb.gardenerCluster.Labels = labels

	return sb
}

func (sb *TestGardenerClusterCR) WithAnnotations(annotations map[string]string) *TestGardenerClusterCR {
	sb.gardenerCluster.Annotations = annotations

	return sb
}

func (sb *TestGardenerClusterCR) ToCluster() imv1.GardenerCluster {
	return sb.gardenerCluster
}

type TestGardenerClusterCR struct {
	gardenerCluster imv1.GardenerCluster
}

func fixGardenerClusterLabels(kymaName, shootName string) map[string]string {
	labels := map[string]string{}

	labels["kyma-project.io/instance-id"] = "instanceID"
	labels["kyma-project.io/runtime-id"] = kymaName
	labels["kyma-project.io/broker-plan-id"] = "planID"
	labels["kyma-project.io/broker-plan-name"] = "planName"
	labels["kyma-project.io/global-account-id"] = "globalAccountID"
	labels["kyma-project.io/subaccount-id"] = "subAccountID"
	labels["kyma-project.io/shoot-name"] = shootName
	labels["kyma-project.io/region"] = "region"
	labels["operator.kyma-project.io/kyma-name"] = kymaName

	return labels
}
