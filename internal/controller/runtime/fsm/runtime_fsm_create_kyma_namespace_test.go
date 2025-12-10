package fsm

import (
	"context"
	"testing"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	metrics_mocks "github.com/kyma-project/infrastructure-manager/internal/controller/metrics/mocks"
	fsm_mocks "github.com/kyma-project/infrastructure-manager/internal/controller/runtime/fsm/mocks"
	fsm_testing "github.com/kyma-project/infrastructure-manager/internal/controller/runtime/fsm/testing"
	errs "github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	core_v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	api "k8s.io/apimachinery/pkg/runtime"
	util "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestNamespaceCreateState(t *testing.T) {
	t.Run("Should create kyma-system namespace and proceed to next state", func(t *testing.T) {
		// given
		ctx := context.Background()
		scheme := createNamespaceTestScheme()
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

		runtimeClientGetter := &fsm_mocks.RuntimeClientGetter{}
		runtimeClientGetter.On("Get", mock.Anything, mock.Anything).Return(fakeClient, nil)

		testFsm := &fsm{K8s: K8s{
			RuntimeClientGetter: runtimeClientGetter,
		}}

		systemState := &systemState{
			instance: runtimeForTest(),
		}

		expectedRuntimeConditions := []metav1.Condition{
			{
				Type:    string(imv1.ConditionTypeKymaSystemCreated),
				Reason:  string(imv1.ConditionReasonKymaSystemNSReady),
				Status:  "True",
				Message: "Creation of kyma-system Namespace",
			},
		}

		// when
		stateFn, _, err := sFnCreateKymaNamespace(ctx, testFsm, systemState)

		// then
		require.NoError(t, err)
		require.Contains(t, stateFn.name(), "sFnInitializeRuntimeBootstrapper")

		var kymaSystemNs core_v1.Namespace
		nsKey := client.ObjectKey{
			Name: "kyma-system",
		}
		err = fakeClient.Get(ctx, nsKey, &kymaSystemNs)
		assert.NoError(t, err)
		assert.Equal(t, "kyma-system", kymaSystemNs.Name)
		assertEqualConditions(t, expectedRuntimeConditions, systemState.instance.Status.Conditions)
	})

	t.Run("Should not error when kyma-system namespace already exists", func(t *testing.T) {
		// given
		ctx := context.Background()
		scheme := createNamespaceTestScheme()
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

		// create namespace beforehand
		kymaSystemNs := &core_v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "kyma-system",
			},
		}
		err := fakeClient.Create(ctx, kymaSystemNs)
		require.NoError(t, err)

		runtimeClientGetter := &fsm_mocks.RuntimeClientGetter{}
		runtimeClientGetter.On("Get", mock.Anything, mock.Anything).Return(fakeClient, nil)

		testFsm := &fsm{K8s: K8s{
			RuntimeClientGetter: runtimeClientGetter,
		}}

		systemState := &systemState{
			instance: runtimeForTest(),
		}

		expectedRuntimeConditions := []metav1.Condition{
			{
				Type:    string(imv1.ConditionTypeKymaSystemCreated),
				Reason:  string(imv1.ConditionReasonKymaSystemNSReady),
				Status:  "True",
				Message: "Creation of kyma-system Namespace",
			},
		}

		// when
		stateFn, _, err := sFnCreateKymaNamespace(ctx, testFsm, systemState)

		// then
		require.NoError(t, err)
		require.Contains(t, stateFn.name(), "sFnInitializeRuntimeBootstrapper")
		assertEqualConditions(t, expectedRuntimeConditions, systemState.instance.Status.Conditions)
	})

	t.Run("Should handle runtime client error and update status", func(t *testing.T) {
		// given
		ctx := context.Background()
		runtimeClientError := errs.New("some client error")
		runtimeClientGetter := &fsm_mocks.RuntimeClientGetter{}
		runtimeClientGetter.On("Get", mock.Anything, mock.Anything).Return(nil, runtimeClientError)

		m := &metrics_mocks.Metrics{}
		m.On("IncRuntimeFSMStopCounter").Return()

		testFsm := &fsm{
			K8s: K8s{RuntimeClientGetter: runtimeClientGetter},
			RCCfg: RCCfg{
				Metrics: m,
			},
		}

		systemState := &systemState{
			instance: runtimeForTest(),
		}

		expectedRuntimeConditions := []metav1.Condition{
			{
				Type:    string(imv1.ConditionTypeKymaSystemCreated),
				Reason:  string(imv1.ConditionReasonKymaSystemNSError),
				Status:  "False",
				Message: runtimeClientError.Error(),
			},
		}

		// when
		stateFn, _, err := sFnCreateKymaNamespace(ctx, testFsm, systemState)

		// then
		require.NoError(t, err)
		require.Contains(t, stateFn.name(), "sFnUpdateStatus")
		assertEqualConditions(t, expectedRuntimeConditions, systemState.instance.Status.Conditions)
	})

	t.Run("Should handle namespace creation error and update status", func(t *testing.T) {
		// given
		ctx := context.Background()
		scheme := createNamespaceTestScheme()
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithInterceptorFuncs(interceptor.Funcs{
				Create: fsm_testing.GetFakeInterceptorThatThrowsErrorOnNSCreation(),
			}).
			Build()

		runtimeClientGetter := &fsm_mocks.RuntimeClientGetter{}
		runtimeClientGetter.On("Get", mock.Anything, mock.Anything).Return(fakeClient, nil)

		m := &metrics_mocks.Metrics{}
		m.On("IncRuntimeFSMStopCounter").Return()

		testFsm := &fsm{
			K8s: K8s{RuntimeClientGetter: runtimeClientGetter},
			RCCfg: RCCfg{
				Metrics: m,
			},
		}

		systemState := &systemState{
			instance: runtimeForTest(),
		}

		// when
		stateFn, _, err := sFnCreateKymaNamespace(ctx, testFsm, systemState)

		// then
		require.NoError(t, err)
		require.Contains(t, stateFn.name(), "sFnUpdateStatus")

		var kymaSystemNs core_v1.Namespace
		nsKey := client.ObjectKey{
			Name: "kyma-system",
		}
		err = fakeClient.Get(ctx, nsKey, &kymaSystemNs)
		assert.True(t, errors.IsNotFound(err))

		expectedRuntimeConditions := []metav1.Condition{
			{
				Type:    string(imv1.ConditionTypeKymaSystemCreated),
				Reason:  string(imv1.ConditionReasonKymaSystemNSError),
				Status:  "False",
				Message: "simulated error to for tests that expect an error when creating a namespace",
			},
		}
		assertEqualConditions(t, expectedRuntimeConditions, systemState.instance.Status.Conditions)
	})
}

func createNamespaceTestScheme() *api.Scheme {
	testScheme := api.NewScheme()
	util.Must(imv1.AddToScheme(testScheme))
	util.Must(core_v1.AddToScheme(testScheme))
	return testScheme
}
