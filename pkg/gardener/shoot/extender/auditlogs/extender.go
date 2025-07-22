package auditlogs

import (
	"strings"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
)

const experimentalAuditPolicy = "experimental-audit-policy"

type Extend = func(runtime imv1.Runtime, shoot *gardener.Shoot) error

type operation = func(*gardener.Shoot) error

func NewAuditlogExtenderForCreate(policyConfigMapName string, data AuditLogData) Extend {
	return func(rt imv1.Runtime, shoot *gardener.Shoot) error {
		experimentalAuditPolicyVal, found := rt.Annotations[experimentalAuditPolicy]
		if found && strings.ToLower(experimentalAuditPolicyVal) == "true" {
			policyConfigMapName = experimentalAuditPolicy
		}

		for _, f := range []operation{
			oSetSecret(data.SecretName),
			oSetPolicyConfigmap(policyConfigMapName),
		} {
			if err := f(shoot); err != nil {
				return err
			}
		}
		return nil
	}
}

func NewAuditlogExtenderForPatch(policyConfigMapName string) Extend {
	return func(_ imv1.Runtime, shoot *gardener.Shoot) error {
		return oSetPolicyConfigmap(policyConfigMapName)(shoot)
	}
}
