package auditlogs

import (
	"fmt"
	"strings"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
)

const experimentalAuditPolicy = "experimental-audit-policy"

var experimentalAuditPolicyAnnotationName = fmt.Sprintf("operator.kyma-project.io/%s", experimentalAuditPolicy)

type Extend = func(runtime imv1.Runtime, shoot *gardener.Shoot) error

type operation = func(*gardener.Shoot) error

func fixPolicyConfigMapName(annotations map[string]string, defaultPolicyMapName string) string {
	annotationVal, found := annotations[experimentalAuditPolicyAnnotationName]
	if found && strings.ToLower(annotationVal) == "true" {
		return experimentalAuditPolicy
	}
	return defaultPolicyMapName
}

func NewAuditlogExtenderForCreate(policyConfigMapName string, data AuditLogData) Extend {
	return func(rt imv1.Runtime, shoot *gardener.Shoot) error {
		policyConfigMapName := fixPolicyConfigMapName(rt.Annotations, policyConfigMapName)
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
	return func(rt imv1.Runtime, shoot *gardener.Shoot) error {
		policyConfigMapName := fixPolicyConfigMapName(rt.Annotations, policyConfigMapName)
		return oSetPolicyConfigmap(policyConfigMapName)(shoot)
	}
}
