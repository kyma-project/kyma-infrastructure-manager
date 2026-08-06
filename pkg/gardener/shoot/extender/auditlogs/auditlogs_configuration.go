package auditlogs

import "fmt"

var (
	ErrConfigurationNotFound = fmt.Errorf("audit logs configuration not found")
)

const (
	SharedAuditlogSecretReference    = "auditlog-credentials"
	DedicatedAuditlogSecretReference = "dedicated-auditlog-credentials"
)

type region = string

type providerType = string

type AuditLogData struct {
	TenantID   string `json:"tenantID" validate:"required"`
	ServiceURL string `json:"serviceURL" validate:"required,url"`
	SecretName string `json:"secretName" validate:"required"`
	// Dedicated is set internally to select the correct secret reference name; it is never
	// (un)marshalled from/to the shared audit log configuration file.
	Dedicated bool `json:"-"`
}

// SecretReferenceName returns the name under which the audit log secret should be
// referenced in shoot.spec.resources and in the extension's providerConfig,
// depending on whether this is shared or dedicated audit log configuration.
func (d AuditLogData) SecretReferenceName() string {
	if d.Dedicated {
		return DedicatedAuditlogSecretReference
	}
	return SharedAuditlogSecretReference
}

type Configuration map[providerType]map[region]AuditLogData

func (a Configuration) GetAuditLogData(providerType, region string) (AuditLogData, error) {
	providerCfg, found := (a)[providerType]
	if !found {
		return AuditLogData{}, fmt.Errorf("%w: missing providerType: '%s'",
			ErrConfigurationNotFound,
			providerType)
	}

	providerCfgForRegion, found := providerCfg[region]
	if !found {
		return AuditLogData{}, fmt.Errorf("%w: missing region: '%s' for providerType: '%s'",
			ErrConfigurationNotFound,
			region,
			providerType)
	}

	return providerCfgForRegion, nil
}
