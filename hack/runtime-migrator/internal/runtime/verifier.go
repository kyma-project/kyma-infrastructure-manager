package runtime

import (
	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/hack/runtime-migrator-app/internal/comparator"
	"github.com/kyma-project/infrastructure-manager/pkg/config"
	gardener_shoot "github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot"
)

type Verifier struct {
	converterConfig config.ConverterConfig
	outputPath      string
}

func NewVerifier(converterConfig config.ConverterConfig, outputPath string) Verifier {
	return Verifier{
		converterConfig: converterConfig,
	}
}

func (v Verifier) Do(runtimeToVerify v1.Runtime, shootToMatch v1beta1.Shoot) (bool, error) {
	converter := gardener_shoot.NewConverter(v.converterConfig)
	shootFromConverter, err := converter.ToShoot(runtimeToVerify)
	if err != nil {
		return false, err
	}

	result, err := comparator.CompareShoots(shootToMatch, shootFromConverter)
	if err != nil {
		return false, err
	}

	if result.Equal {
		_, err := comparator.SaveComparisonReport(result, v.outputPath, shootToMatch.Name)
		if err != nil {
			return false, err
		}
	}

	return result.Equal, nil
}
