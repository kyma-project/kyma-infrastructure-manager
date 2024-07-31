package extender

import (
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	v12 "k8s.io/api/core/v1"
)

func newAuditLogExtensionConfig(policyConfigMapName string) *gardener.AuditConfig {
	return &gardener.AuditConfig{
		AuditPolicy: &gardener.AuditPolicy{
			ConfigMapRef: &v12.ObjectReference{Name: policyConfigMapName},
		},
	}
}

func NewAuditLogExtender(policyConfigMapName string) func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
	return func(runtime imv1.Runtime, shoot *gardener.Shoot) error {

		if policyConfigMapName == "" {
			return nil
		}

		if shoot.Spec.Kubernetes.KubeAPIServer == nil {
			shoot.Spec.Kubernetes.KubeAPIServer = &gardener.KubeAPIServerConfig{}
		}

		shoot.Spec.Kubernetes.KubeAPIServer.AuditConfig = newAuditLogExtensionConfig(policyConfigMapName)

		return nil
	}
}
