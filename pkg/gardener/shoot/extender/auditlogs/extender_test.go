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
		name                        string
		initialShoot                gardener.Shoot
		runtime                     imv1.Runtime
		data                        AuditLogData
		secondData                  AuditLogData
		applyTwice                  bool
		expectedPolicyConfigMap     string
		expectedSharedSecretName    string
		expectedDedicatedSecretName string
		expectedResourceCount       int
		verifyOtherResources        bool
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
				Dedicated:  false,
			},
			expectedPolicyConfigMap:     defaultPolicyConfigmapName,
			expectedSharedSecretName:    "new-audit-secret",
			expectedDedicatedSecretName: "",
			expectedResourceCount:       1,
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
				TenantID:   "shared-tenant-id",
				ServiceURL: "https://shared-audit.example.com",
				SecretName: "new-shared-secret",
				Dedicated:  false,
			},
			expectedPolicyConfigMap:     defaultPolicyConfigmapName,
			expectedSharedSecretName:    "new-shared-secret",
			expectedDedicatedSecretName: "",
			expectedResourceCount:       1,
		},
		{
			name: "should update existing dedicated audit log secret reference and freeze pre-existing shared reference",
			initialShoot: gardener.Shoot{
				Spec: gardener.ShootSpec{
					Kubernetes: gardener.Kubernetes{},
					Resources: []gardener.NamedResourceReference{
						{
							Name: "dedicated-auditlog-credentials",
							ResourceRef: autoscalingv1.CrossVersionObjectReference{
								Name:       "old-dedicated-secret",
								Kind:       "Secret",
								APIVersion: "v1",
							},
						},
						{
							Name: "auditlog-credentials",
							ResourceRef: autoscalingv1.CrossVersionObjectReference{
								Name:       "old-shared-secret",
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
				SecretName: "new-dedicated-secret",
				Dedicated:  true,
			},
			expectedPolicyConfigMap:     defaultPolicyConfigmapName,
			expectedSharedSecretName:    "old-shared-secret",
			expectedDedicatedSecretName: "new-dedicated-secret",
			expectedResourceCount:       2,
		},
		{
			name: "should create shared reference once with fallback secret name when upgrading from missing config to dedicated",
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
				Dedicated:  true,
			},
			expectedPolicyConfigMap:     defaultPolicyConfigmapName,
			expectedSharedSecretName:    "shared-secret",
			expectedDedicatedSecretName: "dedicated-secret",
			expectedResourceCount:       2,
		},
		{
			name: "should freeze shared reference on subsequent calls with new secret name after initial dedicated creation",
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
				SecretName: "initial-secret",
				Dedicated:  true,
			},
			secondData: AuditLogData{
				TenantID:   "dedicated-tenant",
				ServiceURL: "https://dedicated-audit.example.com",
				SecretName: "rotated-secret",
				Dedicated:  true,
			},
			expectedPolicyConfigMap:     defaultPolicyConfigmapName,
			expectedSharedSecretName:    "initial-secret",
			expectedDedicatedSecretName: "rotated-secret",
			expectedResourceCount:       2,
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
				Dedicated:  false,
			},
			applyTwice:                  true,
			expectedPolicyConfigMap:     defaultPolicyConfigmapName,
			expectedSharedSecretName:    "audit-secret",
			expectedDedicatedSecretName: "",
			expectedResourceCount:       1,
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
				Dedicated:  false,
			},
			expectedPolicyConfigMap:     defaultPolicyConfigmapName,
			expectedSharedSecretName:    "new-audit-secret",
			expectedDedicatedSecretName: "",
			expectedResourceCount:       2,
			verifyOtherResources:        true,
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
				Dedicated:  false,
			},
			expectedPolicyConfigMap:     "experimental-audit-policy",
			expectedSharedSecretName:    "audit-secret",
			expectedDedicatedSecretName: "",
			expectedResourceCount:       1,
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

			if tc.secondData != (AuditLogData{}) {
				secondExtend := NewAuditlogExtender(defaultPolicyConfigmapName, tc.secondData)
				err = secondExtend(tc.runtime, shoot)
				require.NoError(t, err)
			} else if tc.applyTwice {
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

			// verify audit log secret references
			var sharedAuditLogResource *gardener.NamedResourceReference
			var dedicatedAuditLogResource *gardener.NamedResourceReference
			for i := range shoot.Spec.Resources {
				if shoot.Spec.Resources[i].Name == SharedAuditlogSecretReference {
					sharedAuditLogResource = &shoot.Spec.Resources[i]
				}
				if shoot.Spec.Resources[i].Name == DedicatedAuditlogSecretReference {
					dedicatedAuditLogResource = &shoot.Spec.Resources[i]
				}
			}

			if tc.expectedSharedSecretName != "" {
				require.NotNil(t, sharedAuditLogResource, "auditlog-credentials resource not found")
				require.Equal(t, tc.expectedSharedSecretName, sharedAuditLogResource.ResourceRef.Name)
			} else {
				require.Nil(t, sharedAuditLogResource, "auditlog-credentials resource should not exist")
			}

			if tc.expectedDedicatedSecretName != "" {
				require.NotNil(t, dedicatedAuditLogResource, "dedicated-auditlog-credentials resource not found")
				require.Equal(t, tc.expectedDedicatedSecretName, dedicatedAuditLogResource.ResourceRef.Name)
			} else {
				require.Nil(t, dedicatedAuditLogResource, "dedicated-auditlog-credentials resource should not exist")
			}

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
