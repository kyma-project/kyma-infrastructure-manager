package main

import (
	"maps"
	"testing"

	gardeneroidc "github.com/gardener/oidc-webhook-authenticator/apis/authentication/v1alpha1"
	"github.com/kyma-project/infrastructure-manager/pkg/config"
	kyma "github.com/kyma-project/lifecycle-manager/api/v1beta2"
	registrycacheapi "github.com/kyma-project/registry-cache/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

func TestEnableClusterTrustBundleFeatureForSKR(t *testing.T) {
	tests := []struct {
		name                 string
		initialFeatureGates  map[string]bool
		initialRuntimeConfig map[string]bool
		wantFeatureGates     map[string]bool
		wantRuntimeConfig    map[string]bool
	}{
		{
			name: "initializes nil maps",
			wantFeatureGates: map[string]bool{
				"ClusterTrustBundle":           true,
				"ClusterTrustBundleProjection": true,
			},
			wantRuntimeConfig: map[string]bool{
				"certificates.k8s.io/v1beta1/clustertrustbundles": true,
			},
		},
		{
			name: "enables disabled gates and preserves existing settings",
			initialFeatureGates: map[string]bool{
				"ClusterTrustBundle":           false,
				"ClusterTrustBundleProjection": false,
				"ExistingEnabledGate":          true,
				"ExistingDisabledGate":         false,
			},
			initialRuntimeConfig: map[string]bool{
				"certificates.k8s.io/v1beta1/clustertrustbundles": false,
				"existing/enabled":  true,
				"existing/disabled": false,
			},
			wantFeatureGates: map[string]bool{
				"ClusterTrustBundle":           true,
				"ClusterTrustBundleProjection": true,
				"ExistingEnabledGate":          true,
				"ExistingDisabledGate":         false,
			},
			wantRuntimeConfig: map[string]bool{
				"certificates.k8s.io/v1beta1/clustertrustbundles": true,
				"existing/enabled":  true,
				"existing/disabled": false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converterConfig := config.ConverterConfig{
				Kubernetes: config.KubernetesConfig{
					KubeApiServer: config.KubeApiServer{
						FeatureGates:  maps.Clone(tt.initialFeatureGates),
						RuntimeConfig: maps.Clone(tt.initialRuntimeConfig),
					},
					Kubelet: config.Kubelet{
						FeatureGates: maps.Clone(tt.initialFeatureGates),
					},
				},
			}

			enableClusterTrustBundleFeatureForSKR(&converterConfig)

			if got := converterConfig.Kubernetes.KubeApiServer.FeatureGates; !maps.Equal(got, tt.wantFeatureGates) {
				t.Errorf("KubeApiServer.FeatureGates = %v, want %v", got, tt.wantFeatureGates)
			}
			if got := converterConfig.Kubernetes.KubeApiServer.RuntimeConfig; !maps.Equal(got, tt.wantRuntimeConfig) {
				t.Errorf("KubeApiServer.RuntimeConfig = %v, want %v", got, tt.wantRuntimeConfig)
			}
			if got := converterConfig.Kubernetes.Kubelet.FeatureGates; !maps.Equal(got, tt.wantFeatureGates) {
				t.Errorf("Kubelet.FeatureGates = %v, want %v", got, tt.wantFeatureGates)
			}
		})
	}
}

func TestPrebuiltRuntimeSchemeRegistersTypes(t *testing.T) {
	// Build the scheme the same way main.go does
	prebuiltRuntimeScheme := CreateRuntimeScheme()

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
