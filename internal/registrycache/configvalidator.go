package registrycache

import (
	"github.com/kyma-project/kim-snatch/api/v1beta1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

type ConfigValidator interface {
	Do(config []v1beta1.RegistryCache) Result
}

type Result struct {
	config v1beta1.RegistryCacheConfig
	Valid  bool
	Errors field.ErrorList
}

type configValidator struct {
}

func NewConfigValidator(config v1beta1.RegistryCacheConfig) configValidator {
	return configValidator{}
}

func (v configValidator) Do(config v1beta1.RegistryCacheConfig) Result {
	return Result{}
}
