package auditlog

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/go-playground/validator/v10"
)

var (
	ErrConfigurationNotFound = fmt.Errorf("audit logs configuration not found")
)

type region = string

type providerType = string

// AuditLogData represents audit log configuration
type AuditLogData struct {
	TenantID            string `json:"tenantID" validate:"required"`
	ServiceURL          string `json:"serviceURL" validate:"required,url"`
	SecretName          string `json:"secretName" validate:"required"`
	ReadCredsSecretName string `json:"readCredsSecretName,omitempty"` // Only used for dedicated audit logging
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

// LoadConfiguration loads audit log configuration from a JSON file
func LoadConfiguration(path string) (Configuration, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var data Configuration
	if err := json.NewDecoder(file).Decode(&data); err != nil {
		return nil, err
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	for _, nestedMap := range data {
		for _, auditLogData := range nestedMap {
			if err := validate.Struct(auditLogData); err != nil {
				return nil, err
			}
		}
	}

	return data, nil
}
