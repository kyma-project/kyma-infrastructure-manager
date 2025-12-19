package extender

import (
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"k8s.io/utils/ptr"
)

// ExtendWithCredentialsBinding extends the Shoot with CredentialsBindingName or SecretBindingName
// This depends on the flag credentialBindingEnabled.
// Extender can be removed and setting CredentialBindingName can be moved to pkg/gardener/shoot/converter#ToShoot
// after CredentialBinding migration is finished.
func ExtendWithCredentialsBinding(credentialBindingEnabled bool) func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
	return func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
		if credentialBindingEnabled {
			shoot.Spec.CredentialsBindingName = ptr.To(runtime.Spec.Shoot.SecretBindingName)
			shoot.Spec.SecretBindingName = nil //nolint:staticcheck
		} else {
			shoot.Spec.SecretBindingName = ptr.To(runtime.Spec.Shoot.SecretBindingName) //nolint:staticcheck
		}

		return nil
	}
}
