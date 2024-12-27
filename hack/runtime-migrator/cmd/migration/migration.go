package main

import (
	"context"
	"fmt"
	"github.com/kyma-project/infrastructure-manager/hack/runtime-migrator-app/internal/shoot"
	"log/slog"

	gardener_types "github.com/gardener/gardener/pkg/client/core/clientset/versioned/typed/core/v1beta1"
	runtimev1 "github.com/kyma-project/infrastructure-manager/api/v1"
	config2 "github.com/kyma-project/infrastructure-manager/hack/runtime-migrator-app/internal/input"
	"github.com/kyma-project/infrastructure-manager/hack/runtime-migrator-app/internal/migration"
	"github.com/kyma-project/infrastructure-manager/pkg/config"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/kubeconfig"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/auditlogs"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Migration struct {
	runtimeMigrator migration.Migrator
	runtimeVerifier migration.Verifier
	kcpClient       client.Client
	shootClient     gardener_types.ShootInterface
	outputWriter    migration.OutputWriter
	isDryRun        bool
}

func NewMigration(migratorConfig config2.Config, converterConfig config.ConverterConfig, auditLogConfig auditlogs.Configuration, kubeconfigProvider kubeconfig.Provider, kcpClient client.Client, shootClient gardener_types.ShootInterface) (Migration, error) {

	outputWriter, err := migration.NewOutputWriter(migratorConfig.OutputPath)
	if err != nil {
		return Migration{}, err
	}

	return Migration{
		runtimeMigrator: migration.NewMigrator(migratorConfig, kubeconfigProvider, kcpClient),
		runtimeVerifier: migration.NewVerifier(converterConfig, auditLogConfig),
		kcpClient:       kcpClient,
		shootClient:     shootClient,
		outputWriter:    outputWriter,
		isDryRun:        migratorConfig.IsDryRun,
	}, nil
}

func (m Migration) Do(ctx context.Context, runtimeIDs []string) error {
	listCtx, cancel := context.WithTimeout(ctx, config2.TimeoutK8sOperation)
	defer cancel()

	shootList, err := m.shootClient.List(listCtx, v1.ListOptions{})
	if err != nil {
		return err
	}

	results := migration.NewMigratorResults(m.outputWriter.NewResultsDir)

	reportError := func(runtimeID, shootName string, msg string, err error) {

		if err != nil {
			msg = fmt.Sprintf("%s: %v", msg, err)
		}

		results.ErrorOccurred(runtimeID, shootName, msg)
		slog.Error(msg, "runtimeID", runtimeID)
	}

	reportValidationError := func(runtimeID, shootName string, msg string, err error) {
		errorMsg := fmt.Sprintf("%s: %v", msg, err)
		results.ValidationErrorOccurred(runtimeID, shootName, errorMsg)
		slog.Error(msg, "runtimeID", runtimeID)
	}

	reportUnwantedUpdateDetected := func(runtimeID, shootName string, msg string) {
		results.ValidationDetectedUnwantedUpdate(runtimeID, shootName)
		slog.Warn(msg, "runtimeID", runtimeID)
	}

	reportSuccess := func(runtimeID, shootName string, msg string) {
		results.OperationSucceeded(runtimeID, shootName)
		slog.Info(msg, "runtimeID", runtimeID)
	}

	run := func(runtimeID string) {
		shootToMigrate, err := shoot.Fetch(ctx, shootList, m.shootClient, runtimeID)
		if err != nil {
			reportError(runtimeID, "", "Failed to fetch shoot", err)
			return
		}

		if shoot.IsBeingDeleted(shootToMigrate) {
			reportError(runtimeID, shootToMigrate.Name, "Runtime is being deleted", nil)
			return
		}

		migrationCtx, cancel := context.WithTimeout(ctx, config2.TimeoutK8sOperation)
		defer cancel()

		runtime, err := m.runtimeMigrator.Do(migrationCtx, *shootToMigrate)

		if err != nil {
			reportError(runtimeID, shootToMigrate.Name, "Failed to migrate runtime", err)
			return
		}

		err = m.outputWriter.SaveRuntimeCR(runtime)
		if err != nil {
			reportError(runtimeID, shootToMigrate.Name, "Failed to save runtime CR", err)
			return
		}

		shootComparisonResult, err := m.runtimeVerifier.Do(runtime, *shootToMigrate)
		if err != nil {
			reportValidationError(runtimeID, shootToMigrate.Name, "Failed to verify runtime", err)
			return
		}

		if !shootComparisonResult.IsEqual() {
			err = m.outputWriter.SaveComparisonResult(shootComparisonResult)
			if err != nil {
				reportError(runtimeID, shootToMigrate.Name, "Failed to save comparison result", err)
				return
			}

			reportUnwantedUpdateDetected(runtimeID, shootToMigrate.Name, "Runtime CR can cause unwanted update in Gardener")
			return
		}

		if m.isDryRun {
			reportSuccess(runtimeID, shootToMigrate.Name, "Runtime processed successfully (dry-run)")
		} else {
			err = m.applyRuntimeCR(runtime)
			if err != nil {
				reportError(runtimeID, shootToMigrate.Name, "Failed to apply Runtime CR", err)
				return
			}

			reportSuccess(runtimeID, shootToMigrate.Name, "Runtime has been applied")
		}
	}

main:
	for _, runtimeID := range runtimeIDs {
		select {
		case <-ctx.Done():
			// application context was canceled
			reportError(runtimeID, "", "Processing interrupted", nil)
			break main

		default:
			run(runtimeID)
		}
	}

	resultsFile, err := m.outputWriter.SaveMigrationResults(results)
	if err != nil {
		return err
	}

	slog.Info(fmt.Sprintf("Migration completed. Successfully migrated runtimes: %d, Failed migrations: %d, Differences detected: %d", results.Succeeded, results.Failed, results.DifferenceDetected))
	slog.Info(fmt.Sprintf("Migration results saved in: %s", resultsFile))

	return nil
}

func (m Migration) applyRuntimeCR(runtime runtimev1.Runtime) error {
	// TODO: This method covers create scenario only, we should implement update as well
	return m.kcpClient.Create(context.Background(), &runtime)
}
