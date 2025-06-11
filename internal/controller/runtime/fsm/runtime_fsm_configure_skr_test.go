package fsm

import (
	"context"
	core_v1 "k8s.io/api/core/v1"
	util "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"testing"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	authenticationv1alpha1 "github.com/gardener/oidc-webhook-authenticator/apis/authentication/v1alpha1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	imv1_client "github.com/kyma-project/infrastructure-manager/internal/controller/runtime/fsm/client"
	fsm_testing "github.com/kyma-project/infrastructure-manager/internal/controller/runtime/fsm/testing"
	"github.com/kyma-project/infrastructure-manager/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	api "k8s.io/apimachinery/pkg/runtime"
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
		stateFn, _, _ := sFnConfigureSKR(ctx, fsm, systemState)

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
			additionalOIDCConfig *[]imv1.OIDCConfig
		}{
			{"Should configure OIDC using defaults when additional OIDC config is nil", nil},
			{"Should configure OIDC using defaults when additional OIDC config contains empty array", &[]imv1.OIDCConfig{}},
			{"Should configure OIDC using defaults when additional OIDC config contains one empty element", &[]imv1.OIDCConfig{{}}},
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
				stateFn, _, _ := sFnConfigureSKR(ctx, testFsm, systemState)

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
		stateFn, _, _ := sFnConfigureSKR(ctx, testFsm, systemState)

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
		additionalOidcConfig := &[]imv1.OIDCConfig{}
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
		stateFn, _, _ := sFnConfigureSKR(ctx, testFsm, systemState)

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
		stateFn, _, _ := sFnConfigureSKR(ctx, testFSM, systemState)

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

	t.Run("Should apply kyma-provisioning-info config map", func(t *testing.T) {
		ctx := context.Background()
		testScheme := api.NewScheme()

		util.Must(imv1.AddToScheme(testScheme))
		util.Must(gardener.AddToScheme(testScheme))
		util.Must(core_v1.AddToScheme(testScheme))
		util.Must(authenticationv1alpha1.AddToScheme(testScheme))

		runtime := makeInputRuntimeWithAnnotation(map[string]string{"operator.kyma-project.io/existing-annotation": "true"})
		shootStub := fsm_testing.TestShootForPatch()
		oidcService := gardener.Extension{
			Type:     "shoot-oidc-service",
			Disabled: ptr.To(false),
		}
		shootStub.Spec.Extensions = append(shootStub.Spec.Extensions, oidcService)

		// start of fake client setup
		var fakeClient = fake.NewClientBuilder().
			WithScheme(testScheme).
			WithInterceptorFuncs(interceptor.Funcs{
				Patch: fsm_testing.GetFakePatchInterceptorForConfigMap(true),
			}).
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
		imv1_client.GetShootClientPatch = func(
			_ context.Context,
			_ client.Client,
			_ imv1.Runtime) (client.Client, error) {
			return fakeClient, nil
		}

		// end of fake client setup

		systemState := &systemState{
			instance: *runtime,
			shoot:    shootStub,
		}

		detailsConfigMap := &core_v1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kyma-provisioning-info",
				Namespace: "kyma-system",
			},
			Data: nil,
		}

		cmCreationErr := testFsm.Create(ctx, detailsConfigMap)
		assert.NoError(t, cmCreationErr)

		// when
		stateFn, _, _ := sFnConfigureSKR(ctx, testFsm, systemState)

		// then

		var detailsCM core_v1.ConfigMap
		key := client.ObjectKey{
			Name: "kyma-provisioning-info",
			Namespace: "kyma-system",
		}
		err := fakeClient.Get(ctx, key, &detailsCM)
		assert.NoError(t, err)
		assert.NotNil(t, detailsCM.Data)
		assert.NotNil(t, detailsCM.Data["details"])
		assert.Equal(t, detailsCM.Data["details"], "globalAccountID: global-account-id\ninfrastructureConfig:\n  apiVersion: aws.provider.extensions.gardener.cloud/v1alpha1\n  kind: InfrastructureConfig\n  networks:\n    vpc:\n      cidr: 10.250.0.0/22\n    zones:\n    - internal: 10.250.0.192/26\n      name: europe-west1-d\n      public: 10.250.0.128/26\n      workers: 10.250.0.0/25\nsubaccountID: subaccount-id\nworkerPools:\n  kyma:\n    autoScalerMax: 1\n    autoScalerMin: 1\n    haZones: false\n    machineType: m5.xlarge\n    name: test-worker\n")
		assert.Contains(t, stateFn.name(), "sFnApplyClusterRoleBindings")
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

func createGardenerOidcConfig(clientId string) imv1.OIDCConfig {
	return imv1.OIDCConfig{
		OIDCConfig: gardener.OIDCConfig{
			ClientID:       ptr.To(clientId),
			GroupsClaim:    ptr.To("groups"),
			IssuerURL:      ptr.To("https://my.cool.tokens.com"),
			SigningAlgs:    []string{"RS256"},
			UsernameClaim:  ptr.To("sub"),
			UsernamePrefix: ptr.To("-"),
		},
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
