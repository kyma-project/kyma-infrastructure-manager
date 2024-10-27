package runtime

import (
	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/hack/shoot-comparator/pkg/shoot"
	"github.com/kyma-project/infrastructure-manager/pkg/config"
	gardener_shoot "github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot"
	"k8s.io/utils/ptr"
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

type ShootComparisonResult struct {
	OriginalShoot  v1beta1.Shoot
	ConvertedShoot v1beta1.Shoot
	Diff           *Difference
}

type Difference string

func (v Verifier) Do(runtimeToVerify v1.Runtime, shootToMatch v1beta1.Shoot) (ShootComparisonResult, error) {
	converter := gardener_shoot.NewConverter(v.converterConfig)
	shootFromConverter, err := converter.ToShoot(runtimeToVerify)
	if err != nil {
		return ShootComparisonResult{}, err
	}

	diff, err := compare(shootToMatch, shootFromConverter)
	if err != nil {
		return ShootComparisonResult{}, err
	}

	return ShootComparisonResult{
		OriginalShoot:  shootToMatch,
		ConvertedShoot: shootFromConverter,
		Diff:           diff,
	}, nil
}

func compare(originalShoot, convertedShoot v1beta1.Shoot) (*Difference, error) {
	matcher := shoot.NewMatcher(originalShoot)
	equal, err := matcher.Match(convertedShoot)
	if err != nil {
		return nil, err
	}

	if !equal {
		diff := Difference(matcher.FailureMessage(nil))
		return ptr.To(diff), nil
	}

	return nil, nil
}
