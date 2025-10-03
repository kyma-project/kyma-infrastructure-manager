package token

import (
	"time"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

const upperBound = 7776000 // 90 days in seconds
const lowerBound = 2592000 // 30 days in seconds
type TimeBoundaries struct {
	TooShort   bool
	TooLong    bool
	NotDefined bool
}

func NewExpirationTimeExtender(maxTokenExpiration string) func(_ imv1.Runtime, shoot *gardener.Shoot) error {
	return func(_ imv1.Runtime, shoot *gardener.Shoot) error {
		if shoot.Spec.Kubernetes.KubeAPIServer == nil {
			shoot.Spec.Kubernetes.KubeAPIServer = &gardener.KubeAPIServerConfig{}
		}
		if shoot.Spec.Kubernetes.KubeAPIServer.ServiceAccountConfig == nil {
			shoot.Spec.Kubernetes.KubeAPIServer.ServiceAccountConfig = &gardener.ServiceAccountConfig{}
		}
		shoot.Spec.Kubernetes.KubeAPIServer.ServiceAccountConfig.ExtendTokenExpiration = ptr.To(false)

		result, err := ValidateTokenExpirationTime(maxTokenExpiration)
		if err != nil {
			return err
		}

		shoot.Spec.Kubernetes.KubeAPIServer.ServiceAccountConfig.MaxTokenExpiration = &metav1.Duration{
			Duration: determineTokenExpiration(result, maxTokenExpiration),
		}

		return nil
	}
}

func ValidateTokenExpirationTime(maxTokenExpiration string) (TimeBoundaries, error) {
	if maxTokenExpiration == "" {
		return TimeBoundaries{NotDefined: true}, nil
	}

	parsedTokenExpirationTime, err := time.ParseDuration(maxTokenExpiration)
	if err != nil {
		return TimeBoundaries{}, err
	}

	seconds := parsedTokenExpirationTime.Seconds()
	return TimeBoundaries{
		TooShort:   seconds < lowerBound,
		TooLong:    seconds > upperBound,
		NotDefined: false,
	}, nil
}

func determineTokenExpiration(result TimeBoundaries, maxTokenExpiration string) time.Duration {
	switch {
	case result.TooShort || result.NotDefined:
		return lowerBound
	case result.TooLong:
		return upperBound
	default:
		parsedDuration, _ := time.ParseDuration(maxTokenExpiration)
		return parsedDuration
	}
}
