package fsm

import (
	"context"
	"fmt"
	"time"

	gardener_api "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe(`runtime_fsm_apply_crb`, Label("applyCRB"), func() {

	var testErr = fmt.Errorf("test error")

	DescribeTable("getMissing",
		func(tc tcGetCRB) {
			actual := getMissing(tc.crbs, tc.admins)
			Expect(actual).To(BeComparableTo(tc.expected))
		},
		Entry("should return a list with CRBs to be created", tcGetCRB{
			admins: []string{"test1", "test2"},
			crbs:   nil,
			expected: []rbacv1.ClusterRoleBinding{
				toAdminClusterRoleBinding("test1"),
				toAdminClusterRoleBinding("test2"),
			},
		}),
		Entry("should return nil list if no admins missing", tcGetCRB{
			admins: []string{"test1"},
			crbs: []rbacv1.ClusterRoleBinding{
				toAdminClusterRoleBinding("test1"),
			},
			expected: nil,
		}),
	)

	DescribeTable("getRemoved",
		func(tc tcGetCRB) {
			actual := getRemoved(tc.crbs, tc.admins)
			Expect(actual).To(BeComparableTo(tc.expected))
		},
		Entry("should return nil list if CRB list is nil", tcGetCRB{
			admins:   []string{"test1"},
			crbs:     nil,
			expected: nil,
		}),
		Entry("should return nil list if CRB list is empty", tcGetCRB{
			admins:   []string{"test1"},
			crbs:     []rbacv1.ClusterRoleBinding{},
			expected: nil,
		}),
		Entry("should return nil list if no admins to remove", tcGetCRB{
			admins:   []string{"test1"},
			crbs:     []rbacv1.ClusterRoleBinding{toAdminClusterRoleBinding("test1")},
			expected: nil,
		}),
		Entry("should return list if with CRBs to remove", tcGetCRB{
			admins: []string{"test2"},
			crbs: []rbacv1.ClusterRoleBinding{
				toAdminClusterRoleBinding("test1"),
				toAdminClusterRoleBinding("test2"),
				toAdminClusterRoleBinding("test3"),
			},
			expected: []rbacv1.ClusterRoleBinding{
				toAdminClusterRoleBinding("test1"),
				toAdminClusterRoleBinding("test3"),
			},
		}),
	)

	testRuntime := imv1.Runtime{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testme1",
			Namespace: "default",
		},
	}

	testRuntimeWithAdmin := imv1.Runtime{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testme1",
			Namespace: "default",
		},
		Spec: imv1.RuntimeSpec{
			Security: imv1.Security{
				Administrators: []string{
					"test-admin1",
				},
			},
		},
	}

	testScheme, err := newTestScheme()
	Expect(err).ShouldNot(HaveOccurred())

	defaultSetup := func(f *fsm) error {
		GetShootClient = func(
			_ context.Context,
			_ client.SubResourceClient,
			_ *gardener_api.Shoot) (client.Client, error) {
			return f.Client, nil
		}
		return nil
	}

	DescribeTable("sFnAppluClusterRoleBindings",
		func(tc tcApplySfn) {
			// initialize test data if required
			Expect(tc.init()).ShouldNot(HaveOccurred())

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			actualResult, actualErr := tc.fsm.Run(ctx, tc.instance)
			Expect(actualResult).Should(BeComparableTo(tc.expected.result))

			matchErr := BeNil()
			if tc.expected.err != nil {
				matchErr = MatchError(tc.expected.err)
			}
			Expect(actualErr).Should(matchErr)
		},

		Entry("add admin", tcApplySfn{
			instance: testRuntimeWithAdmin,
			expected: tcSfnExpected{
				err:    nil,
				result: ctrl.Result{RequeueAfter: time.Second * 15},
			},
			fsm: must(
				newFakeFSM,
				withAuditLogging(true, nil),
				withFakedK8sClient(testScheme, &testRuntimeWithAdmin),
				withFn(sFnApplyClusterRoleBindingsStateSetup),
				withFakeEventRecorder(1),
			),
			setup: defaultSetup,
		}),

		Entry("nothing change", tcApplySfn{
			instance: testRuntime,
			expected: tcSfnExpected{
				err:    nil,
				result: ctrl.Result{RequeueAfter: time.Second * 15},
			},
			fsm: must(
				newFakeFSM,
				withAuditLogging(true, nil),
				withFakedK8sClient(testScheme, &testRuntime),
				withFn(sFnApplyClusterRoleBindingsStateSetup),
				withFakeEventRecorder(1),
			),
			setup: defaultSetup,
		}),

		Entry("error getting client", tcApplySfn{
			instance: testRuntime,
			expected: tcSfnExpected{
				err: testErr,
			},
			fsm: must(
				newFakeFSM,
				withAuditLogging(true, nil),
				withFakedK8sClient(testScheme, &testRuntime),
				withFn(sFnApplyClusterRoleBindingsStateSetup),
				withFakeEventRecorder(1),
			),
			setup: func(f *fsm) error {
				GetShootClient = func(
					_ context.Context,
					_ client.SubResourceClient,
					_ *gardener_api.Shoot) (client.Client, error) {
					return nil, testErr
				}
				return nil

			},
		}),
	)
})

type tcGetCRB struct {
	crbs     []rbacv1.ClusterRoleBinding
	admins   []string
	expected []rbacv1.ClusterRoleBinding
}

type tcSfnExpected struct {
	result ctrl.Result
	err    error
}

type tcApplySfn struct {
	expected tcSfnExpected
	setup    func(m *fsm) error
	fsm      *fsm
	instance imv1.Runtime
}

func (c *tcApplySfn) init() error {
	if c.setup != nil {
		return c.setup(c.fsm)
	}
	return nil
}

func newTestScheme() (*runtime.Scheme, error) {
	schema := runtime.NewScheme()

	for _, fn := range []func(*runtime.Scheme) error{
		imv1.AddToScheme,
		rbacv1.AddToScheme,
	} {
		if err := fn(schema); err != nil {
			return nil, err
		}
	}
	return schema, nil
}
