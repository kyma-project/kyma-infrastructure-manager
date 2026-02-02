package fsm_test

import (
	"context"
	"testing"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/controller/runtime/fsm"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestRuntimeClientGetterWithScheme_Get_InvalidSecret(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)

	// create a fake kcp client with no secrets
	kcpClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	getter := fsm.NewRuntimeClientGetterWithScheme(kcpClient, scheme)

	// runtime without labels will cause getter to fail when locating secret
	rt := imv1.Runtime{}
	ctx := context.Background()
	if _, err := getter.Get(ctx, rt); err == nil {
		t.Fatalf("expected error when runtime has no labels and no secret exists, got nil")
	}
}

func TestRuntimeClientGetterWithScheme_Get_SecretMissingData(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)

	// create secret without data
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "kubeconfig-someid", Namespace: "kcp-system"},
	}

	kcpClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build()

	getter := fsm.NewRuntimeClientGetterWithScheme(kcpClient, scheme)

	rt := imv1.Runtime{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"kyma-project.io/runtime-id": "someid"}, Namespace: "kcp-system"}}
	ctx := context.Background()
	if _, err := getter.Get(ctx, rt); err == nil {
		t.Fatalf("expected error when secret has no kubeconfig data, got nil")
	}
}

