package main

import (
	"context"
	"fmt"
	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardener_types "github.com/gardener/gardener/pkg/client/core/clientset/versioned/typed/core/v1beta1"
	runtimev1 "github.com/kyma-project/infrastructure-manager/api/v1"
	config2 "github.com/kyma-project/infrastructure-manager/hack/runtime-migrator-app/internal/config"
	"github.com/kyma-project/infrastructure-manager/hack/runtime-migrator-app/internal/migration"
	"github.com/kyma-project/infrastructure-manager/hack/runtime-migrator-app/internal/runtime"
	"github.com/kyma-project/infrastructure-manager/pkg/config"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/kubeconfig"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log/slog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewMigration(migratorConfig config2.Config, converterConfig config.ConverterConfig, kubeconfigProvider kubeconfig.Provider, kcpClient client.Client, shootClient gardener_types.ShootInterface) (Migration, error) {

	outputWriter, err := migration.NewOutputWriter(migratorConfig.OutputPath)
	if err != nil {
		return Migration{}, err
	}

	return Migration{
		runtimeMigrator: runtime.NewMigrator(migratorConfig, converterConfig, kubeconfigProvider, kcpClient),
		runtimeVerifier: runtime.NewVerifier(converterConfig, migratorConfig.OutputPath),
		kcpClient:       kcpClient,
		shootClient:     shootClient,
		outputWriter:    outputWriter,
		isDryRun:        migratorConfig.IsDryRun,
	}, nil
}

type Migration struct {
	runtimeMigrator runtime.Migrator
	runtimeVerifier runtime.Verifier
	kcpClient       client.Client
	shootClient     gardener_types.ShootInterface
	outputWriter    migration.OutputWriter
	isDryRun        bool
}

func (m Migration) Do(runtimeIDs []string) error {

	migratorContext, cancel := context.WithTimeout(context.Background(), contextTimeout)
	defer cancel()

	shootList, err := m.shootClient.List(migratorContext, v1.ListOptions{})
	if err != nil {
		return err
	}

	results := migration.NewMigratorResults(m.outputWriter.NewResultsDir)

	reportError := func(runtimeID, shootName string, msg string, err error) {
		var errorMsg string

		if err != nil {
			errorMsg = fmt.Sprintf("%s: %v", msg, err)
		} else {
			errorMsg = fmt.Sprintf(msg)
		}

		results.ErrorOccurred(runtimeID, shootName, errorMsg)
		slog.Error(errorMsg, "runtimeID", runtimeID)
	}

	reportValidationError := func(runtimeID, shootName string, msg string, err error) {
		errorMsg := fmt.Sprintf("%s: %v", msg, err)
		results.ValidationErrorOccurred(runtimeID, shootName, errorMsg)
		slog.Warn(msg, "runtimeID", runtimeID)
	}

	reportUnwantedUpdateDetected := func(runtimeID, shootName string, msg string) {
		results.ValidationDetectedUnwantedUpdate(runtimeID, shootName)
		slog.Info(msg, "runtimeID", runtimeID)
	}

	reportSuccess := func(runtimeID, shootName string, msg string) {
		results.OperationSucceeded(runtimeID, shootName)
		slog.Info(msg, "runtimeID", runtimeID)
	}

	for _, runtimeID := range runtimeIDs {
		shoot := findShoot(runtimeID, shootList)
		if shoot == nil {
			reportError(runtimeID, "", "Failed to find shoot", nil)

			continue
		}

		runtime, err := m.runtimeMigrator.Do(migratorContext, *shoot)
		if err != nil {
			reportError(runtimeID, shoot.Name, "Failed to migrate runtime", err)

			continue
		}

		err = m.outputWriter.SaveRuntimeCR(runtime)
		if err != nil {
			reportError(runtimeID, shoot.Name, "Failed to save runtime CR", err)

			continue
		}

		shootComparisonResult, err := m.runtimeVerifier.Do(runtime, *shoot)
		if err != nil {
			reportValidationError(runtimeID, shoot.Name, "Failed to verify runtime", err)

			continue
		}

		if !shootComparisonResult.IsEqual() {
			err = m.outputWriter.SaveComparisonResult(shootComparisonResult)
			if err != nil {
				reportError(runtimeID, shoot.Name, "Failed to save comparison result", err)
			} else {
				reportUnwantedUpdateDetected(runtimeID, shoot.Name, "Runtime CR can cause unwanted update in Gardener. Please verify the runtime CR.")
			}

			continue
		}

		if !m.isDryRun {
			err = m.applyRuntimeCR(runtime)
			if err != nil {
				reportError(runtimeID, shoot.Name, "Failed to apply Runtime CR", err)
			}

			continue
		}

		reportSuccess(runtimeID, shoot.Name, "Runtime processed successfully")
	}

	resultsFile, err := m.outputWriter.SaveMigrationResults(results)
	if err != nil {
		return err
	}

	slog.Info(fmt.Sprintf("Migration completed. Successfully migrated runtimes: %d, Failed migrations: %d, Differences detected: %d", results.Succeeded, results.Failed, results.DifferenceDetected))
	slog.Info(fmt.Sprintf("Migration results saved in: %s", resultsFile))

	return nil
}

func findShoot(runtimeID string, shootList *v1beta1.ShootList) *v1beta1.Shoot {
	for _, shoot := range shootList.Items {
		if shoot.Annotations[runtimeIDAnnotation] == runtimeID {
			return &shoot
		}
	}
	return nil
}

func (m Migration) applyRuntimeCR(runtime runtimev1.Runtime) error {
	// TODO: This method covers create scenario only, we should implement update as well
	return m.kcpClient.Create(context.Background(), &runtime)
}
