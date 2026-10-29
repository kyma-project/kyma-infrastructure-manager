package extender

import (
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/hyperscaler"
	"github.com/pkg/errors"
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
// so on patch (existingCloudProfile non-nil) the existing reference is preserved exactly — including
// its Kind, which may be CloudProfile or NamespacedCloudProfile — and a changed config value cannot
// rewrite it on an existing shoot. On create (existingCloudProfile nil) the value is derived from config.
func ExtendWithCloudProfile(existingCloudProfile *gardener.CloudProfileReference, gdchCloudProfileOverride, openstackCloudProfileOverride string) func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
	return func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
		if existingCloudProfile != nil {
			shoot.Spec.CloudProfile = existingCloudProfile
			return nil
		}

		configuredName, err := getCloudProfileName(runtime, gdchCloudProfileOverride, openstackCloudProfileOverride)
		if err != nil {
			return err
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
