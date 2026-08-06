package auditlogs

import (
	"testing"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/stretchr/testify/require"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_AuditlogExtenderExperimentalCfg(t *testing.T) {
	defaultPolicyConfigmapName := "default"
	for _, tc := range []struct {
		rt                 imv1.Runtime
		shoot              gardener.Shoot
		data               AuditLogData
		expectedRefMapName string
	}{
		{
			shoot: gardener.Shoot{},
			rt: imv1.Runtime{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"operator.kyma-project.io/experimental-audit-policy": "xxx",
					},
				},
			},
			data: AuditLogData{
				TenantID:   "tenant-id",
				ServiceURL: "testme",
			},
			expectedRefMapName: defaultPolicyConfigmapName,
		},
		{
			shoot: gardener.Shoot{},
			rt: imv1.Runtime{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"operator.kyma-project.io/experimental-audit-policy": "false",
					},
				},
			},
			data: AuditLogData{
				TenantID:   "tenant-id",
				ServiceURL: "testme",
			},
			expectedRefMapName: defaultPolicyConfigmapName,
		},
		{
			shoot: gardener.Shoot{},
			rt: imv1.Runtime{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"operator.kyma-project.io/experimental-audit-policy": "true",
					},
				},
			},
			data: AuditLogData{
				TenantID:   "tenant-id",
				ServiceURL: "testme",
			},
			expectedRefMapName: "experimental-audit-policy",
		},
		{
			shoot: gardener.Shoot{},
			data: AuditLogData{
				TenantID:   "tenant-id",
				ServiceURL: "testme",
			},
			expectedRefMapName: defaultPolicyConfigmapName,
		},
	} {
		// given
		extendWithAuditlogs := NewAuditlogExtender(defaultPolicyConfigmapName, tc.data)

		// when
		err := extendWithAuditlogs(tc.rt, &tc.shoot)

		// then
		require.NoError(t, err)

		// then
		require.Equal(t, tc.expectedRefMapName, tc.shoot.Spec.Kubernetes.KubeAPIServer.AuditConfig.AuditPolicy.ConfigMapRef.Name)
	}
}

func Test_AuditlogExtender(t *testing.T) {
	var zero imv1.Runtime
	for _, tc := range []struct {
		shoot               gardener.Shoot
		data                AuditLogData
		policyConfigmapName string
	}{
		{
			shoot: gardener.Shoot{},
			data: AuditLogData{
				TenantID:   "tenant-id",
				ServiceURL: "testme",
			},
		},
	} {
		// given
		extendWithAuditlogs := NewAuditlogExtender(tc.policyConfigmapName, tc.data)

		// when
		err := extendWithAuditlogs(zero, &tc.shoot)

		// then
		require.NoError(t, err)
	}
}

func Test_AuditlogExtender_ConfigurationUpdate(t *testing.T) {
	defaultPolicyConfigmapName := "audit-policy"

	testCases := []struct {
		name                    string
		initialShoot            gardener.Shoot
		runtime                 imv1.Runtime
		data                    AuditLogData
		applyTwice              bool
		expectedPolicyConfigMap string
		expectedSecretName      string
		expectedResourceCount   int
		verifyOtherResources    bool
	}{
		{
			name: "should add audit log configuration to shoot without audit log",
			initialShoot: gardener.Shoot{
				Spec: gardener.ShootSpec{
					Kubernetes: gardener.Kubernetes{},
				},
			},
			runtime: imv1.Runtime{},
			data: AuditLogData{
				TenantID:   "new-tenant-id",
				ServiceURL: "https://new-audit.example.com",
				SecretName: "new-audit-secret",
			},
			expectedPolicyConfigMap: defaultPolicyConfigmapName,
			expectedSecretName:      "new-audit-secret",
			expectedResourceCount:   1,
		},
		{
			name: "should update existing audit log secret reference",
			initialShoot: gardener.Shoot{
				Spec: gardener.ShootSpec{
					Kubernetes: gardener.Kubernetes{
						KubeAPIServer: &gardener.KubeAPIServerConfig{
							AuditConfig: &gardener.AuditConfig{
								AuditPolicy: &gardener.AuditPolicy{
									ConfigMapRef: &corev1.ObjectReference{Name: "audit-policy"},
								},
							},
						},
					},
					Resources: []gardener.NamedResourceReference{
						{
							Name: "auditlog-credentials",
							ResourceRef: autoscalingv1.CrossVersionObjectReference{
								Name:       "shared-audit-secret",
								Kind:       "Secret",
								APIVersion: "v1",
							},
						},
					},
				},
			},
			runtime: imv1.Runtime{},
			data: AuditLogData{
				TenantID:   "dedicated-tenant-id",
				ServiceURL: "https://dedicated-audit.example.com",
				SecretName: "dedicated-audit-secret",
			},
			expectedPolicyConfigMap: defaultPolicyConfigmapName,
			expectedSecretName:      "dedicated-audit-secret",
			expectedResourceCount:   1,
		},
		{
			name: "should update secret reference when upgrading from missing shared config to dedicated",
			initialShoot: gardener.Shoot{
				Spec: gardener.ShootSpec{
					Kubernetes: gardener.Kubernetes{},
					Resources:  []gardener.NamedResourceReference{},
				},
			},
			runtime: imv1.Runtime{},
			data: AuditLogData{
				TenantID:   "dedicated-tenant",
				ServiceURL: "https://dedicated-audit.example.com",
				SecretName: "dedicated-secret",
			},
			expectedPolicyConfigMap: defaultPolicyConfigmapName,
			expectedSecretName:      "dedicated-secret",
			expectedResourceCount:   1,
		},
		{
			name: "should be idempotent when called multiple times with same configuration",
			initialShoot: gardener.Shoot{
				Spec: gardener.ShootSpec{
					Kubernetes: gardener.Kubernetes{},
				},
			},
			runtime: imv1.Runtime{},
			data: AuditLogData{
				TenantID:   "tenant-id",
				ServiceURL: "https://audit.example.com",
				SecretName: "audit-secret",
			},
			applyTwice:              true,
			expectedPolicyConfigMap: defaultPolicyConfigmapName,
			expectedSecretName:      "audit-secret",
			expectedResourceCount:   1,
		},
		{
			name: "should preserve other resources when updating audit log secret",
			initialShoot: gardener.Shoot{
				Spec: gardener.ShootSpec{
					Kubernetes: gardener.Kubernetes{},
					Resources: []gardener.NamedResourceReference{
						{
							Name: "other-resource",
							ResourceRef: autoscalingv1.CrossVersionObjectReference{
								Name:       "other-secret",
								Kind:       "Secret",
								APIVersion: "v1",
							},
						},
						{
							Name: "auditlog-credentials",
							ResourceRef: autoscalingv1.CrossVersionObjectReference{
								Name:       "old-audit-secret",
								Kind:       "Secret",
								APIVersion: "v1",
							},
						},
					},
				},
			},
			runtime: imv1.Runtime{},
			data: AuditLogData{
				TenantID:   "tenant-id",
				ServiceURL: "https://audit.example.com",
				SecretName: "new-audit-secret",
			},
			expectedPolicyConfigMap: defaultPolicyConfigmapName,
			expectedSecretName:      "new-audit-secret",
			expectedResourceCount:   2,
			verifyOtherResources:    true,
		},
		{
			name: "should update policy configmap when experimental annotation is set",
			initialShoot: gardener.Shoot{
				Spec: gardener.ShootSpec{
					Kubernetes: gardener.Kubernetes{
						KubeAPIServer: &gardener.KubeAPIServerConfig{
							AuditConfig: &gardener.AuditConfig{
								AuditPolicy: &gardener.AuditPolicy{
									ConfigMapRef: &corev1.ObjectReference{Name: "audit-policy"},
								},
							},
						},
					},
				},
			},
			runtime: imv1.Runtime{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"operator.kyma-project.io/experimental-audit-policy": "true",
					},
				},
			},
			data: AuditLogData{
				TenantID:   "tenant-id",
				ServiceURL: "https://audit.example.com",
				SecretName: "audit-secret",
			},
			expectedPolicyConfigMap: "experimental-audit-policy",
			expectedSecretName:      "audit-secret",
			expectedResourceCount:   1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			shoot := tc.initialShoot.DeepCopy()
			extendWithAuditlogs := NewAuditlogExtender(defaultPolicyConfigmapName, tc.data)

			// when
			err := extendWithAuditlogs(tc.runtime, shoot)
			require.NoError(t, err)

			if tc.applyTwice {
				err = extendWithAuditlogs(tc.runtime, shoot)
				require.NoError(t, err)
			}

			// then - verify policy configmap
			require.NotNil(t, shoot.Spec.Kubernetes.KubeAPIServer)
			require.NotNil(t, shoot.Spec.Kubernetes.KubeAPIServer.AuditConfig)
			require.NotNil(t, shoot.Spec.Kubernetes.KubeAPIServer.AuditConfig.AuditPolicy)
			require.Equal(t, tc.expectedPolicyConfigMap, shoot.Spec.Kubernetes.KubeAPIServer.AuditConfig.AuditPolicy.ConfigMapRef.Name)

			// verify resource count
			require.Len(t, shoot.Spec.Resources, tc.expectedResourceCount)

			// verify audit log secret reference
			var auditLogResource *gardener.NamedResourceReference
			for i := range shoot.Spec.Resources {
				if shoot.Spec.Resources[i].Name == "auditlog-credentials" {
					auditLogResource = &shoot.Spec.Resources[i]
					break
				}
			}
			require.NotNil(t, auditLogResource, "auditlog-credentials resource not found")
			require.Equal(t, tc.expectedSecretName, auditLogResource.ResourceRef.Name)

			// verify other resources are preserved (if applicable)
			if tc.verifyOtherResources {
				var otherResource *gardener.NamedResourceReference
				for i := range shoot.Spec.Resources {
					if shoot.Spec.Resources[i].Name == "other-resource" {
						otherResource = &shoot.Spec.Resources[i]
						break
					}
				}
				require.NotNil(t, otherResource, "other-resource should be preserved")
				require.Equal(t, "other-secret", otherResource.ResourceRef.Name)
			}
		})
	}
}
