package main

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

var _ = Describe("Envtest", func() {
	ctx := context.Background()
	testenv := &envtest.Environment{}
	cfg, err := testenv.Start()
	Expect(err).ToNot(HaveOccurred())

	client, err := kubernetes.NewForConfig(cfg)
	Expect(err).ToNot(HaveOccurred())
	crbClient := client.RbacV1().ClusterRoleBindings()
	fetcher := NewCRBFetcher(crbClient, "old=true", "new=true")
	cleaner := NewCRBCleaner(crbClient)

	BeforeEach(func() {
		new, err := fetcher.FetchNew(ctx)
		Expect(err).ToNot(HaveOccurred())

		old, err := fetcher.FetchOld(ctx)
		Expect(err).ToNot(HaveOccurred())

		cleaner.Clean(ctx, append(new, old...))
	})

	It("removes old CRBs", func() {
		By("Generate test data")
		old, new := generateCRBs(5)

		for _, crb := range append(old, new...) {
			_, err := crbClient.Create(ctx, crb, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred(), "Failed to create CRB %q", crb.Name)
		}

		By("Processing CRBs")
		failures, err := ProcessCRBs(fetcher, cleaner, nil, Config{
			DryRun: false,
			Force:  false,
		})

		Expect(err).ToNot(HaveOccurred())
		Expect(failures).To(BeEmpty())

		Eventually(func() ([]rbacv1.ClusterRoleBinding, error) {
			return fetcher.FetchOld(ctx)
		}).Should(BeEmpty())
	})

	It("skips removal when mismatch is found", func() {
		By("Generate test data")
		old, new := generateCRBs(5)

		for _, crb := range append(old, new[3:]...) {
			_, err := crbClient.Create(ctx, crb, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred(), "Failed to create CRB %q", crb.Name)
		}

		By("Processing CRBs")
		failures, err := ProcessCRBs(fetcher, cleaner, nil, Config{
			DryRun: false,
			Force:  false,
		})

		Expect(err).ToNot(HaveOccurred())
		Expect(failures).To(BeEmpty())
		Consistently(func() ([]rbacv1.ClusterRoleBinding, error) {
			return fetcher.FetchOld(ctx)
		}).Should(HaveLen(5))
	})

	It("removes despite mismatch, with -force", func() {
		By("Generate test data")
		old, new := generateCRBs(5)

		for _, crb := range append(old, new[3:]...) {
			_, err := crbClient.Create(ctx, crb, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred(), "Failed to create CRB %q", crb.Name)
		}

		By("Processing CRBs")
		failures, err := ProcessCRBs(fetcher, cleaner, nil, Config{
			DryRun: false,
			Force:  true,
		})

		Expect(err).ToNot(HaveOccurred())
		Expect(failures).To(BeEmpty())
		Eventually(func() ([]rbacv1.ClusterRoleBinding, error) {
			return fetcher.FetchOld(ctx)
		}).Should(BeEmpty())
	})
})

func generateCRBs(count int) ([]*rbacv1.ClusterRoleBinding, []*rbacv1.ClusterRoleBinding) {
	old, new := make([]*rbacv1.ClusterRoleBinding, count), make([]*rbacv1.ClusterRoleBinding, count)
	for i := 0; i < count; i++ {
		old[i] = ClusterRoleBinding(fmt.Sprintf("old%2d", i), fmt.Sprintf("user%d@sap.com", i), fmt.Sprintf("role%2d", i), "old", "true")
		new[i] = ClusterRoleBinding(fmt.Sprintf("new%2d", i), fmt.Sprintf("user%d@sap.com", i), fmt.Sprintf("role%2d", i), "new", "true")
	}
	return old, new
}

func ClusterRoleBinding(name, user, role string, labels ...string) *rbacv1.ClusterRoleBinding {
	labelsMap := map[string]string{}
	for i := 0; i < len(labels); i += 2 {
		labelsMap[labels[i]] = labels[i+1]
	}
	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labelsMap,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: "User",
				Name: user,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     role,
		},
	}
}
