package fsm

import (
	"context"
	ctrl "sigs.k8s.io/controller-runtime"
)

func sFnGardenClusterPostProcessing(ctx context.Context, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) { //nolint:unused
	//runtimeClient, err := m.RuntimeClientGetter.Get(ctx, s.instance)
	//	//if err != nil {
	//	//	s.instance.UpdateStatePending(
	//	//		imv1.ConditionTypeRuntimeKubeconfigReady,
	//	//		imv1.ConditionReasonSeedClusterPostProcessingError,
	//	//		"False",
	//	//		err.Error(),
	//	//	)
	//	//	m.log.Error(err, "Failed to get runtime client")
	//	//
	//	//	return updateStatusAndRequeue()
	//	//}
	//	//
	//	//// TODO: pass Garden namespace name
	//	//secretSyncer := registrycache.NewSecretSyncer(m.GardenClient, runtimeClient, "", s.instance.Name)
	//	//registryCachesWitSecrets := getRegistryCachesWithSecrets(s.instance)
	//	//
	//	//if len(registryCachesWitSecrets) > 0 {
	//	//	err = secretSyncer.Delete(ctx, registryCachesWitSecrets)
	//	//	if err != nil {
	//	//		s.instance.UpdateStatePending(
	//	//			imv1.ConditionTypeRuntimeKubeconfigReady,
	//	//			imv1.ConditionReasonSeedClusterPostProcessingError,
	//	//			"False",
	//	//			err.Error(),
	//	//		)
	//	//		m.log.Error(err, "Failed to remove not used registry cache secrets")
	//	//
	//	//		return updateStatusAndRequeue()
	//	//	}
	//	//}
	//	//
	return switchState(sFnConfigureSKR)
}
