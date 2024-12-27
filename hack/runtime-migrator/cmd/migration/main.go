package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/auditlogs"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/kyma-project/infrastructure-manager/hack/runtime-migrator-app/internal/input"
	kimConfig "github.com/kyma-project/infrastructure-manager/pkg/config"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const (
	expirationTime = 60 * time.Minute
)

func main() {
	slog.Info("Starting runtime-migrator")
	cfg := input.NewConfig()

	opts := zap.Options{
		Development: true,
	}
	logger := zap.New(zap.UseFlagOptions(&opts))
	logf.SetLogger(logger)

	gardenerNamespace := fmt.Sprintf("garden-%s", cfg.GardenerProjectName)

	kubeconfigProvider, err := input.SetupKubernetesKubeconfigProvider(cfg.GardenerKubeconfigPath, gardenerNamespace, expirationTime)
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to create kubeconfig provider: %v", err))
		os.Exit(1)
	}

	kcpClient, err := input.CreateKcpClient(&cfg)
	if err != nil {
		slog.Error("Failed to create kcp client", slog.Any("error", err))
		os.Exit(1)
	}

	gardenerShootClient, err := input.SetupGardenerShootClient(cfg.GardenerKubeconfigPath, gardenerNamespace)
	if err != nil {
		slog.Error("Failed to setup Gardener shoot client", slog.Any("error", err))
		os.Exit(1)
	}

	auditLogConfig, err := getAuditLogConfig(kcpClient)
	if err != nil {
		slog.Error("Failed to get audit log config", slog.Any("error", err))
		os.Exit(1)
	}

	converterConfig, err := getConverterConfig(kcpClient)
	if err != nil {
		slog.Error("Failed to get converter config", slog.Any("error", err))
		os.Exit(1)
	}

	slog.Info("Migrating runtimes")
	migrator, err := NewMigration(cfg, converterConfig, auditLogConfig, kubeconfigProvider, kcpClient, gardenerShootClient)
	if err != nil {
		slog.Error("Failed to create migrator", slog.Any("error", err))
		os.Exit(1)
	}

	slog.Info("Reading runtimeIds from input file")
	runtimeIds, err := input.GetRuntimeIDsFromInputFile(cfg)
	if err != nil {
		slog.Error("Failed to read runtime Ids from input", slog.Any("error", err))
		os.Exit(1)
	}

	err = migrator.Do(context.Background(), runtimeIds)
	if err != nil {
		slog.Error("Failed to migrate runtimes", slog.Any("error", err))
		os.Exit(1)
	}
}

func getConverterConfig(kcpClient client.Client) (kimConfig.ConverterConfig, error) {
	var cm v12.ConfigMap
	key := types.NamespacedName{
		Name:      "infrastructure-manager-converter-config",
		Namespace: "kcp-system",
	}

	err := kcpClient.Get(context.Background(), key, &cm)
	if err != nil {
		return kimConfig.ConverterConfig{}, err
	}

	getReader := func() (io.Reader, error) {
		data, found := cm.Data["converter_config.json"]
		if !found {
			return nil, fmt.Errorf("converter_config.json not found in ConfigMap")
		}
		return strings.NewReader(data), nil
	}

	var cfg kimConfig.Config
	if err = cfg.Load(getReader); err != nil {
		return kimConfig.ConverterConfig{}, err
	}

	return cfg.ConverterConfig, nil
}

func getAuditLogConfig(kcpClient client.Client) (auditlogs.Configuration, error) {
	var cm v12.ConfigMap
	key := types.NamespacedName{
		Name:      "audit-extension-config",
		Namespace: "kcp-system",
	}

	err := kcpClient.Get(context.Background(), key, &cm)
	if err != nil {
		return nil, err
	}

	configBytes := []byte(cm.Data["config"])

	var data auditlogs.Configuration
	if err := json.Unmarshal(configBytes, &data); err != nil {
		return nil, err
	}

	validate := validator.New(validator.WithRequiredStructEnabled())

	for _, nestedMap := range data {
		for _, auditLogData := range nestedMap {
			if err := validate.Struct(auditLogData); err != nil {
				return nil, err
			}
		}
	}

	return data, nil
}
