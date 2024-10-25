package main

import (
	"context"
	"fmt"
	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardener_types "github.com/gardener/gardener/pkg/client/core/clientset/versioned/typed/core/v1beta1"
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
		migrationConfig:    migratorConfig,
		converterConfig:    converterConfig,
		kubeconfigProvider: kubeconfigProvider,
		kcpClient:          kcpClient,
		shootClient:        shootClient,
	}

}

type Migration struct {
	migrationConfig    migrator.Config
	converterConfig    config.ConverterConfig
	kubeconfigProvider kubeconfig.Provider
	kcpClient          client.Client
	shootClient        gardener_types.ShootInterface
}

func (m Migration) Do(runtimeIDs []string) (MigrationResults, error) {
	runtimeMigrator := runtime.NewMigrator(m.converterConfig, m.kubeconfigProvider, m.kcpClient)
	runtimeVerifier := runtime.NewVerifier(m.converterConfig, m.migrationConfig.OutputPath)

	shootList, err := m.shootClient.List(context.Background(), v1.ListOptions{})
	if err != nil {
		return MigrationResults{}, err
	}

	findShoot := func(runtimeID string) *v1beta1.Shoot {
		for _, shoot := range shootList.Items {
			if shoot.Annotations[runtimeIDAnnotation] == runtimeID {
				return &shoot
			}
		}
		return nil
	}

	results := NewMigratorResults(m.migrationConfig.OutputPath)

	for _, runtimeID := range runtimeIDs {
		slog.Info(fmt.Sprintf("Migrating runtime with ID: %s", runtimeID))
		shoot := findShoot(runtimeID)
		if shoot == nil {
			msg := "Failed to find shoot"
			results.ErrorOccurred(runtimeID, "", msg)
			slog.Error(msg, "runtimeID", runtimeID)

			continue
		}

		runtime, err := runtimeMigrator.Do(*shoot)
		if err != nil {
			msg := "Failed to migrate runtime"
			results.ErrorOccurred(runtimeID, shoot.Name, msg)
			slog.Error(msg, "runtimeID", runtimeID)

			continue
		}

		equal, err := runtimeVerifier.Do(runtime, *shoot)
		if err != nil {
			msg := "Failed to verify runtime"
			results.ValidationFailed(runtimeID, shoot.Name)
			slog.Error(msg, "runtimeID", runtimeID)

			continue
		}

		if equal && !m.migrationConfig.IsDryRun {
			err := m.kcpClient.Create(context.Background(), &runtime)
			if err != nil {
				slog.Error(fmt.Sprintf("Failed to create runtime with ID: %s - %v", runtimeID, err))
				continue
			}
		}
	}

	return results, nil
}
