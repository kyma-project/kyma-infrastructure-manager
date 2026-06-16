package auditlog

import "fmt"

var (
	ErrConfigurationNotFound = fmt.Errorf("audit logs configuration not found")
)

type region = string

type providerType = string

// AuditLogData represents audit log configuration
type AuditLogData struct {
	TenantID    string `json:"tenantID" validate:"required"`
	ServiceURL  string `json:"serviceURL" validate:"required,url"`
	SecretName  string `json:"secretName" validate:"required"`
	IsDedicated bool   `json:"-"` // Indicates if this is from dedicated or shared config
}

// Configuration is the map-based shared audit log configuration
// Key structure: providerType -> region -> AuditLogData
type Configuration map[providerType]map[region]AuditLogData

// GetAuditLogData retrieves audit log configuration for a specific provider and region
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
