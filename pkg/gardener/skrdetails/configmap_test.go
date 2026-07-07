package skrdetails

import (
	"testing"
	"time"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
)

func TestToKymaProvisioningInfo(t *testing.T) {
	lastReconcileTime := metav1.Time{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	t.Run("Should include environmentInstanceID and instanceName from labels", func(t *testing.T) {
		// given
		runtimeCR := imv1.Runtime{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-runtime",
				Namespace: "kcp-system",
				Labels: map[string]string{
					imv1.LabelKymaInstanceID:      "test-instance-id",
					imv1.LabelKymaName:            "test-kyma-name",
					imv1.LabelKymaGlobalAccountID: "test-global-account-id",
					imv1.LabelKymaSubaccountID:    "test-subaccount-id",
				},
			},
			Spec: imv1.RuntimeSpec{
				Shoot: imv1.RuntimeShoot{
					Name:           "test-shoot",
					Region:         "eu-central-1",
					PlatformRegion: "cf-eu12",
					Provider: imv1.Provider{
						Type: "aws",
						Workers: []gardener.Worker{
							{
								Name: "worker-1",
								Machine: gardener.Machine{
									Type: "m5.xlarge",
								},
								Minimum: 1,
								Maximum: 3,
								Zones:   []string{"eu-west-1a"},
							},
						},
					},
				},
			},
			Status: imv1.RuntimeStatus{
				ShootLastOperation: &gardener.LastOperation{
					LastUpdateTime: lastReconcileTime,
				},
			},
		}

		shoot := &gardener.Shoot{
			Spec: gardener.ShootSpec{
				Provider: gardener.Provider{
					InfrastructureConfig: &apiruntime.RawExtension{
						Raw: []byte(`{"apiVersion":"aws.provider.extensions.gardener.cloud/v1alpha1","kind":"InfrastructureConfig"}`),
					},
				},
			},
		}

		seed := gardener.Seed{
			Spec: gardener.SeedSpec{
				Provider: gardener.SeedProvider{
					Region: "eu-west-1",
				},
			},
		}

		// when
		result := ToKymaProvisioningInfo(runtimeCR, shoot, seed)

		// then
		assert.Equal(t, "test-instance-id", result.EnvironmentInstanceID)
		assert.Equal(t, "test-kyma-name", result.InstanceName)
		assert.Equal(t, "test-global-account-id", result.GlobalAccountID)
		assert.Equal(t, "test-subaccount-id", result.SubaccountID)
		assert.Equal(t, "eu-central-1", result.Region)
		assert.Equal(t, "cf-eu12", result.PlatformRegion)
		assert.Equal(t, "eu-west-1", result.SeedRegion)
		assert.Equal(t, lastReconcileTime, result.LastReconcileTime)
	})

	t.Run("Should handle missing labels gracefully", func(t *testing.T) {

		// given
		runtimeCR := imv1.Runtime{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-runtime",
				Namespace: "kcp-system",
				Labels:    map[string]string{},
			},
			Spec: imv1.RuntimeSpec{
				Shoot: imv1.RuntimeShoot{
					Name: "test-shoot",
					Provider: imv1.Provider{
						Type:    "aws",
						Workers: []gardener.Worker{},
					},
				},
			},
			Status: imv1.RuntimeStatus{
				ShootLastOperation: &gardener.LastOperation{
					LastUpdateTime: metav1.Time{},
				},
			},
		}

		shoot := &gardener.Shoot{
			Spec: gardener.ShootSpec{
				Provider: gardener.Provider{
					InfrastructureConfig: &apiruntime.RawExtension{
						Raw: []byte(`{}`),
					},
				},
			},
		}

		seed := gardener.Seed{
			Spec: gardener.SeedSpec{
				Provider: gardener.SeedProvider{
					Region: "",
				},
			},
		}

		// when
		result := ToKymaProvisioningInfo(runtimeCR, shoot, seed)

		// then
		assert.Empty(t, result.EnvironmentInstanceID)
		assert.Empty(t, result.InstanceName)
		assert.Empty(t, result.GlobalAccountID)
		assert.Empty(t, result.SubaccountID)
		assert.Empty(t, result.SeedRegion)
		assert.Empty(t, result.LastReconcileTime)
	})
}

func TestToKymaProvisioningInfoWithACL(t *testing.T) {
	t.Run("Should include ACL when allowedCIDRs is set", func(t *testing.T) {
		// given
		runtimeCR := imv1.Runtime{
			Spec: imv1.RuntimeSpec{
				Shoot: imv1.RuntimeShoot{
					Name: "test-shoot",
					Provider: imv1.Provider{
						Type: "aws",
						Workers: []gardener.Worker{
							{
								Name: "worker-1",
								Machine: gardener.Machine{
									Type: "m5.xlarge",
								},
								Minimum: 1,
								Maximum: 3,
								Zones:   []string{"eu-west-1a"},
							},
						},
					},
					Kubernetes: imv1.Kubernetes{
						KubeAPIServer: imv1.APIServer{
							ACL: &imv1.ACL{
								AllowedCIDRs: []string{"10.0.0.0/8", "192.168.0.0/16"},
							},
						},
					},
				},
			},
			Status: imv1.RuntimeStatus{
				ShootLastOperation: &gardener.LastOperation{
					LastUpdateTime: metav1.Time{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
				},
			},
		}

		shoot := &gardener.Shoot{
			Spec: gardener.ShootSpec{
				Provider: gardener.Provider{
					InfrastructureConfig: &apiruntime.RawExtension{
						Raw: []byte(`{}`),
					},
				},
			},
		}

		seed := gardener.Seed{
			Spec: gardener.SeedSpec{
				Provider: gardener.SeedProvider{
					Region: "eu-west-1",
				},
			},
		}
		// when
		result := ToKymaProvisioningInfo(runtimeCR, shoot, seed)

		// then
		assert.Equal(t, []string{"10.0.0.0/8", "192.168.0.0/16"}, result.NetworkDetails.KubeAPIServer.ACL)
	})

	t.Run("Should omit ACL field when allowedCIDRs is empty array", func(t *testing.T) {
		// given
		runtimeCR := imv1.Runtime{
			Spec: imv1.RuntimeSpec{
				Shoot: imv1.RuntimeShoot{
					Name: "test-shoot",
					Provider: imv1.Provider{
						Type: "aws",
						Workers: []gardener.Worker{
							{
								Name: "worker-1",
								Machine: gardener.Machine{
									Type: "m5.xlarge",
								},
								Minimum: 1,
								Maximum: 3,
								Zones:   []string{"eu-west-1a"},
							},
						},
					},
					Kubernetes: imv1.Kubernetes{
						KubeAPIServer: imv1.APIServer{
							ACL: &imv1.ACL{
								AllowedCIDRs: []string{},
							},
						},
					},
				},
			},
			Status: imv1.RuntimeStatus{
				ShootLastOperation: &gardener.LastOperation{
					LastUpdateTime: metav1.Time{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
				},
			},
		}

		shoot := &gardener.Shoot{
			Spec: gardener.ShootSpec{
				Provider: gardener.Provider{
					InfrastructureConfig: &apiruntime.RawExtension{
						Raw: []byte(`{}`),
					},
				},
			},
		}

		seed := gardener.Seed{
			Spec: gardener.SeedSpec{
				Provider: gardener.SeedProvider{
					Region: "eu-west-1",
				},
			},
		}

		// when
		result := ToKymaProvisioningInfo(runtimeCR, shoot, seed)

		// then
		assert.Nil(t, result.NetworkDetails.KubeAPIServer.ACL)
		assert.Empty(t, result.NetworkDetails.KubeAPIServer.ACL)
	})

	t.Run("Should omit ACL field when ACL is nil", func(t *testing.T) {
		// given
		runtimeCR := imv1.Runtime{
			Spec: imv1.RuntimeSpec{
				Shoot: imv1.RuntimeShoot{
					Name: "test-shoot",
					Provider: imv1.Provider{
						Type: "aws",
						Workers: []gardener.Worker{
							{
								Name: "worker-1",
								Machine: gardener.Machine{
									Type: "m5.xlarge",
								},
								Minimum: 1,
								Maximum: 3,
								Zones:   []string{"eu-west-1a"},
							},
						},
					},
					Kubernetes: imv1.Kubernetes{
						KubeAPIServer: imv1.APIServer{
							ACL: nil,
						},
					},
				},
			},
			Status: imv1.RuntimeStatus{
				ShootLastOperation: &gardener.LastOperation{
					LastUpdateTime: metav1.Time{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
				},
			},
		}

		shoot := &gardener.Shoot{
			Spec: gardener.ShootSpec{
				Provider: gardener.Provider{
					InfrastructureConfig: &apiruntime.RawExtension{
						Raw: []byte(`{}`),
					},
				},
			},
		}

		seed := gardener.Seed{
			Spec: gardener.SeedSpec{
				Provider: gardener.SeedProvider{
					Region: "eu-west-1",
				},
			},
		}

		// when
		result := ToKymaProvisioningInfo(runtimeCR, shoot, seed)

		// then
		assert.Nil(t, result.NetworkDetails.KubeAPIServer.ACL)
		assert.Empty(t, result.NetworkDetails.KubeAPIServer.ACL)
	})
}

func TestToKymaProvisioningInfoConfigMap(t *testing.T) {
	lastReconcileTime := metav1.Time{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	t.Run("Should create ConfigMap with all fields including environmentInstanceID and instanceName", func(t *testing.T) {
		// given
		runtimeCR := imv1.Runtime{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-runtime",
				Namespace: "kcp-system",
				Labels: map[string]string{
					imv1.LabelKymaInstanceID:      "env-instance-123",
					imv1.LabelKymaName:            "my-kyma-instance",
					imv1.LabelKymaGlobalAccountID: "global-acc-456",
					imv1.LabelKymaSubaccountID:    "sub-acc-789",
				},
			},
			Spec: imv1.RuntimeSpec{
				Shoot: imv1.RuntimeShoot{
					Name:           "test-shoot",
					Region:         "eu-central-1",
					PlatformRegion: "cf-eu12",
					Provider: imv1.Provider{
						Type: "aws",
						Workers: []gardener.Worker{
							{
								Name: "worker-1",
								Machine: gardener.Machine{
									Type: "m5.xlarge",
								},
								Minimum: 1,
								Maximum: 3,
								Zones:   []string{"eu-west-1a"},
							},
						},
					},
				},
			},
			Status: imv1.RuntimeStatus{
				ShootLastOperation: &gardener.LastOperation{
					LastUpdateTime: lastReconcileTime,
				},
			},
		}

		shoot := &gardener.Shoot{
			Spec: gardener.ShootSpec{
				Provider: gardener.Provider{
					InfrastructureConfig: &apiruntime.RawExtension{
						Raw: []byte(`{"apiVersion":"aws.provider.extensions.gardener.cloud/v1alpha1","kind":"InfrastructureConfig"}`),
					},
				},
			},
		}

		seed := gardener.Seed{
			Spec: gardener.SeedSpec{
				Provider: gardener.SeedProvider{
					Region: "eu-west-1",
				},
			},
		}

		// when
		cm, err := ToKymaProvisioningInfoConfigMap(runtimeCR, shoot, &seed)

		// then
		require.NoError(t, err)
		assert.Equal(t, "kyma-provisioning-info", cm.Name)
		assert.Equal(t, "kyma-system", cm.Namespace)
		assert.Contains(t, cm.Data["details"], "environmentInstanceID: env-instance-123")
		assert.Contains(t, cm.Data["details"], "instanceName: my-kyma-instance")
		assert.Contains(t, cm.Data["details"], "globalAccountID: global-acc-456")
		assert.Contains(t, cm.Data["details"], "subaccountID: sub-acc-789")
		assert.Contains(t, cm.Data["details"], "region: eu-central-1")
		assert.Contains(t, cm.Data["details"], "platformRegion: cf-eu12")
		assert.Contains(t, cm.Data["details"], "lastReconcileTime", lastReconcileTime)
	})
}
