package skrdetails

import (
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/hyperscaler"
)

func AppliedACL(runtime imv1.Runtime) []string {
	if runtime.Spec.Shoot.Kubernetes.KubeAPIServer.ACL == nil {
		return nil
	}

	if runtime.Spec.Shoot.Provider.Type == hyperscaler.TypeAWS || runtime.Spec.Shoot.Provider.Type == hyperscaler.TypeAzure {
		return runtime.Spec.Shoot.Kubernetes.KubeAPIServer.ACL.AllowedCIDRs
	}
	return nil
}
