package skrdetails

import (
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
)

func AppliedACL(runtime imv1.Runtime) []string {
	if runtime.Spec.Shoot.Kubernetes.KubeAPIServer.ACL == nil {
		return nil
	}
	return runtime.Spec.Shoot.Kubernetes.KubeAPIServer.ACL.AllowedCIDRs
}
