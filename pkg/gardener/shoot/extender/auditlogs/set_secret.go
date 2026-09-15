package auditlogs

import (
	"slices"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1 "k8s.io/api/autoscaling/v1"
)

// UpsertAuditLogSecretReferences adds or updates the NamedResourceReferences in shoot.spec.resources
// for audit log secret references.
//
// In shared mode (dedicated = false), it upserts only SharedAuditlogSecretReference ("auditlog-credentials").
// In dedicated mode (dedicated = true), it always upserts DedicatedAuditlogSecretReference ("dedicated-auditlog-credentials")
// with the current secret, and only creates SharedAuditlogSecretReference if missing.
//
// SharedAuditlogSecretReference is created if missing and left unchanged if already present because Gardener's
// shoot-auditlog-service admission webhook rejects shoots if "auditlog-credentials" is missing from resources,
// even when the extension points to "dedicated-auditlog-credentials".
func UpsertAuditLogSecretReferences(s *gardener.Shoot, dedicated bool, secretName string) {
	if !dedicated {
		upsertNamedResourceReference(s, SharedAuditlogSecretReference, secretName)
		return
	}

	upsertNamedResourceReference(s, DedicatedAuditlogSecretReference, secretName)
	createIfMissingNamedResourceReference(s, SharedAuditlogSecretReference, secretName)
}

func oSetSecret(dedicated bool, secretName string) operation {
	return func(s *gardener.Shoot) error {
		UpsertAuditLogSecretReferences(s, dedicated, secretName)
		return nil
	}
}

func upsertNamedResourceReference(s *gardener.Shoot, referenceName, secretName string) {
	resource := newSecretResourceReference(referenceName, secretName)
	index := findResourceIndex(s, referenceName)

	if index == -1 {
		s.Spec.Resources = append(s.Spec.Resources, resource)
		return
	}

	s.Spec.Resources[index] = resource
}

func createIfMissingNamedResourceReference(s *gardener.Shoot, referenceName, secretName string) {
	if findResourceIndex(s, referenceName) != -1 {
		return
	}

	s.Spec.Resources = append(s.Spec.Resources, newSecretResourceReference(referenceName, secretName))
}

func findResourceIndex(s *gardener.Shoot, referenceName string) int {
	return slices.IndexFunc(s.Spec.Resources, func(r gardener.NamedResourceReference) bool {
		return r.Name == referenceName
	})
}

func newSecretResourceReference(referenceName, secretName string) gardener.NamedResourceReference {
	return gardener.NamedResourceReference{
		Name: referenceName,
		ResourceRef: v1.CrossVersionObjectReference{
			Name:       secretName,
			Kind:       "Secret",
			APIVersion: "v1",
		},
	}
}
