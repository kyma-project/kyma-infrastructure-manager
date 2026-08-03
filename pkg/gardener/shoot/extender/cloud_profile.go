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

func ExtendWithCloudProfile(gdchCloudProfileOverride string) func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
	return func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
		cloudProfileName, err := getCloudProfileName(runtime, gdchCloudProfileOverride)
		if err != nil {
			return err
		}

		shoot.Spec.CloudProfile = CreateCloudProfileReference(cloudProfileName)

		return nil
	}
}

func CreateCloudProfileReference(cloudProfileName string) *gardener.CloudProfileReference {
	return &gardener.CloudProfileReference{
		Kind: CloudProfileKind,
		Name: cloudProfileName,
	}
}

func getCloudProfileName(runtime imv1.Runtime, gdchCloudProfileOverride string) (string, error) {
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
		return DefaultOpenStackCloudProfileName, nil
	case hyperscaler.TypeAlicloud:
		return DefaultAlicloudCloudProfileName, nil
	}

	return "", errors.New("provider not supported")
}
