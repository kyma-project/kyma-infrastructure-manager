package registrycache

import (
	"github.com/kyma-project/kim-snatch/api/v1beta1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

type ConfigVerifier interface {
	Verify(config []v1beta1.RegistryCache) field.ErrorList
}

type configVerifier struct {
}

func (v configVerifier) Verify(config []v1beta1.RegistryCache) field.ErrorList {
	return nil
}
