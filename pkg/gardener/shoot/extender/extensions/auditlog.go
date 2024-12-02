package extensions

import (
	"bytes"
	"encoding/json"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/auditlogs"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	auditlogExtensionType = "shoot-auditlog-service"
	auditlogReferenceName = "auditlog-credentials"
)

type AuditlogExtensionConfig struct {
	metav1.TypeMeta `json:",inline"`
	// Type is the type of auditlog service provider.
	Type string `json:"type"`
	// TenantID is the id of the tenant.
	TenantID string `json:"tenantID"`
	// ServiceURL is the URL of the auditlog service.
	ServiceURL string `json:"serviceURL"`
	// SecretReferenceName is the name of the reference for the secret containing the auditlog service credentials.
	SecretReferenceName string `json:"secretReferenceName"`
}

func NewAuditLogExtension(d auditlogs.AuditLogData) (*gardener.Extension, error) {
	cfg := AuditlogExtensionConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AuditlogConfig",
			APIVersion: "service.auditlog.extensions.gardener.cloud/v1alpha1",
		},
		Type:                "standard",
		TenantID:            d.TenantID,
		ServiceURL:          d.ServiceURL,
		SecretReferenceName: auditlogReferenceName,
	}
	var buffer bytes.Buffer
	if err := json.NewEncoder(&buffer).Encode(&cfg); err != nil {
		return nil, err
	}

	return &gardener.Extension{
		Type: auditlogExtensionType,
		ProviderConfig: &runtime.RawExtension{
			Raw: buffer.Bytes(),
		},
	}, nil
}
