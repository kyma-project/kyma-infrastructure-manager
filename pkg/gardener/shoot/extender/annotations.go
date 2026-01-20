package extender

import (
	"fmt"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
)

// Provisioner was setting the following annotations:
//- kcp.provisioner.kyma-project.io/licence-type
//- kcp.provisioner.kyma-project.io/runtime-id
//- support.gardener.cloud/eu-access-for-cluster-nodes

const (
	ShootRuntimeGenerationAnnotation = "infrastructuremanager.kyma-project.io/runtime-generation"
	ShootRuntimeIDAnnotation         = "infrastructuremanager.kyma-project.io/runtime-id"
	ShootLicenceTypeAnnotation       = "infrastructuremanager.kyma-project.io/licence-type"
	RuntimeIDLabel                   = "kyma-project.io/runtime-id"
	ShootCleanupExtendedApisGracePeriodAnnotation = "shoot.gardener.cloud/cleanup-extended-apis-finalize-grace-period-seconds"
	DefaultShootCleanupExtendedApisGracePeriodValue = "120"
	ShootCleanupKubernetesResourcesGracePeriodAnnotation = "shoot.gardener.cloud/cleanup-kubernetes-resources-finalize-grace-period-seconds"
	DefaultCleanupKubernetesResourcesGracePeriodValue= "120"
	ShootCleanupWebhooksGracePeriodAnnotation = "shoot.gardener.cloud/cleanup-webhooks-finalize-grace-period-seconds"
	DefaultCleanupWebhooksGracePeriodValue= "60"
)

func ExtendWithAnnotations(runtime imv1.Runtime, shoot *gardener.Shoot) error {
	shoot.Annotations = getAnnotations(runtime)

	return nil
}

func getAnnotations(runtime imv1.Runtime) map[string]string {
	annotations := map[string]string{
		ShootRuntimeIDAnnotation:         runtime.Labels[RuntimeIDLabel],
		ShootRuntimeGenerationAnnotation: fmt.Sprintf("%v", runtime.Generation),
		ShootCleanupExtendedApisGracePeriodAnnotation: DefaultShootCleanupExtendedApisGracePeriodValue,
		ShootCleanupKubernetesResourcesGracePeriodAnnotation: DefaultCleanupKubernetesResourcesGracePeriodValue,
		ShootCleanupWebhooksGracePeriodAnnotation: DefaultCleanupWebhooksGracePeriodValue,
	}

	if runtime.Spec.Shoot.LicenceType != nil && *runtime.Spec.Shoot.LicenceType != "" {
		annotations[ShootLicenceTypeAnnotation] = *runtime.Spec.Shoot.LicenceType
	}

	return annotations
}
