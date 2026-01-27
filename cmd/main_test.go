package main

import (
	"testing"

	gardeneroidc "github.com/gardener/oidc-webhook-authenticator/apis/authentication/v1alpha1"
	kyma "github.com/kyma-project/lifecycle-manager/api/v1beta2"
	registrycacheapi "github.com/kyma-project/registry-cache/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
)

func TestPrebuiltRuntimeSchemeRegistersTypes(t *testing.T) {
	// Build the scheme the same way main.go does
	prebuiltRuntimeScheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(prebuiltRuntimeScheme))
	utilruntime.Must(registrycacheapi.AddToScheme(prebuiltRuntimeScheme))
	utilruntime.Must(kyma.AddToScheme(prebuiltRuntimeScheme))
	utilruntime.Must(gardeneroidc.AddToScheme(prebuiltRuntimeScheme))
	utilruntime.Must(apiextensions.AddToScheme(prebuiltRuntimeScheme))

	// Verify Kyma Kyma type is registered
	if gvks, _, err := prebuiltRuntimeScheme.ObjectKinds(&kyma.Kyma{}); err != nil {
		t.Fatalf("kyma.Kyma not registered in scheme: %v", err)
	} else if len(gvks) == 0 {
		t.Fatalf("kyma.Kyma returned no GVKs from scheme")
	}

	// Verify registry-cache config type is registered
	if gvks, _, err := prebuiltRuntimeScheme.ObjectKinds(&registrycacheapi.RegistryCacheConfig{}); err != nil {
		t.Fatalf("registry-cache RegistryCacheConfig not registered in scheme: %v", err)
	} else if len(gvks) == 0 {
		t.Fatalf("registry-cache RegistryCacheConfig returned no GVKs from scheme")
	}

	// Verify gardener oidc type is registered
	if gvks, _, err := prebuiltRuntimeScheme.ObjectKinds(&gardeneroidc.OpenIDConnect{}); err != nil {
		t.Fatalf("gardener oidc OpenIDConnect not registered in scheme: %v", err)
	} else if len(gvks) == 0 {
		t.Fatalf("gardener oidc OpenIDConnect returned no GVKs from scheme")
	}

	// Verify apiextensions CRD type is registered
	if gvks, _, err := prebuiltRuntimeScheme.ObjectKinds(&apiextensions.CustomResourceDefinition{}); err != nil {
		t.Fatalf("apiextensions CustomResourceDefinition not registered in scheme: %v", err)
	} else if len(gvks) == 0 {
		t.Fatalf("apiextensions CustomResourceDefinition returned no GVKs from scheme")
	}

	// Verify core v1 Pod is registered (client-go scheme)
	if gvks, _, err := prebuiltRuntimeScheme.ObjectKinds(&corev1.Pod{}); err != nil {
		t.Fatalf("corev1.Pod not registered in scheme: %v", err)
	} else if len(gvks) == 0 {
		t.Fatalf("corev1.Pod returned no GVKs from scheme")
	}
}

