package fsm

import (
	"context"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	util "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/utils/ptr"
	"testing"
	"time"

	registrycachev1beta1 "github.com/kyma-project/kim-snatch/api/v1beta1"
	"github.com/onsi/gomega"
	"github.com/stretchr/testify/require"
)

func TestFnPrepareRegistryCache(t *testing.T) {
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

	// Setup fake FSM and runtime with caching enabled
	fsm := setupFakeFSMForTest(testScheme, registryCacheConfig, secret1)
	rt := makeInputRuntimeWithRegistryCache()
	systemState := &systemState{instance: rt}

	// Call the function
	nextState, res, err := sFnPrepareRegistryCache(testCtx, fsm, systemState)

	// Assertions
	require.NoError(t, err)
	require.Nil(t, res)
	require.NotNil(t, nextState)
	require.Contains(t, nextState.name(), "sFnPatchExistingShoot")
}

func makeInputRuntimeWithRegistryCache() imv1.Runtime {
	return imv1.Runtime{
		Spec: imv1.RuntimeSpec{
			Caching: []imv1.ImageRegistryCache{
				{
					Name:      "cache1",
					Namespace: "default",
					Config: registrycachev1beta1.RegistryCacheConfigSpec{
						SecretReferenceName: ptr.To("secret1"),
					},
				},
			},
		},
	}
}
