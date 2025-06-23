package fsm

import (
	"context"
	"fmt"
	"time"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/controller/metrics/mocks"
	imv1_client "github.com/kyma-project/infrastructure-manager/internal/controller/runtime/fsm/client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe(`runtime_fsm_apply_crb`, Label("applyCRB"), func() {

	var testErr = fmt.Errorf("test error")

	withMockedMetrics := func() fakeFSMOpt {
		m := &mocks.Metrics{}
		m.On("SetRuntimeStates", mock.Anything).Return()
		m.On("CleanUpRuntimeGauge", mock.Anything, mock.Anything).Return()
		m.On("IncRuntimeFSMStopCounter").Return()
		return withMetrics(m)
	}

	DescribeTable("getMissing",
		func(tc tcCRBData) {
			actual := getMissing(tc.crbs, tc.admins)
			Expect(actual).To(BeComparableTo(tc.expected))
		},
		Entry("should return a list with CRBs to be created", tcCRBData{
			admins: []string{"test1", "test2"},
			crbs:   nil,
			expected: []rbacv1.ClusterRoleBinding{
				toAdminClusterRoleBinding("test1"),
				toAdminClusterRoleBinding("test2"),
			},
		}),
		Entry("should return nil list if no admins missing", tcCRBData{
			admins: []string{"test1", "test2", "test3"},
			crbs: []rbacv1.ClusterRoleBinding{
				toAdminClusterRoleBinding("test1"),
				toManagedClusterRoleBinding("test2", "infrastructure-manager"),
				toManagedClusterRoleBinding("test3", "infrastructure-manager"),
			},
			expected: nil,
		}),
		Entry("should return nil list if no admins missing", tcCRBData{
			admins: []string{"test1"},
			crbs: []rbacv1.ClusterRoleBinding{
				toAdminClusterRoleBinding("test1"),
			},
			expected: nil,
		}),
	)

	DescribeTable("getRemoved",
		func(tc tcCRBData) {
			actual := getRemoved(tc.crbs, tc.admins)
			Expect(actual).To(BeComparableTo(tc.expected))
		},
		Entry("should return nil list if CRB list is nil", tcCRBData{
			admins:   []string{"test1"},
			crbs:     nil,
			expected: nil,
		}),
		Entry("should return nil list if CRB list is empty", tcCRBData{
			admins:   []string{"test1"},
			crbs:     []rbacv1.ClusterRoleBinding{},
			expected: nil,
		}),
		Entry("should return nil list if no admins to remove", tcCRBData{
			admins:   []string{"test1"},
			crbs:     []rbacv1.ClusterRoleBinding{toAdminClusterRoleBinding("test1")},
			expected: nil,
		}),
		Entry("should return list if with CRBs to remove", tcCRBData{
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
		Entry("should not remove CRB managed by reconciler", tcCRBData{
			admins: []string{"test1", "test2"},
			crbs: []rbacv1.ClusterRoleBinding{
				toManagedClusterRoleBinding("test1", "reconciler"),
				toAdminClusterRoleBinding("test3"),
			},
			expected: []rbacv1.ClusterRoleBinding{
				toAdminClusterRoleBinding("test3"),
			},
		}),
		Entry("should not remove CRB not managed by reconciler or KIM", tcCRBData{
			admins: []string{"test1", "test2"},
			crbs: []rbacv1.ClusterRoleBinding{
				toManagedClusterRoleBinding("test3", "should-stay"),
				toManagedClusterRoleBinding("test4", ""),
			},
			expected: nil,
		}),
		Entry("should not remove Service account CRB not managed by reconciler or KIM", tcCRBData{
			admins: []string{"test1", "test2"},
			crbs: []rbacv1.ClusterRoleBinding{
				toServiceAccountClusterRoleBinding("test3-should-stay"),
				toServiceAccountClusterRoleBinding("test4-should-stay"),
			},
			expected: nil,
		}),
		Entry("should remove CRB managed by reconciler or KIM, that are not in the admin list", tcCRBData{
			admins: []string{"test4", "test5"},
			crbs: []rbacv1.ClusterRoleBinding{
				toManagedClusterRoleBinding("test1", "infrastructure-manager"),
				toManagedClusterRoleBinding("test2", "reconciler"),
				toManagedClusterRoleBinding("test3", "should-stay"),
				toManagedClusterRoleBinding("test4", "infrastructure-manager"),
				toManagedClusterRoleBinding("test5", "reconciler"),
			},
			expected: []rbacv1.ClusterRoleBinding{
				toManagedClusterRoleBinding("test1", "infrastructure-manager"),
			},
		}),
		Entry("should not remove admins with random label", tcCRBData{
			admins: []string{"test1", "test2"},
			crbs: []rbacv1.ClusterRoleBinding{
				toAdminClusterRoleBindingWithLabel("test3-should-stay", "test", "me"),
			},
			expected: nil,
		}),
		Entry("should not remove admins managed by others", tcCRBData{
			admins: []string{"test1", "test2"},
			crbs: []rbacv1.ClusterRoleBinding{
				toAdminClusterRoleBindingWithLabel("test4-should-stay", "reconciler.kyma-project.io/managed-by", "others"),
			},
			expected: nil,
		}),
		Entry("should not remove admins without labels", tcCRBData{
			admins: []string{"test1", "test2"},
			crbs: []rbacv1.ClusterRoleBinding{
				toAdminClusterRoleBindingNoLabels("test4-should-stay"),
			},
			expected: nil,
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
		imv1_client.GetShootClient = func(
			_ context.Context,
			_ client.Client,
			_ imv1.Runtime) (client.Client, error) {
			return f.ShootClient, nil
		}
		return nil
	}

	DescribeTable("sFnApplyClusterRoleBindings",
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
				result: ctrl.Result{RequeueAfter: 0},
			},
			fsm: must(
				newFakeFSM,
				withFakedK8sClient(testScheme, &testRuntimeWithAdmin),
				withFn(sFnApplyClusterRoleBindingsStateSetup),
				withFakeEventRecorder(1),
				withMockedMetrics(),
				withDefaultReconcileDuration(),
			),
			setup: defaultSetup,
		}),

		Entry("nothing change", tcApplySfn{
			instance: testRuntime,
			expected: tcSfnExpected{
				err:    nil,
				result: ctrl.Result{RequeueAfter: 0},
			},
			fsm: must(
				newFakeFSM,
				withFakedK8sClient(testScheme, &testRuntime),
				withFn(sFnApplyClusterRoleBindingsStateSetup),
				withFakeEventRecorder(1),
				withMockedMetrics(),
				withDefaultReconcileDuration(),
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
				withFakedK8sClient(testScheme, &testRuntime),
				withFn(sFnApplyClusterRoleBindingsStateSetup),
				withFakeEventRecorder(1),
				withMockedMetrics(),
				withDefaultReconcileDuration(),
			),
			setup: func(f *fsm) error {
				imv1_client.GetShootClient = func(
					_ context.Context,
					_ client.Client,
					_ imv1.Runtime) (client.Client, error) {
					return nil, testErr
				}
				return nil

			},
		}),
	)
})

type tcCRBData struct {
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

func toManagedClusterRoleBinding(name, managedBy string) rbacv1.ClusterRoleBinding {
	return toAdminClusterRoleBindingWithLabel(name,
		"reconciler.kyma-project.io/managed-by", managedBy)
}

func toServiceAccountClusterRoleBinding(name string) rbacv1.ClusterRoleBinding {
	return rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Subjects: []rbacv1.Subject{{
			Kind:      rbacv1.ServiceAccountKind,
			Name:      "cluster-admin",
			Namespace: "cicdnamespace",
			APIGroup:  rbacv1.GroupName,
		}},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     "cluster-admin",
		},
	}
}
