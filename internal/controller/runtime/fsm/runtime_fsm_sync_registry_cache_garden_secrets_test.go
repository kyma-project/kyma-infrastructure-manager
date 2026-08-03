package fsm

import (
	"context"
	"errors"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/registrycache"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	util "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/utils/ptr"
	"testing"
	"time"

	registrycachev1beta1 "github.com/kyma-project/registry-cache/api/v1beta1"
	"github.com/onsi/gomega"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestFnSyncRegistryCacheGardenSecrets(t *testing.T) {
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
			Namespace: "default",
		},
		Immutable: ptr.To(true),
		Data: map[string][]byte{
			"username": []byte("user"),
			"password": []byte("password")},
	}

	fsm := setupFakeFSMForTest(testScheme, registryCacheConfig, secret1)
	rt := makeInputRuntimeWithRegistryCache()
	systemState := &systemState{instance: rt}

	// when
	nextState, res, err := sFnSyncRegistryCacheGardenSecrets(testCtx, fsm, systemState)

	// then
	require.NoError(t, err)
	require.Nil(t, res)
	require.NotNil(t, nextState)
	require.Contains(t, nextState.name(), "sFnPatchExistingShoot")
}

func TestFnSyncRegistryCacheGardenSecrets_ControllerDisabled(t *testing.T) {
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
	nextState, res, err := sFnSyncRegistryCacheGardenSecrets(testCtx, testFSM, systemState)

	// then
	require.NoError(t, err)
	require.Nil(t, res)
	require.NotNil(t, nextState)
	require.Contains(t, nextState.name(), "sFnPatchExistingShoot")
}

func TestFnSyncRegistryCacheGardenSecrets_EmptyCaching(t *testing.T) {
	// given
	testCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	testScheme := runtime.NewScheme()
	util.Must(registrycachev1beta1.AddToScheme(testScheme))
	util.Must(corev1.AddToScheme(testScheme))

	gomega.RegisterTestingT(t)

	testFSM := setupFakeFSMForTest(testScheme)
	testFSM.RegistryCacheConfigControllerEnabled = true

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
	nextState, res, err := sFnSyncRegistryCacheGardenSecrets(testCtx, testFSM, systemState)

	// then
	require.NoError(t, err)
	require.Nil(t, res)
	require.NotNil(t, nextState)
	require.Contains(t, nextState.name(), "sFnPatchExistingShoot")
}

func TestFnSyncRegistryCacheGardenSecrets_RuntimeClientGetterFails(t *testing.T) {
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
	nextState, res, err := sFnSyncRegistryCacheGardenSecrets(testCtx, testFSM, systemState)

	// then
	require.NoError(t, err)
	require.Nil(t, res)
	require.NotNil(t, nextState)
	require.Contains(t, nextState.name(), "sFnUpdateStatus")
}

func TestFnSyncRegistryCacheGardenSecrets_CachingWithoutSecrets(t *testing.T) {
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

	testFSM := setupFakeFSMForTest(testScheme, registryCacheConfig)
	testFSM.RegistryCacheConfigControllerEnabled = true

	rt := imv1.Runtime{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-runtime",
			Namespace: "kcp-system",
		},
		Spec: imv1.RuntimeSpec{
			Caching: []imv1.ImageRegistryCache{
				{
					Name:      "cache1",
					Namespace: "default",
					UID:       "uid1",
					Config: registrycachev1beta1.RegistryCacheConfigSpec{
						Upstream: "docker.io",
						// SecretReferenceName intentionally nil
					},
				},
			},
		},
	}
	systemState := &systemState{instance: rt}

	// when
	nextState, res, err := sFnSyncRegistryCacheGardenSecrets(testCtx, testFSM, systemState)

	// then
	require.NoError(t, err)
	require.Nil(t, res)
	require.NotNil(t, nextState)
	require.Contains(t, nextState.name(), "sFnPatchExistingShoot")
}

func TestFnSyncRegistryCacheGardenSecrets_CachingWithSecret(t *testing.T) {
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
			Upstream:            "docker.io",
			SecretReferenceName: ptr.To("secret1"),
		},
	}

	// secret on the runtime cluster referenced by the cache config
	runtimeSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "secret1",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"username": []byte("user"),
			"password": []byte("password"),
		},
	}

	testFSM := setupFakeFSMForTest(testScheme, registryCacheConfig, runtimeSecret)
	testFSM.RegistryCacheConfigControllerEnabled = true

	rt := makeInputRuntimeWithRegistryCache()
	systemState := &systemState{instance: rt}

	// when
	nextState, res, err := sFnSyncRegistryCacheGardenSecrets(testCtx, testFSM, systemState)

	// then
	require.NoError(t, err)
	require.Nil(t, res)
	require.NotNil(t, nextState)
	require.Contains(t, nextState.name(), "sFnPatchExistingShoot")

	// garden secret should have been created from the runtime secret
	var gardenSecrets corev1.SecretList

	getErr := testFSM.GardenClient.List(testCtx, &gardenSecrets, client.MatchingLabels{registrycache.CacheIDLabel: rt.Spec.Caching[0].UID}, client.InNamespace("garden-"))

	require.NoError(t, getErr)
	require.Equal(t, runtimeSecret.Data, gardenSecrets.Items[0].Data)
}

func TestFnSyncRegistryCacheGardenSecrets_StatusSetToPending(t *testing.T) {
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

	testFSM := setupFakeFSMForTest(testScheme, registryCacheConfig)
	testFSM.RegistryCacheConfigControllerEnabled = true

	rt := imv1.Runtime{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-runtime",
			Namespace: "kcp-system",
		},
		Spec: imv1.RuntimeSpec{
			Caching: []imv1.ImageRegistryCache{
				{
					Name:      "cache1",
					Namespace: "default",
					UID:       "uid1",
					Config: registrycachev1beta1.RegistryCacheConfigSpec{
						Upstream: "docker.io",
						// SecretReferenceName intentionally nil — no secret to sync
					},
				},
			},
		},
	}
	systemState := &systemState{instance: rt}

	// when
	nextState, res, err := sFnSyncRegistryCacheGardenSecrets(testCtx, testFSM, systemState)

	// then
	require.NoError(t, err)
	require.Nil(t, res)
	require.NotNil(t, nextState)
	require.Contains(t, nextState.name(), "sFnPatchExistingShoot")

	runtimeClient, err := testFSM.RuntimeClientGetter.Get(testCtx, rt)
	require.NoError(t, err)

	var updatedCacheConfig registrycachev1beta1.RegistryCacheConfig
	require.NoError(t, runtimeClient.Get(testCtx, client.ObjectKey{Name: "cache1", Namespace: "default"}, &updatedCacheConfig))
	require.Equal(t, registrycachev1beta1.PendingState, updatedCacheConfig.Status.State)
	require.Len(t, updatedCacheConfig.Status.Conditions, 1)
	require.Equal(t, string(registrycachev1beta1.ConditionTypeRegistryCacheConfigured), updatedCacheConfig.Status.Conditions[0].Type)
	require.Equal(t, metav1.ConditionUnknown, updatedCacheConfig.Status.Conditions[0].Status)
	require.Equal(t, string(registrycachev1beta1.ConditionReasonRegistryCacheConfigured), updatedCacheConfig.Status.Conditions[0].Reason)
}

func makeInputRuntimeWithRegistryCache() imv1.Runtime {
	return imv1.Runtime{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-runtime",
			Namespace: "kcp-system",
		},
		Spec: imv1.RuntimeSpec{
			Caching: []imv1.ImageRegistryCache{
				{
					Name:      "cache1",
					Namespace: "default",
					UID:       "uid1",
					Config: registrycachev1beta1.RegistryCacheConfigSpec{
						SecretReferenceName: ptr.To("secret1"),
					},
				},
			},
		},
	}
}
