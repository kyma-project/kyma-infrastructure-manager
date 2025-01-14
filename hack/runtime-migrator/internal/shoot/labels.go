package shoot

import (
	"context"
	runtimev1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func SetControlledByKIM(ctx context.Context, kcpClient client.Client, runtimeID string, fieldManagerName string) error {
	return patchControlledByProvisionerLabel(ctx, kcpClient, runtimeID, fieldManagerName, "false")
}

func SetControlledByProvisioner(ctx context.Context, kcpClient client.Client, runtimeID string, fieldManagerName string) error {
	return patchControlledByProvisionerLabel(ctx, kcpClient, runtimeID, fieldManagerName, "true")
}

func patchControlledByProvisionerLabel(ctx context.Context, kcpClient client.Client, runtimeID string, fieldManagerName string, labelValue string) error {
	getCtx, cancelGet := context.WithTimeout(ctx, timeoutK8sOperation)
	defer cancelGet()

	key := types.NamespacedName{
		Name:      runtimeID,
		Namespace: "kcp-system",
	}
	var runtime runtimev1.Runtime

	err := kcpClient.Get(getCtx, key, &runtime, &client.GetOptions{})
	if err != nil {
		return err
	}

	runtime.Labels["kyma-project.io/controlled-by-provisioner"] = labelValue

	patchCtx, cancelPatch := context.WithTimeout(ctx, timeoutK8sOperation)
	defer cancelPatch()

	runtime.Kind = "Runtime"
	runtime.APIVersion = "infrastructuremanager.kyma-project.io/v1"
	runtime.ManagedFields = nil

	return kcpClient.Patch(patchCtx, &runtime, client.Apply, &client.PatchOptions{
		FieldManager: fieldManagerName,
		Force:        ptr.To(true),
	})
}
