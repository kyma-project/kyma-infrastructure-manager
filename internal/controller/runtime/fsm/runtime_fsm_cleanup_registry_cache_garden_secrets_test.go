package fsm

import (
	"context"
	"errors"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/registrycache"
	"testing"
	"time"

	registrycachev1beta1 "github.com/kyma-project/registry-cache/api/v1beta1"
	"github.com/onsi/gomega"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	util "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestFnCleanupRegistryCacheGardenSecrets(t *testing.T) {
	// given
	testCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	testScheme := runtime.NewScheme()
	util.Must(registrycachev1beta1.AddToScheme(testScheme))
	util.Must(corev1.AddToScheme(testScheme))

	gomega.RegisterTestingT(t)

	registryCacheConfig := &registrycachev1beta1.RegistryCacheConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cache1",
			Namespace: "default",
		},
		Spec: registrycachev1beta1.RegistryCacheConfigSpec{
			Upstream: "docker.io",
		},
	}

	secret1 := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "secret1",
			Namespace: "garden-",
			Labels: map[string]string{
				registrycache.RuntimeSecretLabel: "test-runtime",
			},
			Annotations: map[string]string{
				registrycache.CacheIDAnnotation: "uid2",
			},
		},
		Immutable: ptr.To(true),
		Data: map[string][]byte{
			"username": []byte("user"),
			"password": []byte("password"),
		},
	}

	testFSM := setupFakeFSMForTest(testScheme, registryCacheConfig, secret1)
	testFSM.RegistryCacheConfigControllerEnabled = true
	rt := makeInputRuntimeWithRegistryCache()
	systemState := &systemState{instance: rt}

	// When
	nextState, res, err := sFnCleanupRegistryCacheGardenSecrets(testCtx, testFSM, systemState)

	// Then
	require.NoError(t, err)
	require.Nil(t, res)
	require.NotNil(t, nextState)
	require.Contains(t, nextState.name(), "sFnUpdateStatus")

	var updatedCacheConfig registrycachev1beta1.RegistryCacheConfig
	runtimeClient, err := testFSM.RuntimeClientGetter.Get(testCtx, rt)
	require.NoError(t, err)

	require.NoError(t, runtimeClient.Get(testCtx, client.ObjectKey{Name: "cache1", Namespace: "default"}, &updatedCacheConfig))
	require.Equal(t, registrycachev1beta1.ReadyState, updatedCacheConfig.Status.State)
	require.Len(t, updatedCacheConfig.Status.Conditions, 1)
	require.Equal(t, string(registrycachev1beta1.ConditionTypeRegistryCacheConfigured), updatedCacheConfig.Status.Conditions[0].Type)
	require.Equal(t, metav1.ConditionTrue, updatedCacheConfig.Status.Conditions[0].Status)
	require.Equal(t, string(registrycachev1beta1.ConditionReasonRegistryCacheConfigured), updatedCacheConfig.Status.Conditions[0].Reason)
}

func TestFnCleanupRegistryCacheGardenSecrets_ControllerDisabled(t *testing.T) {
	// given
	testCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	testScheme := runtime.NewScheme()
	util.Must(corev1.AddToScheme(testScheme))

	gomega.RegisterTestingT(t)

	testFSM := setupFakeFSMForTest(testScheme)
	testFSM.RegistryCacheConfigControllerEnabled = false

	rt := makeInputRuntimeWithRegistryCache()
	systemState := &systemState{instance: rt}

	// when
	nextState, res, err := sFnCleanupRegistryCacheGardenSecrets(testCtx, testFSM, systemState)

	// then
	require.NoError(t, err)
	require.Nil(t, res)
	require.NotNil(t, nextState)
	require.Contains(t, nextState.name(), "sFnConfigureSKR")
}

func TestFnCleanupRegistryCacheGardenSecrets_RuntimeClientGetterFails(t *testing.T) {
	// given
	testCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	testScheme := runtime.NewScheme()
	util.Must(registrycachev1beta1.AddToScheme(testScheme))
	util.Must(corev1.AddToScheme(testScheme))

	gomega.RegisterTestingT(t)

	getterErr := errors.New("runtime client not available")
	testFSM := must(newFakeFSM,
		withMockedMetrics(),
		withTestFinalizer,
		withShootNamespace("garden-"),
		withFailedRuntimeK8sClient(getterErr, testScheme),
		withFakeEventRecorder(1),
		withDefaultReconcileDuration(),
	)
	testFSM.RegistryCacheConfigControllerEnabled = true

	rt := makeInputRuntimeWithRegistryCache()
	systemState := &systemState{instance: rt}

	// when
	nextState, res, err := sFnCleanupRegistryCacheGardenSecrets(testCtx, testFSM, systemState)

	// then
	require.NoError(t, err)
	require.Nil(t, res)
	require.NotNil(t, nextState)
	require.Contains(t, nextState.name(), "sFnUpdateStatus")
}

func TestFnCleanupRegistryCacheGardenSecrets_NoCachingAfterCleanup(t *testing.T) {
	// given
	testCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	testScheme := runtime.NewScheme()
	util.Must(registrycachev1beta1.AddToScheme(testScheme))
	util.Must(corev1.AddToScheme(testScheme))

	gomega.RegisterTestingT(t)

	// orphaned garden secret to be removed
	orphanedSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "reg-cache-old-uid",
			Namespace: "garden-",
			Labels: map[string]string{
				registrycache.RuntimeSecretLabel: "test-runtime",
			},
			Annotations: map[string]string{
				registrycache.CacheIDAnnotation: "old-uid",
			},
		},
	}

	testFSM := setupFakeFSMForTest(testScheme, orphanedSecret)
	testFSM.RegistryCacheConfigControllerEnabled = true

	// runtime with no caching specs (module was disabled)
	rt := imv1.Runtime{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-runtime",
			Namespace: "kcp-system",
		},
		Spec: imv1.RuntimeSpec{
			Caching: []imv1.ImageRegistryCache{},
		},
	}
	systemState := &systemState{instance: rt}

	// when
	nextState, res, err := sFnCleanupRegistryCacheGardenSecrets(testCtx, testFSM, systemState)

	// then
	require.NoError(t, err)
	require.Nil(t, res)
	require.NotNil(t, nextState)
	require.Contains(t, nextState.name(), "sFnConfigureSKR")

	var deletedSecret corev1.Secret
	getErr := testFSM.GardenClient.Get(testCtx, client.ObjectKey{Name: orphanedSecret.Name, Namespace: orphanedSecret.Namespace}, &deletedSecret)
	require.True(t, apierrors.IsNotFound(getErr), "orphaned garden secret should have been deleted")
}
