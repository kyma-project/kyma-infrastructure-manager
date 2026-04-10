package configreload

import (
	"context"
	"time"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/reconciler"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	certificatesv1beta1 "k8s.io/api/certificates/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("ConfigReloadWatcher", func() {
	const (
		timeout  = 30 * time.Second
		interval = 1 * time.Second
	)

	ctx := context.Background()

	createRuntime := func(name string) *imv1.Runtime {
		rt := &imv1.Runtime{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: suiteNamespace,
			},
			Spec: imv1.RuntimeSpec{
				Shoot: imv1.RuntimeShoot{
					Name: "shoot-" + name,
					Provider: imv1.Provider{
						Type: "aws",
						Workers: []gardener.Worker{
							{Name: "worker-1"},
						},
					},
				},
				Security: imv1.Security{Administrators: []string{"admin@test.com"}},
			}}
		ExpectWithOffset(1, k8sClient.Create(ctx, rt)).To(Succeed())
		return rt
	}

	createConfigMap := func(name string) *corev1.ConfigMap {
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: suiteNamespace,
			},
			Data: map[string]string{"key": "initial"},
		}
		ExpectWithOffset(1, k8sClient.Create(ctx, cm)).To(Succeed())
		return cm
	}

	createSecret := func(name string) *corev1.Secret {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: suiteNamespace,
			},
			Data: map[string][]byte{"key": []byte("initial")},
		}
		ExpectWithOffset(1, k8sClient.Create(ctx, secret)).To(Succeed())
		return secret
	}

	updateConfigMap := func(cm *corev1.ConfigMap, value string) {
		cm.Data["key"] = value
		ExpectWithOffset(1, k8sClient.Update(ctx, cm)).To(Succeed())
	}

	updateSecret := func(s *corev1.Secret, value string) {
		s.Data["key"] = []byte(value)
		ExpectWithOffset(1, k8sClient.Update(ctx, s)).To(Succeed())
	}

	hasForceReconcileAnnotation := func(name string) func() bool {
		return func() bool {
			var rt imv1.Runtime
			if err := k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: suiteNamespace}, &rt); err != nil {
				return false
			}
			return rt.Annotations != nil && rt.Annotations[reconciler.ForceReconcileAnnotation] == "true"
		}
	}

	clearAnnotation := func(name string) {
		var rt imv1.Runtime
		ExpectWithOffset(1, k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: suiteNamespace}, &rt)).To(Succeed())
		if rt.Annotations != nil {
			delete(rt.Annotations, reconciler.ForceReconcileAnnotation)
			ExpectWithOffset(1, k8sClient.Update(ctx, &rt)).To(Succeed())
		}
	}

	clearAllAnnotations := func(names ...string) {
		for _, name := range names {
			clearAnnotation(name)
		}
	}

	Context("ACL ConfigMap", func() {
		var (
			rt1 *imv1.Runtime
			rt2 *imv1.Runtime
			cm  *corev1.ConfigMap
		)

		BeforeEach(func() {
			rt1 = createRuntime("runtime-acl-1")
			rt2 = createRuntime("runtime-acl-2")
			cm = createConfigMap(aclConfigMapName)
		})

		It("Should trigger reconcile on all matching Runtimes when ACL ConfigMap is updated", func() {
			updateConfigMap(cm, "updated-acl")

			Eventually(hasForceReconcileAnnotation(rt1.Name), timeout, interval).Should(BeTrue())
			Eventually(hasForceReconcileAnnotation(rt2.Name), timeout, interval).Should(BeTrue())
		})
	})

	Context("ACL ConfigMap with RuntimePredicate", func() {
		It("Should not annotate excluded Runtimes", func() {
			createRuntime("runtime-excluded")
			included := createRuntime("runtime-acl-pred")
			cm := createConfigMap(aclConfigMapName + "-pred")

			var existing corev1.ConfigMap
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: aclConfigMapName, Namespace: suiteNamespace}, &existing)).To(Succeed())
			clearAnnotation(included.Name)

			updateConfigMap(&existing, "trigger-predicate-filter")

			Eventually(hasForceReconcileAnnotation(included.Name), timeout, interval).Should(BeTrue())
			Consistently(hasForceReconcileAnnotation("runtime-excluded"), 5*time.Second, interval).Should(BeFalse())

			_ = cm
		})
	})

	Context("Runtime Bootstrapper - PullSecret", func() {
		It("Should trigger reconcile on all Runtimes when PullSecret is updated", func() {
			rt := createRuntime("runtime-pullsecret-1")
			secret := createSecret(pullSecretName)

			updateSecret(secret, "updated-pull-secret")

			Eventually(hasForceReconcileAnnotation(rt.Name), timeout, interval).Should(BeTrue())
		})
	})

	Context("Runtime Bootstrapper - Manifests ConfigMap", func() {
		It("Should trigger reconcile on all Runtimes when Manifests ConfigMap is updated", func() {
			rt := createRuntime("runtime-manifests-1")
			cm := createConfigMap(manifestsConfigName)

			updateConfigMap(cm, "updated-manifests")

			Eventually(hasForceReconcileAnnotation(rt.Name), timeout, interval).Should(BeTrue())
		})
	})

	Context("Runtime Bootstrapper - KcpConfig ConfigMap", func() {
		It("Should trigger reconcile on all Runtimes when KcpConfig ConfigMap is updated", func() {
			rt := createRuntime("runtime-kcpconfig-1")
			cm := createConfigMap(kcpConfigName)

			updateConfigMap(cm, "updated-kcp-config")

			Eventually(hasForceReconcileAnnotation(rt.Name), timeout, interval).Should(BeTrue())
		})
	})

	Context("Runtime Bootstrapper - ClusterTrustBundle", func() {
		It("Should trigger reconcile on all Runtimes when ClusterTrustBundle is updated", func() {
			rt := createRuntime("runtime-ctb-1")

			trustBundle1 := `-----BEGIN CERTIFICATE-----
MIIBdDCCARmgAwIBAgIUB6E1xfwjup5Jooc3M7QE8n+2GNcwCgYIKoZIzj0EAwIw
DzENMAsGA1UEAwwEdGVzdDAeFw0yNjA0MTAwODUwMDRaFw0yNjA0MTEwODUwMDRa
MA8xDTALBgNVBAMMBHRlc3QwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAAQ8SIyt
kKG+amzWKlddXKgqXWVPTU8tziXx21x9phyu0WbMNpDM5savJ9lILgYQ6lc6Jz8J
a1a3EbkqGFM1C6o0o1MwUTAdBgNVHQ4EFgQUqLQcRmDTzc8JITEEkjnX8jaGu3Iw
HwYDVR0jBBgwFoAUqLQcRmDTzc8JITEEkjnX8jaGu3IwDwYDVR0TAQH/BAUwAwEB
/zAKBggqhkjOPQQDAgNJADBGAiEA7UuMEgHFjzK8/6QGJkOmXXGzjWSFDhO7cShb
98h/wkcCIQCfsKvaf3RwrGQMqkCc9bRuy2WyMWNmAnJfPSwSZBfKew==
-----END CERTIFICATE-----
`
			trustBundle2 := `-----BEGIN CERTIFICATE-----
MIIBdTCCARugAwIBAgIUZLll8ViU9lCmukjCT80M4uPaSvAwCgYIKoZIzj0EAwIw
EDEOMAwGA1UEAwwFdGVzdDIwHhcNMjYwNDEwMDg1MDEwWhcNMjYwNDExMDg1MDEw
WjAQMQ4wDAYDVQQDDAV0ZXN0MjBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABHxp
5WqKXidDGD4kJtF9VrLDCh+R2hghKi83o+BQZce3mY6YcFytujOfQGcA/CE8l0G1
bLqjBoFE2lWE6qT//x2jUzBRMB0GA1UdDgQWBBQR+7bVkF/Nf99KnC5rAaYXZB/W
fzAfBgNVHSMEGDAWgBQR+7bVkF/Nf99KnC5rAaYXZB/WfzAPBgNVHRMBAf8EBTAD
AQH/MAoGCCqGSM49BAMCA0gAMEUCIQCvN7V7GPzMYnj12XVG5/QPfkkPTWXFT9nr
t9hFBq7XhgIgMv88RVsLCEsTXNbfrzy5SdlY+i0Udql2XAvC/HT6Au0=
-----END CERTIFICATE-----
`

			ctb := &certificatesv1beta1.ClusterTrustBundle{
				ObjectMeta: metav1.ObjectMeta{
					Name: clusterTrustBundleName,
				},
				Spec: certificatesv1beta1.ClusterTrustBundleSpec{
					TrustBundle: trustBundle1,
				},
			}
			ExpectWithOffset(1, k8sClient.Create(ctx, ctb)).To(Succeed())

			ctb.Spec.TrustBundle = trustBundle2
			ExpectWithOffset(1, k8sClient.Update(ctx, ctb)).To(Succeed())

			Eventually(hasForceReconcileAnnotation(rt.Name), timeout, interval).Should(BeTrue())
		})
	})

	Context("Unwatched resource", func() {
		It("Should not annotate any Runtime CRs", func() {
			rt := createRuntime("runtime-unwatched")
			unwatched := createConfigMap("unwatched-configmap")

			updateConfigMap(unwatched, "should-not-trigger")

			Consistently(hasForceReconcileAnnotation(rt.Name), 5*time.Second, interval).Should(BeFalse())
		})
	})

	_ = clearAllAnnotations
})
