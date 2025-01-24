package main

import (
	"context"
	"log/slog"
	"os"
)

const LabelSelectorOld = "kyma-project.io/deprecation=to-be-removed-soon,reconciler.kyma-project.io/managed-by=provisioner"
const LabelSelectorNew = "reconciler.kyma-project.io/managed-by=infrastructure-manager"

func main() {
	cfg := ParseConfig()

	if cfg.Verbose {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	} else {
		slog.SetLogLoggerLevel(slog.LevelInfo)
	}

	kubectl := setupKubectl(cfg.Kubeconfig)
	client := kubectl.RbacV1().ClusterRoleBindings()
	fetcher := NewCRBFetcher(client, cfg.OldLabel, cfg.NewLabel)

	filer := NewJSONFiler(cfg.Output)

	var cleaner Cleaner
	if cfg.DryRun {
		slog.Info("Running in dry-run mode")
		cleaner = NewDryCleaner(filer)
	} else {
		cleaner = NewCRBCleaner(client)
	}

	failures, err := ProcessCRBs(fetcher, cleaner, filer, cfg)
	if err != nil {
		slog.Error("Error processing CRBs", "error", err)
		os.Exit(1)
	}
	err = filer.Failures(failures)
	if err != nil {
		slog.Error("Error marshaling list of failures", "error", err, "failures", failures)
		os.Exit(1)
	}
	slog.Info("Completed without errors")
}

// ProcessCRBs fetches old and new CRBs, compares them and cleans old CRBs
// It returns error on fetch errors
// It does nothing, if old CRBs are not found in new CRBs, unless force flag is set
// It returns list of failures on removal errors
func ProcessCRBs(fetcher Fetcher, cleaner Cleaner, filer Filer, cfg Config) ([]Failure, error) {
	ctx := context.Background()
	oldCRBs, err := fetcher.FetchOld(ctx)
	if err != nil {
		slog.Error("Error fetching old CRBs", "error", err)
		return nil, err
	}

	newCRBs, err := fetcher.FetchNew(ctx)
	if err != nil {
		slog.Error("Error fetching new CRBs", "error", err)
		return nil, err
	}

	compared := Compare(ctx, oldCRBs, newCRBs)

	if len(compared.missing) != 0 {
		slog.Warn("Old CRBs not found in new CRBs", CRBNames(compared.missing))
		if filer != nil {
			err := filer.Missing(compared.missing)
			if err != nil {
				slog.Error("Error saving unmatched CRBs", "error", err)
			}
		}
		if !cfg.Force {
			slog.Info("Use -force to remove old CRBs without match")
			return nil, nil
		}
	}

	return cleaner.Clean(ctx, oldCRBs), nil
}
