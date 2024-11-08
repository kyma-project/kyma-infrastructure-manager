package auditlogs

import (
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	core_v1 "k8s.io/api/core/v1"
)

func oSetPolicyConfigmap(policyConfigMapName string) operation {
	return func(s *gardener.Shoot) error {
		if s.Spec.Kubernetes.KubeAPIServer == nil {
			s.Spec.Kubernetes.KubeAPIServer = &gardener.KubeAPIServerConfig{}
		}

		s.Spec.Kubernetes.KubeAPIServer.AuditConfig = &gardener.AuditConfig{
			AuditPolicy: &gardener.AuditPolicy{
				ConfigMapRef: &core_v1.ObjectReference{Name: policyConfigMapName},
			},
		}

		return nil
	}
}
