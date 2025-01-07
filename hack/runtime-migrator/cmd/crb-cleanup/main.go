package main

import (
	"context"
	"log/slog"
	"os"
)

// const LabelSelectorOld = "kyma-project.io/deprecation=to-be-removed-soon,reconciler.kyma-project.io/managed-by=provisioner"
const LabelSelectorOld = "reconciler.kyma-project.io/managed-by=infrastructure-manager"
const LabelSelectorNew = "reconciler.kyma-project.io/managed-by=infrastructure-manager"

func main() {
	cfg := ParseConfig()

	if cfg.Verbose {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	} else {
		slog.SetLogLoggerLevel(slog.LevelInfo)
	}

	client := setupKubectl(cfg.Kubeconfig).RbacV1().ClusterRoleBindings()
	fetcher := NewCRBFetcher(client, cfg.OldLabel, cfg.NewLabel)
	cleaner := NewCRBCleaner(client)

	ProcessCRBs(fetcher, cleaner, cfg)
}

func ProcessCRBs(fetcher Fetcher, cleaner Cleaner, cfg Config) []Failure {
	ctx := context.Background()
	oldCRBs, err := fetcher.FetchOld(ctx)
	if err != nil {
		slog.Error("Error fetching old CRBs", "error", err)
		os.Exit(1)
	}

	newCRBs, err := fetcher.FetchNew(ctx)
	if err != nil {
		slog.Error("Error fetching new CRBs", "error", err)
		os.Exit(1)
	}

	compared := cleaner.Compare(ctx, oldCRBs, newCRBs)

	if len(compared.additional) != 0 {
		slog.Info("New CRBs not found in old CRBs", "crbs", compared.additional)
	}
	if len(compared.missing) != 0 {
		slog.Warn("Old CRBs not found in new CRBs", "crbs", compared.missing)
		if !cfg.Force {
			slog.Info("Use -force to remove old CRBs without match")
			os.Exit(1)
		}
	}

	return cleaner.Clean(ctx, oldCRBs)
}
