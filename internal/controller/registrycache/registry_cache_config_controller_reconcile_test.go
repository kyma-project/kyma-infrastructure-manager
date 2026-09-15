package registrycache

import (
	"context"
	"errors"
	"testing"
	"time"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	kyma "github.com/kyma-project/lifecycle-manager/api/v1beta2"
	registrycache "github.com/kyma-project/registry-cache/api/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	util "k8s.io/apimachinery/pkg/util/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func newTestScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	util.Must(imv1.AddToScheme(scheme))
	util.Must(corev1.AddToScheme(scheme))
	util.Must(kyma.AddToScheme(scheme))
	util.Must(registrycache.AddToScheme(scheme))
	util.Must(apiextensionsv1.AddToScheme(scheme))
	return scheme
}

func newTestReconciler(kcpClient client.Client, runtimeClientGetter RuntimeClientGetter) *RegistryCacheConfigReconciler {
	logger := zap.New(zap.UseDevMode(true))
	return &RegistryCacheConfigReconciler{
		KcpClient:           kcpClient,
		Log:                 logger,
		RuntimeClientGetter: runtimeClientGetter,
		KcpNamespace:        "kcp-system",
		ReconcilePeriod:     1 * time.Minute,
	}
}

func newKIMSecret(name, runtimeID string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "kcp-system",
			Labels: map[string]string{
				RuntimeIDLabel:                        runtimeID,
				"operator.kyma-project.io/managed-by": "infrastructure-manager",
			},
		},
	}
}

func newRuntime(name string) *imv1.Runtime {
	return &imv1.Runtime{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "kcp-system",
		},
		Spec: imv1.RuntimeSpec{
			Shoot: imv1.RuntimeShoot{
				Name: name + "-shoot",
			},
		},
	}
}

func reconcileRequest(secretName string) ctrl.Request {
	return ctrl.Request{NamespacedName: types.NamespacedName{Name: secretName, Namespace: "kcp-system"}}
}

// TestReconcile_RuntimeBeingDeleted_DeletionTimestamp verifies that reconciliation
// is skipped when the Runtime has a non-zero DeletionTimestamp.
func TestReconcile_RuntimeBeingDeleted_DeletionTimestamp(t *testing.T) {
	scheme := newTestScheme()
	now := metav1.Now()
	rt := newRuntime("runtime-deleting")
	rt.DeletionTimestamp = &now
	rt.Finalizers = []string{"test-finalizer"}

	secret := newKIMSecret("secret-deleting", "runtime-deleting")

	kcpClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(rt, secret).Build()
	reconciler := newTestReconciler(kcpClient, func(_ corev1.Secret) (client.Client, error) {
		return nil, errors.New("should not be called")
	})

	result, err := reconciler.Reconcile(context.Background(), reconcileRequest("secret-deleting"))

	require.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
}

// TestReconcile_RuntimeBeingDeleted_TerminatingState verifies that reconciliation
// is skipped when the Runtime status is Terminating.
func TestReconcile_RuntimeBeingDeleted_TerminatingState(t *testing.T) {
	scheme := newTestScheme()
	rt := newRuntime("runtime-terminating")
	rt.Status.State = imv1.RuntimeStateTerminating

	secret := newKIMSecret("secret-terminating", "runtime-terminating")

	kcpClient := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(rt).WithObjects(rt, secret).Build()
	// Update the status separately since fake client requires status subresource
	require.NoError(t, kcpClient.Status().Update(context.Background(), rt))

	reconciler := newTestReconciler(kcpClient, func(_ corev1.Secret) (client.Client, error) {
		return nil, errors.New("should not be called")
	})

	result, err := reconciler.Reconcile(context.Background(), reconcileRequest("secret-terminating"))

	require.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
}

// TestReconcile_SecretNotFound verifies that reconciliation stops gracefully when
// the Secret is not found.
func TestReconcile_SecretNotFound(t *testing.T) {
	scheme := newTestScheme()
	kcpClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	reconciler := newTestReconciler(kcpClient, nil)

	result, err := reconciler.Reconcile(context.Background(), reconcileRequest("non-existent-secret"))

	require.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
}

// TestReconcile_SecretGetError verifies that reconciliation returns an error when
// the KCP client fails to get the Secret with a non-NotFound error.
func TestReconcile_SecretGetError(t *testing.T) {
	scheme := newTestScheme()
	getErr := apierrors.NewInternalError(errors.New("kcp unavailable"))
	kcpClient := fake.NewClientBuilder().WithScheme(scheme).WithInterceptorFuncs(interceptor.Funcs{
		Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
			if _, ok := obj.(*corev1.Secret); ok {
				return getErr
			}
			return c.Get(ctx, key, obj, opts...)
		},
	}).Build()

	reconciler := newTestReconciler(kcpClient, nil)

	result, err := reconciler.Reconcile(context.Background(), reconcileRequest("any-secret"))

	require.Error(t, err)
	assert.Equal(t, ctrl.Result{}, result)
}

// TestReconcile_RuntimeNotFound verifies that reconciliation stops gracefully when
// the Runtime is not found.
func TestReconcile_RuntimeNotFound(t *testing.T) {
	scheme := newTestScheme()
	secret := newKIMSecret("secret-no-runtime", "runtime-missing")

	kcpClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build()
	reconciler := newTestReconciler(kcpClient, nil)

	result, err := reconciler.Reconcile(context.Background(), reconcileRequest("secret-no-runtime"))

	require.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
}

// TestApplyRegistryCacheConfig_SecondGetFails verifies that applyRegistryCacheConfig
// returns an error when the second KcpClient.Get for the Runtime fails.
func TestApplyRegistryCacheConfig_SecondGetFails(t *testing.T) {
	scheme := newTestScheme()

	rt := newRuntime("runtime-second-get-fail")
	secret := newKIMSecret("secret-second-get-fail", "runtime-second-get-fail")

	getCallCount := 0
	internalErr := apierrors.NewInternalError(errors.New("server error"))
	kcpClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(rt, secret).WithInterceptorFuncs(interceptor.Funcs{
		Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
			if _, ok := obj.(*imv1.Runtime); ok {
				getCallCount++
				if getCallCount == 2 {
					return internalErr
				}
			}
			return c.Get(ctx, key, obj, opts...)
		},
	}).Build()

	runtimeClient := fake.NewClientBuilder().WithScheme(newRuntimeClientScheme()).Build()
	reconciler := newTestReconciler(kcpClient, func(_ corev1.Secret) (client.Client, error) {
		return runtimeClient, nil
	})

	result, err := reconciler.Reconcile(context.Background(), reconcileRequest("secret-second-get-fail"))

	require.Error(t, err)
	assert.Equal(t, ctrl.Result{}, result)
}

// TestApplyRegistryCacheConfig_UpdateFails verifies that applyRegistryCacheConfig
// propagates an error returned by KcpClient.Update.
func TestApplyRegistryCacheConfig_UpdateFails(t *testing.T) {
	scheme := newTestScheme()

	rt := newRuntime("runtime-update-fail")
	rt.Spec.Caching = []imv1.ImageRegistryCache{
		{Name: "old-config", Namespace: "test", Config: registrycache.RegistryCacheConfigSpec{Upstream: "old.io"}},
	}
	secret := newKIMSecret("secret-update-fail", "runtime-update-fail")

	updateErr := apierrors.NewInternalError(errors.New("update conflict"))
	kcpClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(rt, secret).WithInterceptorFuncs(interceptor.Funcs{
		Update: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
			if _, ok := obj.(*imv1.Runtime); ok {
				return updateErr
			}
			return c.Update(ctx, obj, opts...)
		},
	}).Build()

	// Registry cache with different upstream to trigger an update
	runtimeClient := fake.NewClientBuilder().WithScheme(newRuntimeClientScheme()).WithObjects(
		&apiextensionsv1.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{Name: "kymas.operator.kyma-project.io"},
		},
		&kyma.Kyma{
			ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: "kyma-system"},
			Spec:       kyma.KymaSpec{Modules: []kyma.Module{{Name: "registry-cache"}}},
		},
		&registrycache.RegistryCacheConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "new-config", Namespace: "test"},
			Spec:       registrycache.RegistryCacheConfigSpec{Upstream: "new.io"},
		},
	).Build()

	reconciler := newTestReconciler(kcpClient, func(_ corev1.Secret) (client.Client, error) {
		return runtimeClient, nil
	})

	result, err := reconciler.Reconcile(context.Background(), reconcileRequest("secret-update-fail"))

	require.Error(t, err)
	assert.Equal(t, ctrl.Result{}, result)
}

// TestFetchConfigs_CRDGetError verifies that fetchConfigs returns an error when
// the CRD Get call returns a non-NotFound error.
func TestFetchConfigs_CRDGetError(t *testing.T) {
	scheme := newRuntimeClientScheme()

	crdErr := apierrors.NewInternalError(errors.New("CRD store unavailable"))
	runtimeClient := fake.NewClientBuilder().WithScheme(scheme).WithInterceptorFuncs(interceptor.Funcs{
		Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
			if _, ok := obj.(*apiextensionsv1.CustomResourceDefinition); ok {
				return crdErr
			}
			return c.Get(ctx, key, obj, opts...)
		},
	}).Build()

	logger := zap.New(zap.UseDevMode(true))
	result, err := fetchConfigs(context.Background(), logger, runtimeClient)

	require.Error(t, err)
	assert.Nil(t, result)
}

// TestFetchConfigs_KymaGetError verifies that fetchConfigs returns an error when
// the Kyma CR Get call fails.
func TestFetchConfigs_KymaGetError(t *testing.T) {
	scheme := newRuntimeClientScheme()

	kymaErr := apierrors.NewInternalError(errors.New("kyma CR unavailable"))
	runtimeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&apiextensionsv1.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{Name: "kymas.operator.kyma-project.io"},
		},
	).WithInterceptorFuncs(interceptor.Funcs{
		Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
			if _, ok := obj.(*kyma.Kyma); ok {
				return kymaErr
			}
			return c.Get(ctx, key, obj, opts...)
		},
	}).Build()

	logger := zap.New(zap.UseDevMode(true))
	result, err := fetchConfigs(context.Background(), logger, runtimeClient)

	require.Error(t, err)
	assert.Nil(t, result)
}

// TestFetchConfigs_ListError verifies that fetchConfigs returns an error when
// listing RegistryCacheConfig objects fails.
func TestFetchConfigs_ListError(t *testing.T) {
	scheme := newRuntimeClientScheme()

	listErr := apierrors.NewInternalError(errors.New("list failed"))
	runtimeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&apiextensionsv1.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{Name: "kymas.operator.kyma-project.io"},
		},
		&kyma.Kyma{
			ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: "kyma-system"},
			Spec: kyma.KymaSpec{
				Modules: []kyma.Module{{Name: "registry-cache"}},
			},
		},
	).WithInterceptorFuncs(interceptor.Funcs{
		List: func(ctx context.Context, c client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
			if _, ok := list.(*registrycache.RegistryCacheConfigList); ok {
				return listErr
			}
			return c.List(ctx, list, opts...)
		},
	}).Build()

	logger := zap.New(zap.UseDevMode(true))
	result, err := fetchConfigs(context.Background(), logger, runtimeClient)

	require.Error(t, err)
	assert.Nil(t, result)
}

// TestStopIfNotFound_NotFoundError verifies that stopIfNotFound returns no error
// and an empty result when err is a NotFound API error.
func TestStopIfNotFound_NotFoundError(t *testing.T) {
	logger := zap.New(zap.UseDevMode(true))
	notFoundErr := apierrors.NewNotFound(schema.GroupResource{}, "test-resource")

	result, err := stopIfNotFound(notFoundErr, logger, "msg", "not found msg")

	require.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
}

// TestStopIfNotFound_OtherError verifies that stopIfNotFound propagates a non-NotFound error.
func TestStopIfNotFound_OtherError(t *testing.T) {
	logger := zap.New(zap.UseDevMode(true))
	otherErr := apierrors.NewInternalError(errors.New("internal"))

	result, err := stopIfNotFound(otherErr, logger, "msg", "not found msg")

	require.Error(t, err)
	assert.Equal(t, ctrl.Result{}, result)
}

// TestSecretControlledByKIM_NoLabels verifies secretControlledByKIM returns false
// when the secret has no labels.
func TestSecretControlledByKIM_NoLabels(t *testing.T) {
	secret := corev1.Secret{}
	assert.False(t, secretControlledByKIM(secret))
}

// TestSecretControlledByKIM_ManagedByKIM verifies secretControlledByKIM returns true
// when the secret has the expected labels.
func TestSecretControlledByKIM_ManagedByKIM(t *testing.T) {
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				RuntimeIDLabel:                        "some-runtime-id",
				"operator.kyma-project.io/managed-by": "infrastructure-manager",
			},
		},
	}
	assert.True(t, secretControlledByKIM(secret))
}

// TestSecretControlledByKIM_DifferentManager verifies secretControlledByKIM returns false
// when managed-by label is not infrastructure-manager.
func TestSecretControlledByKIM_DifferentManager(t *testing.T) {
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				RuntimeIDLabel:                        "some-runtime-id",
				"operator.kyma-project.io/managed-by": "something-else",
			},
		},
	}
	assert.False(t, secretControlledByKIM(secret))
}

func newRuntimeClientScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	util.Must(registrycache.AddToScheme(scheme))
	util.Must(corev1.AddToScheme(scheme))
	util.Must(kyma.AddToScheme(scheme))
	util.Must(apiextensionsv1.AddToScheme(scheme))
	return scheme
}
