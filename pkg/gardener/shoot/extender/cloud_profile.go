package extender

import (
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/hyperscaler"
	"github.com/pkg/errors"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	DefaultAWSCloudProfileName       = "aws"
	DefaultAzureCloudProfileName     = "az"
	DefaultGCPCloudProfileName       = "gcp"
	DefaultGDCHCloudProfileName      = "cat"
	DefaultOpenStackCloudProfileName = "converged-cloud-kyma"
	DefaultAlicloudCloudProfileName  = "alicloud"
	CloudProfileKind                 = "CloudProfile"
)

// ExtendWithCloudProfile sets the shoot cloud profile. The cloud profile is immutable in Gardener,
// so on patch (existingCloudProfileName non-empty) the existing value is preserved and never
// re-derived from config; a changed config value cannot rewrite it on an existing shoot. If the
// configured value differs from the existing one, a warning is logged that the change is ignored.
// On create (existingCloudProfileName empty) the value is derived from config.
func ExtendWithCloudProfile(existingCloudProfileName, gdchCloudProfileOverride, openstackCloudProfileOverride string) func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
	return func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
		configuredName, err := getCloudProfileName(runtime, gdchCloudProfileOverride, openstackCloudProfileOverride)

		if err != nil {
			return err
		}

		// On create (existingCloudProfileName empty) the config-derived name is used as-is.
		// On patch the existing value is always kept, because the cloud profile is immutable; the
		// configured name is then only used to detect and warn about an ignored change.
		if existingCloudProfileName != "" && configuredName != existingCloudProfileName {
			logf.Log.Info("Cloud profile is immutable and cannot be changed during patch; keeping the existing value",
				"existingCloudProfileName", existingCloudProfileName,
				"requestedCloudProfileName", configuredName,
				"shoot", runtime.Spec.Shoot.Name)
			configuredName = existingCloudProfileName
		}

		shoot.Spec.CloudProfile = CreateCloudProfileReference(configuredName)
		return nil
	}
}

func CreateCloudProfileReference(cloudProfileName string) *gardener.CloudProfileReference {
	return &gardener.CloudProfileReference{
		Kind: CloudProfileKind,
		Name: cloudProfileName,
	}
}

func getCloudProfileName(runtime imv1.Runtime, gdchCloudProfileOverride, openstackCloudProfileOverride string) (string, error) {
	switch runtime.Spec.Shoot.Provider.Type {
	case hyperscaler.TypeAWS:
		return DefaultAWSCloudProfileName, nil
	case hyperscaler.TypeGCP:
		return DefaultGCPCloudProfileName, nil
	case hyperscaler.TypeAzure:
		return DefaultAzureCloudProfileName, nil
	case hyperscaler.TypeGDCH:
		if gdchCloudProfileOverride != "" {
			return gdchCloudProfileOverride, nil
		}
		return DefaultGDCHCloudProfileName, nil
	case hyperscaler.TypeOpenStack:
		if openstackCloudProfileOverride != "" {
			return openstackCloudProfileOverride, nil
		}
		return DefaultOpenStackCloudProfileName, nil
	case hyperscaler.TypeAlicloud:
		return DefaultAlicloudCloudProfileName, nil
	}

	return "", errors.New("provider not supported")
}
