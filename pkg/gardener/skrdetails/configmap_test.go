package skrdetails

import (
	"testing"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
)

func TestToKymaProvisioningInfo(t *testing.T) {
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

		// when
		result := ToKymaProvisioningInfo(runtimeCR, shoot)

		// then
		assert.Equal(t, "test-instance-id", result.EnvironmentInstanceID)
		assert.Equal(t, "test-kyma-name", result.InstanceName)
		assert.Equal(t, "test-global-account-id", result.GlobalAccountID)
		assert.Equal(t, "test-subaccount-id", result.SubaccountID)
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

		// when
		result := ToKymaProvisioningInfo(runtimeCR, shoot)

		// then
		assert.Empty(t, result.EnvironmentInstanceID)
		assert.Empty(t, result.InstanceName)
		assert.Empty(t, result.GlobalAccountID)
		assert.Empty(t, result.SubaccountID)
	})
}

func TestToKymaProvisioningInfoConfigMap(t *testing.T) {
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

		// when
		cm, err := ToKymaProvisioningInfoConfigMap(runtimeCR, shoot)

		// then
		require.NoError(t, err)
		assert.Equal(t, "kyma-provisioning-info", cm.Name)
		assert.Equal(t, "kyma-system", cm.Namespace)
		assert.Contains(t, cm.Data["details"], "environmentInstanceID: env-instance-123")
		assert.Contains(t, cm.Data["details"], "instanceName: my-kyma-instance")
		assert.Contains(t, cm.Data["details"], "globalAccountID: global-acc-456")
		assert.Contains(t, cm.Data["details"], "subaccountID: sub-acc-789")
	})
}
