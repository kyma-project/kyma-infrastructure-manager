package fsm

import (
	"context"
	"fmt"
	registrycacheapi "github.com/kyma-project/registry-cache/api/v1beta1"
	"reflect"

	"github.com/kyma-project/infrastructure-manager/pkg/auditlog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/log_level"
	"github.com/kyma-project/infrastructure-manager/internal/registrycache"
	gardener_shoot "github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/auditlogs"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/token"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/structuredauth"
	"github.com/kyma-project/infrastructure-manager/pkg/reconciler"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const fieldManagerName = "kim"

func sFnPatchExistingShoot(ctx context.Context, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) {

	auditLogConfig, nextState, res, err := resolveAuditLogData(ctx, m, s)
	if nextState != nil {
		return nextState, res, err
	}

	oidcConfig := structuredauth.GetOIDCConfigOrDefault(s.instance, m.ConverterConfig.Kubernetes.DefaultOperatorOidc.ToOIDCConfig())

	cmName := fmt.Sprintf(extender.StructuredAuthConfigFmt, s.instance.Spec.Shoot.Name)
	err = structuredauth.CreateOrUpdateStructuredAuthConfigMap(
		ctx,
		m.GardenClient,
		types.NamespacedName{Name: cmName, Namespace: m.ShootNamesapace},
		oidcConfig,
	)

	if err != nil {
		m.Metrics.IncRuntimeFSMStopCounter()
		return updateStateFailedWithErrorAndStop(
			&s.instance,
			imv1.ConditionTypeRuntimeProvisioned,
			imv1.ConditionReasonOidcError,
			msgFailedStructuredConfigMap)
	}

	timeBoundaries, _ := token.ValidateTokenExpirationTime(m.ConverterConfig.Kubernetes.KubeApiServer.MaxTokenExpiration)
	logTokenExpirationInfo(m.log, timeBoundaries)

	patchOptions, err := getPatchOptions(ctx, m, s, auditLogConfig)
	if err != nil {
		m.log.Error(err, "Failed to check if registry cache secret should be removed")

		s.instance.UpdateStatePending(
			imv1.ConditionTypeRuntimeProvisioned,
			imv1.ConditionReasonProcessing,
			metav1.ConditionFalse,
			"Failed to get patch options",
		)

		return updateStatusAndRequeueAfter(m.StatusRequeueDelay)
	}

	// NOTE: In the future we want to pass the whole shoot object here
	updatedShoot, err := convertPatch(ctx, &s.instance, patchOptions)

	if err != nil {
		m.log.Error(err, "Failed to convert Runtime instance to shoot object, exiting with no retry")
		m.Metrics.IncRuntimeFSMStopCounter()
		if m.RegistryCacheConfigControllerEnabled {
			setRegistryCacheStatusFailed(ctx, m, s)
		}

		return updateStateFailedWithErrorAndStop(&s.instance, imv1.ConditionTypeRuntimeProvisioned, imv1.ConditionReasonConversionError, fmt.Sprintf("Runtime conversion error %v", err))
	}

	m.log.V(log_level.DEBUG).Info("Shoot converted successfully", "Name", updatedShoot.Name, "Namespace", updatedShoot.Namespace)

	hasRegistryCacheCountChanged, err := registrycache.HasRegistryCacheCountChanged(s.shoot.Spec.Extensions, s.instance.Spec.Caching)
	if err != nil {
		m.log.Error(err, "Failed to check if registry cache secret should be removed")

		s.instance.UpdateStatePending(imv1.ConditionTypeRuntimeProvisioned, imv1.ConditionReasonRegistryCacheConfigured, metav1.ConditionFalse, "Failed to check if registry cache secret should be removed")
		return updateStatusAndRequeueAfter(m.StatusRequeueDelay)
	}

	workersShouldBeUpdated := !workersAreEqual(s.shoot.Spec.Provider.Workers, updatedShoot.Spec.Provider.Workers)

	// The additional Update function is required to fully replace collections with the ones defined in updated runtime object.
	// This is a workaround for the sigs.k8s.io/controller-runtime/pkg/client, which does not support replacing collections with client.Patch.
	// The client is able to add an item to the collection, but not to remove it.
	// More info: https://github.com/kyma-project/infrastructure-manager/issues/640

	if workersShouldBeUpdated || hasRegistryCacheCountChanged {
		copyShoot := s.shoot.DeepCopy()

		if workersShouldBeUpdated {
			copyShoot.Spec.Provider.Workers = updatedShoot.Spec.Provider.Workers
			copyShoot.Spec.Provider.ControlPlaneConfig = updatedShoot.Spec.Provider.ControlPlaneConfig
			copyShoot.Spec.Provider.InfrastructureConfig = updatedShoot.Spec.Provider.InfrastructureConfig
		}

		if hasRegistryCacheCountChanged {
			copyShoot.Spec.Extensions = updatedShoot.Spec.Extensions
			copyShoot.Spec.Resources = updatedShoot.Spec.Resources
		}

		updateErr := m.GardenClient.Update(ctx, copyShoot,
			&client.UpdateOptions{
				FieldManager: fieldManagerName,
			})

		nextState, res, err := handleUpdateError(updateErr, m, s, "Failed to update shoot object, exiting with no retry", "Gardener API shoot update error")
		if nextState != nil {
			return nextState, res, err
		}
	}

	bindingShouldBePatched := m.ConverterConfig.Gardener.EnableCredentialBinding && s.shoot.Spec.SecretBindingName != nil && *s.shoot.Spec.SecretBindingName != "" //nolint:staticcheck
	// Gardener is not handling properly the change from SecretBindingName to CredentialsBindingName with the Patch operation.
	// Therefore, we need to do an additional Update operation to set the CredentialsBindingName and remove the SecretBindingName.
	// This update can be removed after migration to CredentialsBinding is completed and all runtimes are using it.
	if bindingShouldBePatched {
		copyShoot := s.shoot.DeepCopy()
		copyShoot.Spec.CredentialsBindingName = ptr.To(s.instance.Spec.Shoot.SecretBindingName)
		copyShoot.Spec.SecretBindingName = nil //nolint:staticcheck

		updateErr := m.GardenClient.Update(ctx, copyShoot,
			&client.UpdateOptions{
				FieldManager: fieldManagerName,
			})

		nextState, res, err := handleUpdateError(updateErr, m, s, "Failed to update shoot object with new CredentialsBinding, exiting with no retry", "Gardener API shoot update error")
		if nextState != nil {
			return nextState, res, err
		}
	}

	//nolint:staticcheck // SA1019: client.Apply is used with Patch, which is the correct API for this version
	patchErr := m.GardenClient.Patch(ctx, &updatedShoot, client.Apply, &client.PatchOptions{
		FieldManager: fieldManagerName,
		Force:        ptr.To(true),
	})
	nextState, res, patchErr = handleUpdateError(patchErr, m, s, "Failed to patch shoot object, exiting with no retry", "Gardener API shoot patch error")

	if nextState != nil {
		return nextState, res, patchErr
	}

	err = handleForceReconciliationAnnotation(&s.instance, m, ctx)
	if err != nil {
		m.log.Error(err, "could not handle force reconciliation annotation. Scheduling for retry.")
		return requeue()
	}

	if updatedShoot.Generation == s.shoot.Generation {
		m.log.V(log_level.DEBUG).Info("Gardener shoot for runtime did not change after patch, moving to processing", "Name", s.shoot.Name, "Namespace", s.shoot.Namespace)

		s.instance.UpdateStatePending(
			imv1.ConditionTypeRuntimeProvisioned,
			imv1.ConditionReasonProcessing,
			metav1.ConditionTrue,
			"Shoot patched without changes",
		)

		return switchState(sFnHandleKubeconfig)
	}

	m.log.V(log_level.DEBUG).Info("Gardener shoot for runtime patched successfully", "Name", s.shoot.Name, "Namespace", s.shoot.Namespace)

	s.instance.UpdateStatePending(
		imv1.ConditionTypeRuntimeProvisioned,
		imv1.ConditionReasonProcessing,
		metav1.ConditionUnknown,
		"Shoot is pending for update after patch",
	)

	return updateStatusAndRequeueAfter(m.GardenerRequeueDuration)
}

func handleUpdateError(err error, m *fsm, s *systemState, errMsg, statusMsg string) (stateFn, *ctrl.Result, error) {
	if err != nil {
		if k8serrors.IsConflict(err) {
			m.log.Info("Gardener shoot for runtime is outdated, retrying", "Name", s.shoot.Name, "Namespace", s.shoot.Namespace)

			s.instance.UpdateStatePending(
				imv1.ConditionTypeRuntimeProvisioned,
				imv1.ConditionReasonProcessing,
				metav1.ConditionUnknown,
				"Shoot is pending for update after conflict error",
			)

			return updateStatusAndRequeueAfter(m.GardenerRequeueDuration)
		}

		// We're retrying on Forbidden error because Gardener returns them from time too time for operations that are properly authorized.
		if k8serrors.IsForbidden(err) {
			m.log.Error(err, "Gardener shoot for runtime is forbidden, retrying")

			s.instance.UpdateStatePending(
				imv1.ConditionTypeRuntimeProvisioned,
				imv1.ConditionReasonProcessing,
				metav1.ConditionUnknown,
				"Shoot is pending for update after forbidden error",
			)

			return updateStatusAndRequeueAfter(m.GardenerRequeueDuration)
		}

		m.log.Error(err, errMsg)
		m.Metrics.IncRuntimeFSMStopCounter()
		return updateStateFailedWithErrorAndStop(&s.instance, imv1.ConditionTypeRuntimeProvisioned, imv1.ConditionReasonProcessingErr, fmt.Sprintf("%s: %v", statusMsg, err))
	}

	return nil, nil, nil
}

func workersAreEqual(workers []gardener.Worker, workers2 []gardener.Worker) bool {
	if len(workers) != len(workers2) {
		return false
	}

	for i := range workers {
		if !reflect.DeepEqual(workers[i], workers2[i]) {
			return false
		}
	}
	return true
}

func handleForceReconciliationAnnotation(runtime *imv1.Runtime, fsm *fsm, ctx context.Context) error {
	annotations := runtime.Annotations
	if reconciler.ShouldForceReconciliation(annotations) {
		fsm.log.Info("Force reconciliation annotation found, removing the annotation and continuing the reconciliation")
		delete(annotations, reconciler.ForceReconcileAnnotation)
		runtime.SetAnnotations(annotations)

		err := fsm.KcpClient.Update(ctx, runtime)
		if err != nil {
			return err
		}

	}
	return nil
}

func convertPatch(ctx context.Context, instance *imv1.Runtime, opts gardener_shoot.PatchOpts) (gardener.Shoot, error) {
	if err := instance.ValidateRequiredLabels(); err != nil {
		return gardener.Shoot{}, err
	}

	converter := gardener_shoot.NewConverterPatch(ctx, opts)
	newShoot, err := converter.ToShoot(*instance)
	if err != nil {
		return newShoot, err
	}

	return newShoot, nil
}

func updateStateFailedWithErrorAndStop(instance *imv1.Runtime,
	//nolint:unparam
	c imv1.RuntimeConditionType, r imv1.RuntimeConditionReason, msg string) (stateFn, *ctrl.Result, error) {
	instance.UpdateStateFailed(c, r, msg)
	return updateStatusAndStop()
}

func getPatchOptions(ctx context.Context, m *fsm, s *systemState, auditLogConfig auditlogs.AuditLogData) (gardener_shoot.PatchOpts, error) {
	patchOptions := gardener_shoot.PatchOpts{
		KcpClient:                       m.KcpClient,
		ConverterConfig:                 m.ConverterConfig,
		AuditLogData:                    auditLogConfig,
		MaintenanceTimeWindow:           getMaintenanceTimeWindow(s, m),
		Workers:                         s.shoot.Spec.Provider.Workers,
		ShootK8SVersion:                 s.shoot.Spec.Kubernetes.Version,
		Extensions:                      s.shoot.Spec.Extensions,
		Resources:                       s.shoot.Spec.Resources,
		InfrastructureConfig:            s.shoot.Spec.Provider.InfrastructureConfig,
		ControlPlaneConfig:              s.shoot.Spec.Provider.ControlPlaneConfig,
		ApiServerAclEnabled:             m.ApiServerAclEnabled,
		ExistingDNS:                     s.shoot.Spec.DNS,
		NetworkRestrictionGlobalEnabled: m.NetworkRestrictionGlobalEnabled,
	}

	if m.RegistryCacheConfigControllerEnabled {
		secretManager := registrycache.NewGardenSecretManager(m.GardenClient, fmt.Sprintf("garden-%s", m.ConverterConfig.Gardener.ProjectName), s.instance.Labels[imv1.LabelKymaRuntimeID])

		registryCacheGardenSecretNames, err := secretManager.GetCacheUIDToSecretNameMap(ctx)
		if err != nil {
			return patchOptions, err
		}

		patchOptions.RegistryCacheGardenSecretNames = registryCacheGardenSecretNames
	}

	return patchOptions, nil
}

// resolveAuditLogData determines which audit log configuration to use based on:
// - Global feature flag (DedicatedAuditLoggingEnabled)
// - Runtime flag (spec.auditLogAccessEnabled)
// - Existing AuditLog assignment
// Returns the audit log data in extender format and optional state transition in case of error
func resolveAuditLogData(ctx context.Context, m *fsm, s *systemState) (auditlogs.AuditLogData, stateFn, *ctrl.Result, error) {
	// Global feature disabled → use shared config
	if !m.DedicatedAuditLoggingEnabled {
		return getSharedAuditLogDataWithErrorHandling(ctx, m, s)
	}

	runtimeID := s.instance.Labels[imv1.LabelKymaRuntimeID]

	// Check if AuditLog is already assigned (irreversibility check)
	existingData, existingErr := m.AuditLogDataProvider.GetDedicatedAuditLogData(ctx, runtimeID, false)
	if existingErr == nil {
		if !s.instance.IsDedicatedAuditLogEnabled() {
			m.log.Info("Dedicated audit logging is irreversible - ignoring attempt to disable",
				"runtimeID", runtimeID)
		}
		return toExtenderAuditLogData(existingData), nil, nil, nil
	}

	if !s.instance.IsDedicatedAuditLogEnabled() {
		return getSharedAuditLogDataWithErrorHandling(ctx, m, s)
	}

	return claimDedicatedAuditLog(ctx, m, s, runtimeID)
}

// getSharedAuditLogDataWithErrorHandling retrieves shared config and handles errors
func getSharedAuditLogDataWithErrorHandling(ctx context.Context, m *fsm, s *systemState) (auditlogs.AuditLogData, stateFn, *ctrl.Result, error) {
	data, err := m.AuditLogDataProvider.GetSharedAuditLogData(
		ctx,
		s.instance.Spec.Shoot.Provider.Type,
		s.instance.Spec.Shoot.Region,
	)

	if err != nil {
		m.log.Error(err, msgFailedToConfigureAuditlogs)

		if m.AuditLogMandatory {
			m.Metrics.IncRuntimeFSMStopCounter()
			nextState, res, stateErr := updateStateFailedWithErrorAndStop(
				&s.instance,
				imv1.ConditionTypeRuntimeProvisioned,
				imv1.ConditionReasonAuditLogError,
				msgFailedToConfigureAuditlogs)
			return auditlogs.AuditLogData{}, nextState, res, stateErr
		}
	}

	return toExtenderAuditLogData(data), nil, nil, err
}

// claimDedicatedAuditLog claims an AuditLogCR for upgrade scenario
func claimDedicatedAuditLog(ctx context.Context, m *fsm, s *systemState, runtimeID string) (auditlogs.AuditLogData, stateFn, *ctrl.Result, error) {
	m.log.Info("Upgrading shared to dedicated audit logging",
		"runtimeID", runtimeID,
		"region", s.instance.Spec.Shoot.Region)

	data, err := m.AuditLogDataProvider.ClaimAuditLog(
		ctx,
		s.instance.Spec.Shoot.Region,
		runtimeID,
	)

	if err != nil {
		msg := fmt.Sprintf("Dedicated audit logging requested but no available configuration found: %v", err)
		m.log.Error(err, "Cannot upgrade runtime to dedicated audit logging")
		m.Metrics.IncRuntimeFSMStopCounter()
		nextState, res, stateErr := updateStateFailedWithErrorAndStop(
			&s.instance,
			imv1.ConditionTypeRuntimeProvisioned,
			imv1.ConditionReasonCustomAuditLogError,
			msg)
		return auditlogs.AuditLogData{}, nextState, res, stateErr
	}

	m.log.Info("Successfully claimed dedicated audit log for runtime upgrade",
		"runtimeID", runtimeID,
		"tenantID", data.TenantID)

	s.instance.UpdateStatePending(
		imv1.ConditionTypeCustomAuditLogConfigured,
		imv1.ConditionReasonCustomAuditLogConfigured,
		metav1.ConditionUnknown,
		"Dedicated audit logging claimed, configuring shoot",
	)

	return toExtenderAuditLogData(data), nil, nil, nil
}

func setRegistryCacheStatusFailed(ctx context.Context, m *fsm, s *systemState) {
	runtimeClient, err := m.RuntimeClientGetter.Get(ctx, s.instance)
	if err != nil {
		m.log.Error(err, "Failed to get Runtime Client to set Registry Cache status")
		return
	}

	if err = registrycache.NewStatusManager(runtimeClient).SetStatusFailed(ctx, s.instance, registrycacheapi.ConditionReasonRegistryCacheExtensionConfigurationFailed, "failed to apply registry cache configuration"); err != nil {
		m.log.Error(err, "Failed to set Registry Cache status to failed")
	}
}

// toExtenderAuditLogData converts auditlog.AuditLogData to extender auditlogs.AuditLogData
func toExtenderAuditLogData(data auditlog.AuditLogData) auditlogs.AuditLogData {
	return auditlogs.AuditLogData{
		TenantID:   data.TenantID,
		ServiceURL: data.ServiceURL,
		SecretName: data.SecretName,
	}
}
