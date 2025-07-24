package fsm

import (
	"context"
	"fmt"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/log_level"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	kubeconfigSecretKey = "config"
)

func sFnHandleKubeconfig(ctx context.Context, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	runtimeID := s.instance.Labels[imv1.LabelKymaRuntimeID]

	// get section
	var cluster imv1.GardenerCluster
	err := m.KcpClient.Get(ctx, types.NamespacedName{
		Namespace: s.instance.Namespace,
		Name:      runtimeID,
	}, &cluster)

	if err != nil {
		if !k8serrors.IsNotFound(err) {
			m.log.Error(err, "GardenerCluster CR read error", "name", runtimeID)
			s.instance.UpdateStatePending(
				imv1.ConditionTypeRuntimeKubeconfigReady,
				imv1.ConditionReasonKubernetesAPIErr,
				"False",
				err.Error(),
			)
			m.Metrics.IncRuntimeFSMStopCounter()
			return updateStatusAndStop()
		}

		m.log.V(log_level.DEBUG).Info("GardenerCluster CR not found, creating a new one", "name", runtimeID)
		err = m.KcpClient.Create(ctx, makeGardenerClusterForRuntime(s.instance, s.shoot))
		if err != nil {
			m.log.Error(err, "GardenerCluster CR create error", "name", runtimeID)
			s.instance.UpdateStatePending(
				imv1.ConditionTypeRuntimeKubeconfigReady,
				imv1.ConditionReasonKubernetesAPIErr,
				"False",
				err.Error(),
			)
			m.Metrics.IncRuntimeFSMStopCounter()
			return updateStatusAndStop()
		}

		s.instance.UpdateStatePending(imv1.ConditionTypeRuntimeKubeconfigReady, imv1.ConditionReasonGardenerCRCreated, "Unknown", "Gardener Cluster CR created, waiting for readiness")
		return updateStatusAndRequeueAfter(m.ControlPlaneRequeueDuration)
	}

	// wait section
	if cluster.Status.State != imv1.ReadyState {
		return requeueAfter(m.ControlPlaneRequeueDuration)
	}

	m.log.V(log_level.DEBUG).Info("GardenerCluster CR is ready", "name", runtimeID)

	return ensureStatusConditionIsSetAndContinue(&s.instance,
		imv1.ConditionTypeRuntimeKubeconfigReady,
		imv1.ConditionReasonGardenerCRReady,
		"Gardener Cluster CR is ready.",
		sFnGardenClusterPostProcessing)
}

func makeGardenerClusterForRuntime(runtime imv1.Runtime, shoot *gardener.Shoot) *imv1.GardenerCluster {
	gardenCluster := &imv1.GardenerCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "GardenerCluster",
			APIVersion: imv1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      runtime.Labels[imv1.LabelKymaRuntimeID],
			Namespace: runtime.Namespace,
			Annotations: map[string]string{
				"skr-domain": *shoot.Spec.DNS.Domain,
			},
			Labels: map[string]string{
				imv1.LabelKymaInstanceID:      runtime.Labels[imv1.LabelKymaInstanceID],
				imv1.LabelKymaRuntimeID:       runtime.Labels[imv1.LabelKymaRuntimeID],
				imv1.LabelKymaBrokerPlanID:    runtime.Labels[imv1.LabelKymaBrokerPlanID],
				imv1.LabelKymaBrokerPlanName:  runtime.Labels[imv1.LabelKymaBrokerPlanName],
				imv1.LabelKymaGlobalAccountID: runtime.Labels[imv1.LabelKymaGlobalAccountID],
				imv1.LabelKymaSubaccountID:    runtime.Labels[imv1.LabelKymaSubaccountID], // BTW most likely this value will be missing
				imv1.LabelKymaName:            runtime.Labels[imv1.LabelKymaName],

				// values from Runtime CR fields
				imv1.LabelKymaPlatformRegion: runtime.Spec.Shoot.PlatformRegion,
				imv1.LabelKymaRegion:         runtime.Spec.Shoot.Region,
				imv1.LabelKymaShootName:      shoot.Name,

				// hardcoded values
				imv1.LabelKymaManagedBy: "infrastructure-manager",
				imv1.LabelKymaInternal:  "true",
			},
		},
		Spec: imv1.GardenerClusterSpec{
			Shoot: imv1.Shoot{
				Name: shoot.Name,
			},
			Kubeconfig: imv1.Kubeconfig{
				Secret: imv1.Secret{
					Name:      fmt.Sprintf("kubeconfig-%s", runtime.Labels[imv1.LabelKymaRuntimeID]),
					Namespace: runtime.Namespace,
					Key:       kubeconfigSecretKey,
				},
			},
		},
	}

	return gardenCluster
}
