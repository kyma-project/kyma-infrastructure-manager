package shoot

import (
	"fmt"
	"io"
	"strings"
	"testing"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/go-playground/validator/v10"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/config"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/hyperscaler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConverter(t *testing.T) {
	t.Run("Create shoot from Runtime", func(t *testing.T) {
		// given
		runtime := fixRuntime()
		converterConfig := fixConverterConfig()
		converter := NewConverterCreate(CreateOpts{ConverterConfig: converterConfig})

		// when
		shoot, err := converter.ToShoot(runtime)

		// then
		require.NoError(t, err)
		assertShootFields(t, runtime, shoot)
		assert.Equal(t, "1.28", shoot.Spec.Kubernetes.Version)
		assert.Equal(t, "gardenlinux", shoot.Spec.Provider.Workers[0].Machine.Image.Name)
		assert.Equal(t, "1591.1.0", *shoot.Spec.Provider.Workers[0].Machine.Image.Version)
	})

	t.Run("Create shoot with default converter config versions", func(t *testing.T) {
		// given
		runtime := fixRuntimeWithNoVersionsSpecified()
		converterConfig := fixConverterConfig()
		converter := NewConverterCreate(CreateOpts{ConverterConfig: converterConfig})

		// when
		shoot, err := converter.ToShoot(runtime)

		// then
		require.NoError(t, err)
		assertShootFields(t, runtime, shoot)
		assert.Equal(t, "1.29", shoot.Spec.Kubernetes.Version)
		assert.Equal(t, "gardenlinux", shoot.Spec.Provider.Workers[0].Machine.Image.Name)
		assert.Equal(t, "1592.1.0", *shoot.Spec.Provider.Workers[0].Machine.Image.Version)
	})

	t.Run("Create shoot from Runtime for existing shoot and keep versions", func(t *testing.T) {
		// given
		runtime := fixRuntime()
		converterConfig := fixConverterConfig()
		converter := NewConverterPatch(PatchOpts{
			ConverterConfig:   converterConfig,
			Zones:             fixReversedZones(),
			ShootK8SVersion:   "1.30",
			ShootImageName:    "gardenlinux",
			ShootImageVersion: "1592.2.0",
		})

		// when
		shoot, err := converter.ToShoot(runtime)

		// then
		require.NoError(t, err)
		assertShootFields(t, runtime, shoot)

		expectedZonesAreInSameOrder := []string{
			"eu-central-1c",
			"eu-central-1b",
			"eu-central-1a",
		}
		assert.Equal(t, expectedZonesAreInSameOrder, shoot.Spec.Provider.Workers[0].Zones)
		assert.Equal(t, expectedZonesAreInSameOrder, shoot.Spec.Provider.Workers[0].Zones)
		assert.Equal(t, "1.30", shoot.Spec.Kubernetes.Version)
		assert.Equal(t, "gardenlinux", shoot.Spec.Provider.Workers[0].Machine.Image.Name)
		assert.Equal(t, "1592.2.0", *shoot.Spec.Provider.Workers[0].Machine.Image.Version)
		assert.Nil(t, shoot.Spec.DNS)

		extensionLen := len(shoot.Spec.Extensions)
		require.Equalf(t, extensionLen, 0, "unexpected number of extensions: %d, expected: 0", extensionLen)
		// consider switchin to NotElementsMatch, whem released https://github.com/Antonboom/testifylint/issues/99
		for _, extension := range shoot.Spec.Extensions {
			assert.NotEqual(t, "shoot-dns-service", extension.Type, "unexpected immutable field extension: 'shoot-dns-service'")
		}
	})

	t.Run("Create shoot from Runtime for existing shoot and update versions", func(t *testing.T) {
		// given
		runtime := fixRuntime()
		converterConfig := fixConverterConfig()
		converter := NewConverterPatch(PatchOpts{
			ConverterConfig:   converterConfig,
			Zones:             fixReversedZones(),
			ShootK8SVersion:   "1.27",
			ShootImageName:    "gardenlinux",
			ShootImageVersion: "1591.0.0",
		})

		// when
		shoot, err := converter.ToShoot(runtime)

		// then
		require.NoError(t, err)
		assertShootFields(t, runtime, shoot)

		expectedZonesAreInSameOrder := []string{
			"eu-central-1c",
			"eu-central-1b",
			"eu-central-1a",
		}
		assert.Equal(t, expectedZonesAreInSameOrder, shoot.Spec.Provider.Workers[0].Zones)
		assert.Equal(t, expectedZonesAreInSameOrder, shoot.Spec.Provider.Workers[0].Zones)
		assert.Equal(t, "1.28", shoot.Spec.Kubernetes.Version)
		assert.Equal(t, "gardenlinux", shoot.Spec.Provider.Workers[0].Machine.Image.Name)
		assert.Equal(t, "1591.1.0", *shoot.Spec.Provider.Workers[0].Machine.Image.Version)
	})
}

func assertShootFields(t *testing.T, runtime imv1.Runtime, shoot gardener.Shoot) {
	assert.Equal(t, runtime.Spec.Shoot.Purpose, *shoot.Spec.Purpose)
	assert.Equal(t, runtime.Spec.Shoot.Region, shoot.Spec.Region)
	assert.Equal(t, runtime.Spec.Shoot.SecretBindingName, *shoot.Spec.SecretBindingName)
	assert.Equal(t, runtime.Spec.Shoot.ControlPlane, shoot.Spec.ControlPlane)
	assert.Equal(t, runtime.Spec.Shoot.Networking.Nodes, *shoot.Spec.Networking.Nodes)
	assert.Equal(t, runtime.Spec.Shoot.Networking.Pods, *shoot.Spec.Networking.Pods)
	assert.Equal(t, runtime.Spec.Shoot.Networking.Services, *shoot.Spec.Networking.Services)
	assert.Equal(t, "Shoot", shoot.TypeMeta.Kind)
	assert.Equal(t, "core.gardener.cloud/v1beta1", shoot.TypeMeta.APIVersion)
}

func fixReversedZones() []string {
	return []string{
		"eu-central-1c",
		"eu-central-1b",
		"eu-central-1a",
	}
}

func fixConverterConfig() config.ConverterConfig {
	return config.ConverterConfig{
		Kubernetes: config.KubernetesConfig{
			DefaultVersion:                      "1.29",
			EnableKubernetesVersionAutoUpdate:   true,
			EnableMachineImageVersionAutoUpdate: false,
		},
		DNS: config.DNSConfig{
			SecretName:   "dns-secret",
			DomainPrefix: "dev.mydomain.com",
			ProviderType: "aws-route53",
		},
		Provider: config.ProviderConfig{
			AWS: config.AWSConfig{
				EnableIMDSv2: true,
			},
		},
		MachineImage: config.MachineImageConfig{
			DefaultName:    "gardenlinux",
			DefaultVersion: "1592.1.0",
		},
	}
}

func fixRuntime() imv1.Runtime {
	kubernetesVersion := "1.28"
	clientID := "client-id"
	groupsClaim := "groups"
	issuerURL := "https://my.cool.tokens.com"
	usernameClaim := "sub"
	imageVersion := "1591.1.0"

	return imv1.Runtime{
		ObjectMeta: v1.ObjectMeta{
			Name:      "runtime",
			Namespace: "kcp-system",
		},
		Spec: imv1.RuntimeSpec{
			Shoot: imv1.RuntimeShoot{
				Purpose:           "production",
				Region:            "eu-central-1",
				SecretBindingName: "my-secret",
				Provider: imv1.Provider{
					Type: hyperscaler.TypeAWS,
					Workers: []gardener.Worker{
						{
							Name: "worker",
							Machine: gardener.Machine{
								Type: "m6i.large",
								Image: &gardener.ShootMachineImage{
									Name:    "gardenlinux",
									Version: &imageVersion,
								},
							},
							Minimum: 1,
							Maximum: 3,
							Zones: []string{
								"eu-central-1a",
								"eu-central-1b",
								"eu-central-1c",
							},
						},
					},
				},
				Kubernetes: imv1.Kubernetes{
					Version: &kubernetesVersion,
					KubeAPIServer: imv1.APIServer{
						OidcConfig: gardener.OIDCConfig{
							ClientID:    &clientID,
							GroupsClaim: &groupsClaim,
							IssuerURL:   &issuerURL,
							SigningAlgs: []string{
								"RS256",
							},
							UsernameClaim: &usernameClaim,
						},
					},
				},
				Networking: imv1.Networking{
					Pods:     "100.64.0.0/12",
					Nodes:    "10.250.0.0/16",
					Services: "100.104.0.0/13",
				},
				ControlPlane: &gardener.ControlPlane{
					HighAvailability: &gardener.HighAvailability{
						FailureTolerance: gardener.FailureTolerance{
							Type: gardener.FailureToleranceTypeZone,
						},
					},
				},
			},
		},
	}
}

func fixRuntimeWithNoVersionsSpecified() imv1.Runtime {
	clientID := "client-id"
	groupsClaim := "groups"
	issuerURL := "https://my.cool.tokens.com"
	usernameClaim := "sub"

	return imv1.Runtime{
		ObjectMeta: v1.ObjectMeta{
			Name:      "runtime",
			Namespace: "kcp-system",
		},
		Spec: imv1.RuntimeSpec{
			Shoot: imv1.RuntimeShoot{
				Purpose:           "production",
				Region:            "eu-central-1",
				SecretBindingName: "my-secret",
				Provider: imv1.Provider{
					Type: hyperscaler.TypeAWS,
					Workers: []gardener.Worker{
						{
							Name: "worker",
							Machine: gardener.Machine{
								Type: "m6i.large",
							},
							Minimum: 1,
							Maximum: 3,
							Zones: []string{
								"eu-central-1a",
								"eu-central-1b",
								"eu-central-1c",
							},
						},
					},
				},
				Kubernetes: imv1.Kubernetes{
					KubeAPIServer: imv1.APIServer{
						OidcConfig: gardener.OIDCConfig{
							ClientID:    &clientID,
							GroupsClaim: &groupsClaim,
							IssuerURL:   &issuerURL,
							SigningAlgs: []string{
								"RS256",
							},
							UsernameClaim: &usernameClaim,
						},
					},
				},
				Networking: imv1.Networking{
					Pods:     "100.64.0.0/12",
					Nodes:    "10.250.0.0/16",
					Services: "100.104.0.0/13",
				},
				ControlPlane: &gardener.ControlPlane{
					HighAvailability: &gardener.HighAvailability{
						FailureTolerance: gardener.FailureTolerance{
							Type: gardener.FailureToleranceTypeZone,
						},
					},
				},
			},
		},
	}
}

func Test_ConverterConfig_Load_Err(t *testing.T) {
	errTestReaderGetterFailed := fmt.Errorf("test reader getter fail")
	failingReaderGetter := func() (io.Reader, error) {
		return nil, errTestReaderGetterFailed
	}
	var cfg config.Config
	if err := cfg.Load(failingReaderGetter); err != errTestReaderGetterFailed {
		t.Error("ConverterConfig load should fail")
	}
}

var testReader io.Reader = strings.NewReader(
	`
{
  "cluster": {
	  "defaultSharedIASTenant" : {
		"clientID": "test-clientID",
		"groupsClaim": "test-group",
		"issuerURL": "test-issuer-url",
		"signingAlgs": ["test-alg"],
		"usernameClaim": "test-username-claim",
		"usernamePrefix": "-"
	  }
  },
  "converter": {
	  "kubernetes": {
		"defaultVersion": "0.1.2.3",
		"enableKubernetesVersionAutoUpdate": true,
		"enableMachineImageVersionAutoUpdate": false,
		"defaultOperatorOidc": {
			"clientID": "test-clientID",
			"groupsClaim": "test-group",
			"issuerURL": "test-issuer-url",
			"signingAlgs": ["test-alg"],
			"usernameClaim": "test-username-claim",
			"usernamePrefix": "-"
		},
		"defaultSharedIASTenant": {
			"clientID": "test-clientID",
			"groupsClaim": "test-group",
			"issuerURL": "test-issuer-url",
			"signingAlgs": ["test-alg"],
			"usernameClaim": "test-username-claim",
			"usernamePrefix": "-"
		}
	  },
	  "dns": {
		"secretName": "test-secret-name",
		"domainPrefix": "test-domain-prefix",
		"providerType": "test-provider-type"
	  },
	  "provider": {
		"aws": {
		  "enableIMDSv2": true
		}
	  },
	  "machineImage": {
		"defaultName": "test-image-name",
		"defaultVersion": "0.1.2.3.4"
	  },
	  "gardener": {
		"projectName": "test-project"
	  },
	  "auditLogging": {
		"policyConfigMapName": "test-policy",
		"tenantConfigPath": "test-path"
	  }
}
}`)

func Test_ConverterConfig_Load_OK(t *testing.T) {
	readerGetter := func() (io.Reader, error) {
		return testReader, nil
	}
	var cfg config.Config
	if err := cfg.Load(readerGetter); err != nil {
		t.Errorf("ConverterConfig load failed: %s", err)
	}

	expected := config.Config{
		ClusterConfig: config.ClusterConfig{
			DefaultSharedIASTenant: config.OidcProvider{
				ClientID:       "test-clientID",
				GroupsClaim:    "test-group",
				IssuerURL:      "test-issuer-url",
				SigningAlgs:    []string{"test-alg"},
				UsernameClaim:  "test-username-claim",
				UsernamePrefix: "-",
			},
		},
		ConverterConfig: config.ConverterConfig{
			Kubernetes: config.KubernetesConfig{
				DefaultVersion:                      "0.1.2.3",
				EnableKubernetesVersionAutoUpdate:   true,
				EnableMachineImageVersionAutoUpdate: false,
				DefaultOperatorOidc: config.OidcProvider{
					ClientID:       "test-clientID",
					GroupsClaim:    "test-group",
					IssuerURL:      "test-issuer-url",
					SigningAlgs:    []string{"test-alg"},
					UsernameClaim:  "test-username-claim",
					UsernamePrefix: "-",
				},
			},
			DNS: config.DNSConfig{
				SecretName:   "test-secret-name",
				DomainPrefix: "test-domain-prefix",
				ProviderType: "test-provider-type",
			},
			Provider: config.ProviderConfig{
				AWS: config.AWSConfig{
					EnableIMDSv2: true,
				},
			},
			MachineImage: config.MachineImageConfig{
				DefaultName:    "test-image-name",
				DefaultVersion: "0.1.2.3.4",
			},
			Gardener: config.GardenerConfig{
				ProjectName: "test-project",
			},
			AuditLog: config.AuditLogConfig{
				PolicyConfigMapName: "test-policy",
				TenantConfigPath:    "test-path",
			},
		},
	}
	assert.Equal(t, expected, cfg)

	validate := validator.New(validator.WithRequiredStructEnabled())
	assert.Nil(t, validate.Struct(cfg))
}
