package main

import (
	"context"
	"fmt"
	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardener_types "github.com/gardener/gardener/pkg/client/core/clientset/versioned/typed/core/v1beta1"
	runtimev1 "github.com/kyma-project/infrastructure-manager/api/v1"
	migrator "github.com/kyma-project/infrastructure-manager/hack/runtime-migrator-app/internal"
	"github.com/kyma-project/infrastructure-manager/hack/runtime-migrator-app/internal/migration"
	"github.com/kyma-project/infrastructure-manager/hack/runtime-migrator-app/internal/runtime"
	"github.com/kyma-project/infrastructure-manager/pkg/config"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/kubeconfig"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log/slog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewMigration(migratorConfig migrator.Config, converterConfig config.ConverterConfig, kubeconfigProvider kubeconfig.Provider, kcpClient client.Client, shootClient gardener_types.ShootInterface) Migration {

	return Migration{
		runtimeMigrator: runtime.NewMigrator(migratorConfig, converterConfig, kubeconfigProvider, kcpClient),
		runtimeVerifier: runtime.NewVerifier(converterConfig, migratorConfig.OutputPath),
		migrationConfig: migratorConfig,
		kcpClient:       kcpClient,
		shootClient:     shootClient,
	}

}

type Migration struct {
	runtimeMigrator runtime.Migrator
	runtimeVerifier runtime.Verifier
	migrationConfig migrator.Config
	kcpClient       client.Client
	shootClient     gardener_types.ShootInterface
}

func (m Migration) Do(runtimeIDs []string) error {

	outputWriter, err := migration.NewOutputWriter(m.migrationConfig.OutputPath)
	if err != nil {
		return err
	}

	shootList, err := m.shootClient.List(context.Background(), v1.ListOptions{})
	if err != nil {
		return err
	}

	results := migration.NewMigratorResults(outputWriter.NewResultsDir)

	for _, runtimeID := range runtimeIDs {
		slog.Info(fmt.Sprintf("Migrating runtime with ID: %s", runtimeID))
		shoot := findShoot(runtimeID, shootList)
		if shoot == nil {
			msg := "Failed to find shoot"
			results.ErrorOccurred(runtimeID, "", msg)
			slog.Error(msg, "runtimeID", runtimeID)

			continue
		}

		runtime, err := m.runtimeMigrator.Do(context.Background(), *shoot)
		if err != nil {
			msg := "Failed to migrate runtime"
			results.ErrorOccurred(runtimeID, shoot.Name, msg)
			slog.Error(msg, "runtimeID", runtimeID)

			continue
		}

		err = outputWriter.SaveRuntimeCR(runtime)
		if err != nil {
			slog.Error(fmt.Sprintf("Failed to save runtime CR: %v", err), "runtimeID", runtimeID)

			continue
		}

		shootComparisonResult, err := m.runtimeVerifier.Do(runtime, *shoot)
		if err != nil {
			msg := "Failed to verify runtime"
			results.ErrorOccurred(runtimeID, shoot.Name, msg)
			slog.Error(msg, "runtimeID", runtimeID)

			continue
		}

		if shootComparisonResult.IsEqual() && !m.migrationConfig.IsDryRun {
			err = m.applyRuntimeCR(runtime)
			if err != nil {
				msg := "Failed to apply Runtime CR"
				results.ErrorOccurred(runtimeID, shoot.Name, msg)
				slog.Error(fmt.Sprintf("Failed to apply runtime with ID: %s - %v", runtime.Name, err))
			}

			continue
		}

		if shootComparisonResult.IsEqual() {
			results.OperationSucceeded(runtimeID, shoot.Name)
		} else {
			err = outputWriter.SaveComparisonResult(shootComparisonResult)
			if err != nil {
				msg := "Failed to store comparison results"
				results.ErrorOccurred(runtimeID, shoot.Name, msg)
				slog.Error(fmt.Sprintf("Failed to save comparison result: %v", err), "runtimeID", runtimeID)

				continue
			}

			results.ValidationFailed(runtimeID, shoot.Name)
		}
	}

	resultsFile, err := outputWriter.SaveMigrationResults(results)
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
	// TODO: This method covers create scenario onyl, we should implement update as well
	return m.kcpClient.Create(context.Background(), &runtime)
}
