package main

import (
	"context"
	"fmt"
	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardener_types "github.com/gardener/gardener/pkg/client/core/clientset/versioned/typed/core/v1beta1"
	runtimev1 "github.com/kyma-project/infrastructure-manager/api/v1"
	migrator "github.com/kyma-project/infrastructure-manager/hack/runtime-migrator-app/internal"
	"github.com/kyma-project/infrastructure-manager/hack/runtime-migrator-app/internal/runtime"
	"github.com/kyma-project/infrastructure-manager/pkg/config"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/kubeconfig"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log/slog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewMigration(migratorConfig migrator.Config, converterConfig config.ConverterConfig, kubeconfigProvider kubeconfig.Provider, kcpClient client.Client, shootClient gardener_types.ShootInterface) Migration {

	return Migration{
		runtimeMigrator: runtime.NewMigrator(converterConfig, kubeconfigProvider, kcpClient),
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

func (m Migration) Do(runtimeIDs []string) (MigrationResults, error) {

	shootList, err := m.shootClient.List(context.Background(), v1.ListOptions{})
	if err != nil {
		return MigrationResults{}, err
	}

	results := NewMigratorResults(m.migrationConfig.OutputPath)

	outputWriter, err := NewOutputWriter(m.migrationConfig.OutputPath)
	if err != nil {
		return MigrationResults{}, err
	}

	for _, runtimeID := range runtimeIDs {
		slog.Info(fmt.Sprintf("Migrating runtime with ID: %s", runtimeID))
		shoot := findShoot(runtimeID, shootList)
		if shoot == nil {
			msg := "Failed to find shoot"
			results.ErrorOccurred(runtimeID, "", msg)
			slog.Error(msg, "runtimeID", runtimeID)

			continue
		}

		runtime, err := m.runtimeMigrator.Do(*shoot)
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
			results.ValidationFailed(runtimeID, shoot.Name)
			slog.Error(msg, "runtimeID", runtimeID)

			continue
		}

		err = outputWriter.SaveComparisonResult(shootComparisonResult)
		if err != nil {
			slog.Error(fmt.Sprintf("Failed to save comparison result: %v", err), "runtimeID", runtimeID)

			continue
		}

		if shootComparisonResult.Diff != nil && !m.migrationConfig.IsDryRun {
			err = m.applyRuntimeCR(runtime)
			if err != nil {
				slog.Error(fmt.Sprintf("Failed to create runtime with ID: %s - %v", runtime.Name, err))
			}

			continue
		}
	}

	err = outputWriter.SaveMigrationResults(results)
	if err != nil {
		return results, err
	}

	return results, nil
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
