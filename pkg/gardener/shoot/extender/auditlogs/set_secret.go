package auditlogs

import (
	"slices"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1 "k8s.io/api/autoscaling/v1"
)

func oSetSecret(referenceName, secretName string) operation {
	return func(s *gardener.Shoot) error {
		otherReferenceName := SharedAuditlogSecretReference
		if referenceName == SharedAuditlogSecretReference {
			otherReferenceName = DedicatedAuditlogSecretReference
		}

		s.Spec.Resources = slices.DeleteFunc(s.Spec.Resources, func(r gardener.NamedResourceReference) bool {
			return r.Name == otherReferenceName
		})

		resource := gardener.NamedResourceReference{
			Name: referenceName,
			ResourceRef: v1.CrossVersionObjectReference{
				Name:       secretName,
				Kind:       "Secret",
				APIVersion: "v1",
			},
		}
		index := slices.IndexFunc(s.Spec.Resources, func(r gardener.NamedResourceReference) bool {
			return r.Name == referenceName
		})

		if index == -1 {
			s.Spec.Resources = append(s.Spec.Resources, resource)
			return nil
		}

		s.Spec.Resources[index] = resource
		return nil
	}
}
