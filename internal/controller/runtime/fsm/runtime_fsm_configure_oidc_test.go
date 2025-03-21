package fsm

import (
	"context"
	"testing"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	authenticationv1alpha1 "github.com/gardener/oidc-webhook-authenticator/apis/authentication/v1alpha1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestOidcState(t *testing.T) {
	t.Run("Should switch state to ApplyClusterRoleBindings when OIDC extension is disabled", func(t *testing.T) {
		// given
		ctx := context.Background()
		fsm := &fsm{}

		runtimeStub := runtimeForTest()
		shootStub := shootForTest()
		oidcService := gardener.Extension{
			Type:     "shoot-oidc-service",
			Disabled: ptr.To(true),
		}
		shootStub.Spec.Extensions = append(shootStub.Spec.Extensions, oidcService)

		systemState := &systemState{
			instance: runtimeStub,
			shoot:    shootStub,
		}

		expectedRuntimeConditions := []metav1.Condition{
			{
				Type:    string(imv1.ConditionTypeOidcConfigured),
				Reason:  string(imv1.ConditionReasonOidcConfigured),
				Status:  "True",
				Message: "OIDC extension disabled",
			},
		}

		// when
		stateFn, _, _ := sFnConfigureOidc(ctx, fsm, systemState)

		// then
		require.Contains(t, stateFn.name(), "sFnApplyClusterRoleBindings")
		assertEqualConditions(t, expectedRuntimeConditions, systemState.instance.Status.Conditions)
	})

	t.Run("Should switch state to ApplyClusterRoleBindings when  multi OIDC support is disabled", func(t *testing.T) {
		// given
		ctx := context.Background()
		fsm := &fsm{}

		runtimeStub := runtimeForTest()
		runtimeStub.ObjectMeta.Labels = map[string]string{
			"operator.kyma-project.io/created-by-migrator": "true",
		}

		shootStub := shootForTest()
		oidcService := gardener.Extension{
			Type:     "shoot-oidc-service",
			Disabled: ptr.To(false),
		}
		shootStub.Spec.Extensions = append(shootStub.Spec.Extensions, oidcService)

		systemState := &systemState{
			instance: runtimeStub,
			shoot:    shootStub,
		}

		expectedRuntimeConditions := []metav1.Condition{
			{
				Type:    string(imv1.ConditionTypeOidcConfigured),
				Reason:  string(imv1.ConditionReasonOidcConfigured),
				Status:  "True",
				Message: "Multi OIDC not supported for migrated runtimes",
			},
		}

		// when
		stateFn, _, _ := sFnConfigureOidc(ctx, fsm, systemState)

		// then
		require.Contains(t, stateFn.name(), "sFnApplyClusterRoleBindings")
		assertEqualConditions(t, expectedRuntimeConditions, systemState.instance.Status.Conditions)
	})

	t.Run("Should configure OIDC using defaults", func(t *testing.T) {
		// given
		ctx := context.Background()

		// start of fake client setup
		scheme, err := newOIDCTestScheme()
		require.NoError(t, err)
		var fakeClient = fake.NewClientBuilder().
			WithScheme(scheme).
			Build()
		testFsm := &fsm{K8s: K8s{
			ShootClient: fakeClient,
			Client:      fakeClient,
		},
			RCCfg: RCCfg{
				Config: config.Config{
					ClusterConfig: config.ClusterConfig{
						DefaultSharedIASTenant: createConverterOidcConfig("defaut-client-id"),
					},
				},
			},
		}
		GetShootClient = func(
			_ context.Context,
			_ client.Client,
			_ imv1.Runtime) (client.Client, error) {
			return fakeClient, nil
		}
		// end of fake client setup

		for _, tc := range []struct {
			name                 string
			additionalOIDCConfig *[]gardener.OIDCConfig
		}{
			{"Should configure OIDC using defaults when additional OIDC config is nil", nil},
			{"Should configure OIDC using defaults when additional OIDC config contains empty array", &[]gardener.OIDCConfig{}},
			{"Should configure OIDC using defaults when additional OIDC config contains one empty element", &[]gardener.OIDCConfig{{}}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				runtimeStub := runtimeForTest()

				runtimeStub.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig = tc.additionalOIDCConfig

				shootStub := shootForTest()
				oidcService := gardener.Extension{
					Type:     "shoot-oidc-service",
					Disabled: ptr.To(false),
				}
				shootStub.Spec.Extensions = append(shootStub.Spec.Extensions, oidcService)

				systemState := &systemState{
					instance: runtimeStub,
					shoot:    shootStub,
				}

				expectedRuntimeConditions := []metav1.Condition{
					{
						Type:    string(imv1.ConditionTypeOidcConfigured),
						Reason:  string(imv1.ConditionReasonOidcConfigured),
						Status:  "True",
						Message: "OIDC configuration completed",
					},
				}

				// when
				stateFn, _, _ := sFnConfigureOidc(ctx, testFsm, systemState)

				// then
				require.Contains(t, stateFn.name(), "sFnApplyClusterRoleBindings")

				var openIdConnects authenticationv1alpha1.OpenIDConnectList

				err = fakeClient.List(ctx, &openIdConnects)
				require.NoError(t, err)
				assert.Len(t, openIdConnects.Items, 1)

				assertOIDCCRD(t, "kyma-oidc-0", "defaut-client-id", openIdConnects.Items[0])
				assertEqualConditions(t, expectedRuntimeConditions, systemState.instance.Status.Conditions)
			})
		}
	})

	t.Run("Should not crash and configure OIDC using defaults when Disabled field is missing in extension data", func(t *testing.T) {
		// given
		ctx := context.Background()

		// start of fake client setup
		scheme, err := newOIDCTestScheme()
		require.NoError(t, err)
		var fakeClient = fake.NewClientBuilder().
			WithScheme(scheme).
			Build()
		testFsm := &fsm{K8s: K8s{
			ShootClient: fakeClient,
			Client:      fakeClient,
		},
			RCCfg: RCCfg{
				Config: config.Config{
					ClusterConfig: config.ClusterConfig{
						DefaultSharedIASTenant: createConverterOidcConfig("defaut-client-id"),
					},
				},
			},
		}
		GetShootClient = func(
			_ context.Context,
			_ client.Client,
			_ imv1.Runtime) (client.Client, error) {
			return fakeClient, nil
		}
		// end of fake client setup

		runtimeStub := runtimeForTest()
		shootStub := shootForTest()
		oidcService := gardener.Extension{
			Type:     "shoot-oidc-service",
			Disabled: nil,
		}
		shootStub.Spec.Extensions = append(shootStub.Spec.Extensions, oidcService)

		systemState := &systemState{
			instance: runtimeStub,
			shoot:    shootStub,
		}

		expectedRuntimeConditions := []metav1.Condition{
			{
				Type:    string(imv1.ConditionTypeOidcConfigured),
				Reason:  string(imv1.ConditionReasonOidcConfigured),
				Status:  "True",
				Message: "OIDC configuration completed",
			},
		}

		// when
		stateFn, _, _ := sFnConfigureOidc(ctx, testFsm, systemState)

		// then
		require.Contains(t, stateFn.name(), "sFnApplyClusterRoleBindings")

		var openIdConnects authenticationv1alpha1.OpenIDConnectList

		err = fakeClient.List(ctx, &openIdConnects)
		require.NoError(t, err)
		assert.Len(t, openIdConnects.Items, 1)

		assertOIDCCRD(t, "kyma-oidc-0", "defaut-client-id", openIdConnects.Items[0])
		assertEqualConditions(t, expectedRuntimeConditions, systemState.instance.Status.Conditions)
	})

	t.Run("Should configure OIDC based on Runtime CR configuration", func(t *testing.T) {
		// given
		ctx := context.Background()

		// start of fake client setup
		scheme, err := newOIDCTestScheme()
		require.NoError(t, err)
		var fakeClient = fake.NewClientBuilder().
			WithScheme(scheme).
			Build()
		testFsm := &fsm{K8s: K8s{
			ShootClient: fakeClient,
			Client:      fakeClient,
		}}
		GetShootClient = func(
			_ context.Context,
			_ client.Client,
			_ imv1.Runtime) (client.Client, error) {
			return fakeClient, nil
		}
		// end of fake client setup

		runtimeStub := runtimeForTest()
		additionalOidcConfig := &[]gardener.OIDCConfig{}
		*additionalOidcConfig = append(*additionalOidcConfig, createGardenerOidcConfig("runtime-cr-config0"))
		*additionalOidcConfig = append(*additionalOidcConfig, createGardenerOidcConfig("runtime-cr-config1"))
		runtimeStub.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig = additionalOidcConfig

		shootStub := shootForTest()
		oidcService := gardener.Extension{
			Type:     "shoot-oidc-service",
			Disabled: ptr.To(false),
		}
		shootStub.Spec.Extensions = append(shootStub.Spec.Extensions, oidcService)

		systemState := &systemState{
			instance: runtimeStub,
			shoot:    shootStub,
		}

		expectedRuntimeConditions := []metav1.Condition{
			{
				Type:    string(imv1.ConditionTypeOidcConfigured),
				Reason:  string(imv1.ConditionReasonOidcConfigured),
				Status:  "True",
				Message: "OIDC configuration completed",
			},
		}

		// when
		stateFn, _, _ := sFnConfigureOidc(ctx, testFsm, systemState)

		// then
		require.Contains(t, stateFn.name(), "sFnApplyClusterRoleBindings")

		var openIdConnects authenticationv1alpha1.OpenIDConnectList

		err = fakeClient.List(ctx, &openIdConnects)
		require.NoError(t, err)
		assert.Len(t, openIdConnects.Items, 2)
		assert.Equal(t, "kyma-oidc-0", openIdConnects.Items[0].Name)
		assertOIDCCRD(t, "kyma-oidc-0", "runtime-cr-config0", openIdConnects.Items[0])
		assertOIDCCRD(t, "kyma-oidc-1", "runtime-cr-config1", openIdConnects.Items[1])
		assert.Equal(t, imv1.State("Pending"), systemState.instance.Status.State)
		assertEqualConditions(t, expectedRuntimeConditions, systemState.instance.Status.Conditions)
	})

	t.Run("Should first delete existing OpenIDConnect CRs then recreate them", func(t *testing.T) {
		// given
		ctx := context.Background()

		// start of fake client setup
		scheme, err := newOIDCTestScheme()
		require.NoError(t, err)
		var fakeClient = fake.NewClientBuilder().
			WithScheme(scheme).
			Build()
		testFSM := &fsm{K8s: K8s{
			ShootClient: fakeClient,
			Client:      fakeClient,
		}}
		GetShootClient = func(
			_ context.Context,
			_ client.Client,
			_ imv1.Runtime) (client.Client, error) {
			return fakeClient, nil
		}
		// end of fake client setup

		kymaOpenIDConnectCR := createOpenIDConnectCR("old-kyma-oidc", "operator.kyma-project.io/managed-by", "infrastructure-manager")
		err = fakeClient.Create(ctx, kymaOpenIDConnectCR)
		require.NoError(t, err)

		existingOpenIDConnectCR := createOpenIDConnectCR("old-non-kyma-oidc", "customer-label", "should-not-be-deleted")
		err = fakeClient.Create(ctx, existingOpenIDConnectCR)
		require.NoError(t, err)

		runtimeStub := runtimeForTest()
		shootStub := shootForTest()
		oidcService := gardener.Extension{
			Type:     "shoot-oidc-service",
			Disabled: ptr.To(false),
		}
		shootStub.Spec.Extensions = append(shootStub.Spec.Extensions, oidcService)

		systemState := &systemState{
			instance: runtimeStub,
			shoot:    shootStub,
		}

		expectedRuntimeConditions := []metav1.Condition{
			{
				Type:    string(imv1.ConditionTypeOidcConfigured),
				Reason:  string(imv1.ConditionReasonOidcConfigured),
				Status:  "True",
				Message: "OIDC configuration completed",
			},
		}

		// when
		stateFn, _, _ := sFnConfigureOidc(ctx, testFSM, systemState)

		// then
		require.Contains(t, stateFn.name(), "sFnApplyClusterRoleBindings")

		var openIdConnect authenticationv1alpha1.OpenIDConnect
		key := client.ObjectKey{
			Name: "old-kyma-oidc",
		}
		err = fakeClient.Get(ctx, key, &openIdConnect)
		require.Error(t, err)

		key = client.ObjectKey{
			Name: "old-non-kyma-oidc",
		}
		err = fakeClient.Get(ctx, key, &openIdConnect)
		require.NoError(t, err)
		assert.Equal(t, openIdConnect.Name, "old-non-kyma-oidc")

		var openIdConnects authenticationv1alpha1.OpenIDConnectList
		err = fakeClient.List(ctx, &openIdConnects)
		require.NoError(t, err)
		assert.Len(t, openIdConnects.Items, 2)
		assert.Equal(t, "kyma-oidc-0", openIdConnects.Items[0].Name)
		assertEqualConditions(t, expectedRuntimeConditions, systemState.instance.Status.Conditions)
		assert.Equal(t, imv1.State("Pending"), systemState.instance.Status.State)
	})
}

func newOIDCTestScheme() (*runtime.Scheme, error) {
	schema := runtime.NewScheme()

	for _, fn := range []func(*runtime.Scheme) error{
		authenticationv1alpha1.AddToScheme,
	} {
		if err := fn(schema); err != nil {
			return nil, err
		}
	}
	return schema, nil
}

// sets the time to its zero value for comparison purposes
func assertEqualConditions(t *testing.T, expectedConditions []metav1.Condition, actualConditions []metav1.Condition) bool {
	for i := range actualConditions {
		actualConditions[i].LastTransitionTime = metav1.Time{}
	}

	return assert.Equal(t, expectedConditions, actualConditions)
}

func createGardenerOidcConfig(clientId string) gardener.OIDCConfig {
	return gardener.OIDCConfig{
		ClientID:       ptr.To(clientId),
		GroupsClaim:    ptr.To("groups"),
		IssuerURL:      ptr.To("https://my.cool.tokens.com"),
		SigningAlgs:    []string{"RS256"},
		UsernameClaim:  ptr.To("sub"),
		UsernamePrefix: ptr.To("-"),
	}
}

func createConverterOidcConfig(clientId string) config.OidcProvider {
	return config.OidcProvider{
		ClientID:       clientId,
		GroupsClaim:    "groups",
		IssuerURL:      "https://my.cool.tokens.com",
		SigningAlgs:    []string{"RS256"},
		UsernameClaim:  "sub",
		UsernamePrefix: "-",
	}
}

func createOpenIDConnectCR(name string, labelKey, labelValue string) *authenticationv1alpha1.OpenIDConnect {
	return &authenticationv1alpha1.OpenIDConnect{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				labelKey: labelValue,
			},
		},
	}
}

func assertOIDCCRD(t *testing.T, expectedName, expectedClientID string, actual authenticationv1alpha1.OpenIDConnect) {
	assert.Equal(t, expectedName, actual.Name)
	assert.Equal(t, expectedClientID, actual.Spec.ClientID)
	assert.Equal(t, ptr.To("groups"), actual.Spec.GroupsClaim)
	assert.Nil(t, actual.Spec.GroupsPrefix)
	assert.Equal(t, "https://my.cool.tokens.com", actual.Spec.IssuerURL)
	assert.Equal(t, []authenticationv1alpha1.SigningAlgorithm{"RS256"}, actual.Spec.SupportedSigningAlgs)
	assert.Equal(t, ptr.To("sub"), actual.Spec.UsernameClaim)
	assert.Equal(t, ptr.To("-"), actual.Spec.UsernamePrefix)
	assert.Equal(t, map[string]string(nil), actual.Spec.RequiredClaims)
	assert.Equal(t, 0, len(actual.Spec.ExtraClaims))
	assert.Equal(t, 0, len(actual.Spec.CABundle))
	assert.Equal(t, authenticationv1alpha1.JWKSSpec{}, actual.Spec.JWKS)
	assert.Nil(t, actual.Spec.MaxTokenExpirationSeconds)
}

func runtimeForTest() imv1.Runtime {
	return imv1.Runtime{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-runtime",
			Namespace: "namespace",
		},
		Spec: imv1.RuntimeSpec{
			Shoot: imv1.RuntimeShoot{
				Name:     "test-shoot",
				Region:   "region",
				Provider: imv1.Provider{Type: "aws"},
			},
		},
	}
}

func shootForTest() *gardener.Shoot {
	return &gardener.Shoot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-shoot",
			Namespace: "namespace",
		},
		Spec: gardener.ShootSpec{
			Region:   "region",
			Provider: gardener.Provider{Type: "aws"}},
	}
}
