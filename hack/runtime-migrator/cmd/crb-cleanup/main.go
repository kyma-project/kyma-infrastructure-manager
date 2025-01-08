package main

import (
	"context"
	"io"
	"log/slog"
	"os"

	"encoding/json"
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

	var cleaner Cleaner
	if cfg.Pretend {
		slog.Info("Running in pretend mode")
		file, err := os.OpenFile("./removed.json", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
		if err != nil {
			slog.Error("Error opening file, to save list of removed", "error", err)
			os.Exit(1)
		}
		defer file.Close()
		cleaner = NewPretendCleaner(file)
	} else {
		cleaner = NewCRBCleaner(client)
	}

	failureFile, err := os.OpenFile("./failures.json", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		slog.Error("Error opening file, to save list of failures", "error", err)
		os.Exit(1)
	}
	defer failureFile.Close()

	missingFile, err := os.OpenFile("./missing.json", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		slog.Error("Error opening file, to save list of failures", "error", err)
		os.Exit(1)
	}
	defer missingFile.Close()

	failures, err := ProcessCRBs(fetcher, cleaner, missingFile, cfg)
	if err != nil {
		slog.Error("Error processing CRBs", "error", err)
		os.Exit(1)
	}
	err = json.NewEncoder(failureFile).Encode(failures)
	if err != nil {
		slog.Error("Error marshaling list of failures", "error", err, "failures", failures)
		os.Exit(1)
	}
}

// ProcessCRBs fetches old and new CRBs, compares them and cleans old CRBs
// It returns error on fetch errors
// It does nothing, if old CRBs are not found in new CRBs, unless force flag is set
// It returns list of failures on removal errors
func ProcessCRBs(fetcher Fetcher, cleaner Cleaner, unmatched io.Writer, cfg Config) ([]Failure, error) {
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

	if len(compared.additional) != 0 {
		slog.Info("New CRBs not found in old CRBs", "crbs", compared.additional)
	}
	if len(compared.missing) != 0 {
		slog.Warn("Old CRBs not found in new CRBs", "crbs", compared.missing)
		if unmatched != nil {
			err := json.NewEncoder(unmatched).Encode(compared.missing)
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
