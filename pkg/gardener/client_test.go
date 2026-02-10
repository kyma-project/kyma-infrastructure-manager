package gardener

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestGetRuntimeClientWithScheme_NoData(t *testing.T) {
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "no-data-secret"},
	}

	if _, err := GetRuntimeClientWithScheme(secret, runtime.NewScheme()); err == nil {
		t.Fatalf("expected error when secret has no data, got nil")
	}
}

func TestGetRuntimeClientWithScheme_InvalidKubeconfig(t *testing.T) {
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "invalid-kubeconfig"},
		Data: map[string][]byte{
			"config": []byte("this-is-not-a-kubeconfig"),
		},
	}

	if _, err := GetRuntimeClientWithScheme(secret, runtime.NewScheme()); err == nil {
		t.Fatalf("expected error when kubeconfig is invalid, got nil")
	}
}
