package structuredauth

import (
	"context"
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"
	"testing"
)

func TestCreateOrUpdateConfigMap(t *testing.T) {
	// start of fake client setup
	scheme := runtime.NewScheme()

	err := corev1.AddToScheme(scheme)
	require.NoError(t, err)

	var fakeClient = fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	existingConfig := AuthenticationConfiguration{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AuthenticationConfiguration",
			APIVersion: "apiserver.config.k8s.io/v1beta1",
		},
		JWT: []JWTAuthenticator{
			{
				Issuer: Issuer{
					URL:       "issuer",
					Audiences: []string{"client"},
				},
				ClaimMappings: ClaimMappings{
					Username: PrefixedClaim{
						Claim:  "username",
						Prefix: ptr.To("prefix"),
					},
					Groups: PrefixedClaim{
						Claim:  "groups",
						Prefix: ptr.To("groups-prefix"),
					},
				},
			},
		},
	}

	configBytes, err := yaml.Marshal(existingConfig)
	require.NoError(t, err)

	tests := []struct {
		name              string
		cmName            string
		oidcConfig        gardener.OIDCConfig
		existingConfigMap *corev1.ConfigMap
		prepareFunc       func(oidcConfig *gardener.OIDCConfig) error
	}{
		{
			name:   "Should create config map with OIDC config",
			cmName: "cm1",
			oidcConfig: gardener.OIDCConfig{
				ClientID:       ptr.To("client"),
				IssuerURL:      ptr.To("issuer"),
				UsernameClaim:  ptr.To("username"),
				UsernamePrefix: ptr.To("prefix"),
				GroupsClaim:    ptr.To("groups"),
				GroupsPrefix:   ptr.To("groups-prefix"),
			},
		},
		{
			name:   "Should create config map with OIDC config and use default groups prefix",
			cmName: "cm2",
			oidcConfig: gardener.OIDCConfig{
				ClientID:       ptr.To("client"),
				IssuerURL:      ptr.To("issuer"),
				UsernameClaim:  ptr.To("username"),
				UsernamePrefix: ptr.To("prefix"),
				GroupsClaim:    ptr.To("groups"),
			},
		},
		{
			name:   "Should update existing config map with OIDC config",
			cmName: "cm3",
			existingConfigMap: &corev1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ConfigMap",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "name",
					Namespace: "default",
				},
				Data: map[string]string{
					"config.yaml": string(configBytes),
				},
			},
			oidcConfig: gardener.OIDCConfig{
				ClientID:       ptr.To("client1"),
				IssuerURL:      ptr.To("issuer1"),
				UsernameClaim:  ptr.To("username1"),
				UsernamePrefix: ptr.To("prefix1"),
				GroupsClaim:    ptr.To("groups1"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if tt.existingConfigMap != nil {
				err = fakeClient.Create(context.Background(), tt.existingConfigMap)
				require.NoError(t, err)
			}

			err := CreateOrUpdateStructuredAuthConfigMap(context.Background(), fakeClient, types.NamespacedName{Namespace: "default", Name: tt.cmName}, tt.oidcConfig)
			require.NoError(t, err)

			cm := &corev1.ConfigMap{}
			err = fakeClient.Get(context.Background(), types.NamespacedName{Namespace: "default", Name: tt.cmName}, cm)
			require.NoError(t, err)

			authenticationConfigString := cm.Data["config.yaml"]
			var authenticationConfiguration AuthenticationConfiguration
			err = yaml.Unmarshal([]byte(authenticationConfigString), &authenticationConfiguration)

			require.NoError(t, err)

			assert.Equal(t, "apiserver.config.k8s.io/v1beta1", authenticationConfiguration.APIVersion)
			assert.Equal(t, "AuthenticationConfiguration", authenticationConfiguration.Kind)
			assert.Equal(t, 1, len(authenticationConfiguration.JWT))
			assert.Equal(t, *tt.oidcConfig.IssuerURL, authenticationConfiguration.JWT[0].Issuer.URL)
			assert.Equal(t, []string{*tt.oidcConfig.ClientID}, authenticationConfiguration.JWT[0].Issuer.Audiences)
			assert.Equal(t, *tt.oidcConfig.UsernameClaim, authenticationConfiguration.JWT[0].ClaimMappings.Username.Claim)
			assert.Equal(t, tt.oidcConfig.UsernamePrefix, authenticationConfiguration.JWT[0].ClaimMappings.Username.Prefix)
			assert.Equal(t, *tt.oidcConfig.GroupsClaim, authenticationConfiguration.JWT[0].ClaimMappings.Groups.Claim)
			if tt.oidcConfig.GroupsPrefix == nil {
				assert.Equal(t, "-", *authenticationConfiguration.JWT[0].ClaimMappings.Groups.Prefix)
			} else {
				assert.Equal(t, tt.oidcConfig.GroupsPrefix, authenticationConfiguration.JWT[0].ClaimMappings.Groups.Prefix)
			}
		})
	}
}

func TestDeleteStructuredConfigMap(t *testing.T) {

	scheme := runtime.NewScheme()

	err := corev1.AddToScheme(scheme)
	require.NoError(t, err)

	var fakeClient = fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	tests := []struct {
		name              string
		existingConfigMap *corev1.ConfigMap
		shoot             gardener.Shoot
	}{
		{
			name: "Should delete config map",
			existingConfigMap: &corev1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ConfigMap",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "name",
					Namespace: "default",
				},
				Data: map[string]string{
					"config.yaml": "config",
				},
			},
			shoot: gardener.Shoot{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "shoot",
					Namespace: "default",
				},
				Spec: gardener.ShootSpec{
					Kubernetes: gardener.Kubernetes{
						KubeAPIServer: &gardener.KubeAPIServerConfig{
							StructuredAuthentication: &gardener.StructuredAuthentication{
								ConfigMapName: "name",
							},
						},
					},
				},
			},
		},
		{
			name: "Should not fail when config map doesn't exist",
			shoot: gardener.Shoot{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "shoot",
					Namespace: "default",
				},
				Spec: gardener.ShootSpec{
					Kubernetes: gardener.Kubernetes{
						KubeAPIServer: &gardener.KubeAPIServerConfig{
							StructuredAuthentication: &gardener.StructuredAuthentication{
								ConfigMapName: "name",
							},
						},
					},
				},
			},
		},
		{
			name: "Should not fail if structured config is not set",
			shoot: gardener.Shoot{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "shoot",
					Namespace: "default",
				},
				Spec: gardener.ShootSpec{
					Kubernetes: gardener.Kubernetes{
						KubeAPIServer: &gardener.KubeAPIServerConfig{},
					},
				},
			},
		},
		{
			name: "Should not fail if KubeAPIServer config is not set",
			shoot: gardener.Shoot{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "shoot",
					Namespace: "default",
				},
				Spec: gardener.ShootSpec{
					Kubernetes: gardener.Kubernetes{},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if tt.existingConfigMap != nil {
				err = fakeClient.Create(context.Background(), tt.existingConfigMap)
				require.NoError(t, err)
			}

			err = DeleteStructuredConfigMap(context.Background(), fakeClient, tt.shoot)
			require.NoError(t, err)

			cm := &corev1.ConfigMap{}
			err = fakeClient.Get(context.Background(), types.NamespacedName{Namespace: "default", Name: "name"}, cm)
			require.Error(t, err)
			require.Equal(t, true, errors.IsNotFound(err))
		})
	}
}
