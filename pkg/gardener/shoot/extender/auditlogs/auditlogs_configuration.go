package auditlogs

import "fmt"

var (
	ErrConfigurationNotFound = fmt.Errorf("audit logs configuration not found")
	zero                     = AuditLogData{}
)

type AuditlogsConfiguration map[string]map[string]AuditLogData

func (a *AuditlogsConfiguration) GetAuditLogData(providerType, region string) (AuditLogData, error) {
	providerCfg, found := (*a)[providerType]
	if !found {
		return zero, fmt.Errorf("%w: missing providerType: '%s'",
			ErrConfigurationNotFound,
			providerType)
	}

	providerCfgForRegion, found := providerCfg[region]
	if !found {
		return zero, fmt.Errorf("%w: missing region: '%s' for providerType: '%s'",
			ErrConfigurationNotFound,
			region,
			providerType)
	}

	return providerCfgForRegion, nil
}
