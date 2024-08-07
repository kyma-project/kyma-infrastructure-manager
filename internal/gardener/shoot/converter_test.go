package shoot

import (
	"fmt"
	"io"
	"strings"
	"testing"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/go-playground/validator/v10"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/gardener/shoot/hyperscaler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConverter(t *testing.T) {
	t.Run("Create shoot from Runtime", func(t *testing.T) {
		// given
		runtime := fixRuntime()
		converterConfig := fixConverterConfig()
		converter := NewConverter(converterConfig)

		// when
		shoot, err := converter.ToShoot(runtime)

		// then
		require.NoError(t, err)
		assert.Equal(t, runtime.Spec.Shoot.Purpose, *shoot.Spec.Purpose)
		assert.Equal(t, runtime.Spec.Shoot.Region, shoot.Spec.Region)
		assert.Equal(t, runtime.Spec.Shoot.SecretBindingName, *shoot.Spec.SecretBindingName)
		assert.Equal(t, runtime.Spec.Shoot.ControlPlane, *shoot.Spec.ControlPlane)
		assert.Equal(t, runtime.Spec.Shoot.Networking.Nodes, *shoot.Spec.Networking.Nodes)
		assert.Equal(t, runtime.Spec.Shoot.Networking.Pods, *shoot.Spec.Networking.Pods)
		assert.Equal(t, runtime.Spec.Shoot.Networking.Services, *shoot.Spec.Networking.Services)
	})
}

func fixConverterConfig() ConverterConfig {
	return ConverterConfig{
		Kubernetes: KubernetesConfig{
			DefaultVersion:                      "1.29",
			EnableKubernetesVersionAutoUpdate:   true,
			EnableMachineImageVersionAutoUpdate: false,
		},
		DNS: DNSConfig{
			SecretName:   "dns-secret",
			DomainPrefix: "dev.mydomain.com",
			ProviderType: "aws-route53",
		},
		Provider: ProviderConfig{
			AWS: AWSConfig{
				EnableIMDSv2: true,
			},
		},
	}
}

func fixRuntime() imv1.Runtime {
	kubernetesVersion := "1.28"
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
				ControlPlane: gardener.ControlPlane{
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
	var cfg ConverterConfig
	if err := cfg.Load(failingReaderGetter); err != errTestReaderGetterFailed {
		t.Error("ConverterConfig load should fail")
	}
}

var testReader io.Reader = strings.NewReader(`{
  "kubernetes": {
    "defaultVersion": "0.1.2.3",
    "enableKubernetesVersionAutoUpdate": true,
    "enableMachineImageVersionAutoUpdate": false
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
    "defaultVersion": "0.1.2.3.4"
  },
  "gardener": {
    "projectName": "test-project"
  }
}`)

func Test_ConverterConfig_Load_OK(t *testing.T) {
	readerGetter := func() (io.Reader, error) {
		return testReader, nil
	}
	var cfg ConverterConfig
	if err := cfg.Load(readerGetter); err != nil {
		t.Errorf("ConverterConfig load failed: %s", err)
	}

	expected := ConverterConfig{
		Kubernetes: KubernetesConfig{
			DefaultVersion:                      "0.1.2.3",
			EnableKubernetesVersionAutoUpdate:   true,
			EnableMachineImageVersionAutoUpdate: false,
		},
		DNS: DNSConfig{
			SecretName:   "test-secret-name",
			DomainPrefix: "test-domain-prefix",
			ProviderType: "test-provider-type",
		},
		Provider: ProviderConfig{
			AWS: AWSConfig{
				EnableIMDSv2: true,
			},
		},
		MachineImage: MachineImageConfig{
			DefaultVersion: "0.1.2.3.4",
		},
		Gardener: GardenerConfig{
			ProjectName: "test-project",
		},
	}
	assert.Equal(t, expected, cfg)

	validate := validator.New(validator.WithRequiredStructEnabled())
	assert.Nil(t, validate.Struct(cfg))
}
