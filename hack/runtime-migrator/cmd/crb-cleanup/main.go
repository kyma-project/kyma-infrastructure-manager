package main

import (
	"context"
	"log/slog"
	"os"
)

const LabelSelectorProvisioner = "kyma-project.io/deprecation=to-be-removed-soon,reconciler.kyma-project.io/managed-by=provisioner"
const LabelSelectorKim = "reconciler.kyma-project.io/managed-by=infrastructure-manager"

func main() {
	cfg := ParseConfig()

	if cfg.Verbose {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	} else {
		slog.SetLogLoggerLevel(slog.LevelInfo)
	}

	kubectl := setupKubectl(cfg.Kubeconfig)
	client := kubectl.RbacV1().ClusterRoleBindings()
	fetcher := NewCRBFetcher(client, cfg.ProvisionerLabel, cfg.KimLabel)

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

// ProcessCRBs fetches provisioner's and kim's CRBs, compares them and cleans provisioner's CRBs
// It returns error on fetch errors
// It does nothing, if provisioner's CRBs are not found in kim's CRBs, unless force flag is set
// It returns list of failures on removal errors
func ProcessCRBs(fetcher Fetcher, cleaner Cleaner, filer Filer, cfg Config) ([]Failure, error) {
	ctx := context.Background()
	provisionerCRBs, err := fetcher.FetchProvisioner(ctx)
	if err != nil {
		slog.Error("Error fetching provisioner CRBs", "error", err)
		return nil, err
	}

	kimCRBs, err := fetcher.FetchKim(ctx)
	if err != nil {
		slog.Error("Error fetching kim CRBs", "error", err)
		return nil, err
	}

	compared := Compare(ctx, provisionerCRBs, kimCRBs)

	if len(compared.missing) != 0 {
		slog.Warn("Provisioner CRBs not found in kim CRBs", CRBNames(compared.missing))
		if filer != nil {
			err := filer.Missing(compared.missing)
			if err != nil {
				slog.Error("Error saving unmatched CRBs", "error", err)
			}
		}
		if !cfg.Force {
			slog.Info("Use -force to remove provisioner CRBs without match")
			return nil, nil
		}
	}

	return cleaner.Clean(ctx, provisionerCRBs), nil
}
