package fsm

import (
	"context"
	"github.com/kyma-project/infrastructure-manager/internal/registrycache"
	"testing"
	"time"

	registrycachev1beta1 "github.com/kyma-project/kim-snatch/api/v1beta1"
	"github.com/onsi/gomega"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	util "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/utils/ptr"
)

func TestFnFinalizeRegistryCache(t *testing.T) {
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

	fsm := setupFakeFSMForTest(testScheme, registryCacheConfig, secret1)
	rt := makeInputRuntimeWithRegistryCache()
	systemState := &systemState{instance: rt}

	// When
	nextState, res, err := sFnFinalizeRegistryCache(testCtx, fsm, systemState)

	// Then
	require.NoError(t, err)
	require.Nil(t, res)
	require.NotNil(t, nextState)
	require.Contains(t, nextState.name(), "sFnConfigureSKR")
}
