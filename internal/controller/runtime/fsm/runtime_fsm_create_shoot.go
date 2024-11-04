package fsm

import (
	"context"
	"fmt"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/config"
	gardener_shoot "github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/auditlogs"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	ErrInvalidShootSeedName          = fmt.Errorf("invalid shoot seed name")
	ErrProviderConfigurationNotFound = fmt.Errorf("provider configuration not found")
	ErrRegionConfigurationNotFound   = fmt.Errorf("provider configuration not found")
)

func sFnCreateShoot(ctx context.Context, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	m.log.Info("Create shoot state")

	shoot, err := convertCreate(&s.instance, m.ConverterConfig)
	if err != nil {
		m.log.Error(err, "Failed to convert Runtime instance to shoot object")
		m.Metrics.IncRuntimeFSMStopCounter()
		return updateStatePendingWithErrorAndStop(
			&s.instance,
			imv1.ConditionTypeRuntimeProvisioned,
			imv1.ConditionReasonConversionError,
			"Runtime conversion error")
	}

	err = m.ShootClient.Create(ctx, &shoot)
	if err != nil {
		m.log.Error(err, "Failed to create new gardener Shoot")
		s.instance.UpdateStatePending(
			imv1.ConditionTypeRuntimeProvisioned,
			imv1.ConditionReasonGardenerError,
			"False",
			fmt.Sprintf("Gardener API create error: %v", err),
		)
		return updateStatusAndRequeueAfter(m.GardenerRequeueDuration)
	}

	// audit log configuration contains 3 parts - ech may fail;
	// in case of failure the AuditLogMandatory flag controls the flow

	var data auditlogs.AuditLogData

	fetchConfiguration := func() error {
		fetchCfg := buildFetchAuditLogConfig(get(m.ShootClient.Get), m.AuditLogging)

		data, err = fetchCfg(ctx, shoot.Spec.SeedName, shoot.Spec.Region)
		return err
	}

	convert := func() error {
		auditlogConverter := gardener_shoot.NewAuditlogConverter(
			m.ConverterConfig.AuditLog.PolicyConfigMapName,
			data,
		)

		shoot, err = auditlogConverter.ToShoot(s.instance)
		return err
	}

	patchShoot := func() error {
		return m.Patch(ctx, &shoot, client.Apply, &client.PatchOptions{
			FieldManager: fieldManagerName,
			Force:        ptr.To(true),
		})
	}

	for _, fn := range []func() error{
		fetchConfiguration,
		convert,
		patchShoot,
	} {
		err = fn()

		if err == nil {
			continue
		}

		m.log.Error(err, "Failed to configure audit logs", "auditLogMandatory", m.AuditLogMandatory)

		if !m.AuditLogMandatory {
			break
		}

		s.instance.UpdateStatePending(
			imv1.ConditionTypeRuntimeProvisioned,
			imv1.ConditionReasonAuditLogError,
			"False",
			err.Error(),
		)
		return updateStatusAndRequeueAfter(m.GardenerRequeueDuration)
	}

	m.log.Info(
		"Gardener shoot for runtime initialised successfully",
		"Name", shoot.Name,
		"Namespace", shoot.Namespace,
	)

	s.instance.UpdateStatePending(
		imv1.ConditionTypeRuntimeProvisioned,
		imv1.ConditionReasonShootCreationPending,
		"Unknown",
		"Shoot is pending",
	)

	// it will be executed only once because created shoot is executed only once
	shouldDumpShootSpec := m.PVCPath != ""
	if shouldDumpShootSpec {
		s.shoot = shoot.DeepCopy()
		return switchState(sFnDumpShootSpec)
	}

	return updateStatusAndRequeueAfter(m.RCCfg.GardenerRequeueDuration)
}

func convertCreate(instance *imv1.Runtime, cfg config.ConverterConfig) (gardener.Shoot, error) {
	if err := instance.ValidateRequiredLabels(); err != nil {
		return gardener.Shoot{}, err
	}

	converter := gardener_shoot.NewConverterCreate(gardener_shoot.CreateOpts{
		ConverterConfig: cfg,
	})
	newShoot, err := converter.ToShoot(*instance)
	if err != nil {
		return newShoot, err
	}

	return newShoot, nil
}

type get func(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error

func (get get) providerType(ctx context.Context, seedName *string) (string, error) {
	isValidSeed := seedName != nil && *seedName != ""
	if !isValidSeed {
		return "", ErrInvalidShootSeedName
	}

	seedKey := types.NamespacedName{Name: *seedName, Namespace: ""}
	var seed gardener.Seed
	if err := get(ctx, seedKey, &seed); err != nil {
		return "", err
	}

	return seed.Spec.Provider.Type, nil
}

type providerConfig = map[string]auditlogs.AuditLogData

type auditLogDataMap = map[string]providerConfig

func buildFetchAuditLogConfig(get get, m auditLogDataMap) func(context.Context, *string, string) (auditlogs.AuditLogData, error) {
	return func(ctx context.Context, seedName *string, region string) (auditlogs.AuditLogData, error) {
		providerType, err := get.providerType(ctx, seedName)
		if err != nil {
			return auditlogs.AuditLogData{}, err
		}

		providerCfg, found := m[providerType]
		if !found {
			return auditlogs.AuditLogData{}, fmt.Errorf("%w: %s", ErrProviderConfigurationNotFound, providerType)
		}

		regionCfg, found := providerCfg[region]
		if !found {
			return auditlogs.AuditLogData{}, fmt.Errorf("%w: %s", ErrRegionConfigurationNotFound, region)
		}

		return regionCfg, nil
	}
}
