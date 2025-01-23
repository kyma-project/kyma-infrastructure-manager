package main

import (
	"flag"
	"log/slog"
	"os"
	"reflect"
	"strings"
)

type Config struct {
	Kubeconfig string
	DryRun     bool
	Verbose    bool
	Force      bool
	OldLabel   string
	NewLabel   string
	Output     string
}

func ParseConfig() Config {
	cfg := Config{}
	flag.StringVar(&cfg.Kubeconfig, "kubeconfig", "", "Kubeconfig file path")
	flag.StringVar(&cfg.OldLabel, "oldLabel", LabelSelectorOld, "Label marking old CRBs")
	flag.StringVar(&cfg.NewLabel, "newLabel", LabelSelectorNew, "Label marking new CRBs")
	flag.StringVar(&cfg.Output, "output", "", "Output folder for created files. Can also contain file prefix, if it doesn't end with `/` (can be a folder, eg ./foo/)")
	flag.BoolVar(&cfg.DryRun, "dry-run", true, "Don't remove CRBs, write what would be removed  to ./removed.json")
	flag.BoolVar(&cfg.Verbose, "verbose", false, "Increase the log level to debug (default: info)")
	flag.BoolVar(&cfg.Force, "force", false, "Force remove CRBs without checking migration status")
	flag.Parse()
	if cfg.Kubeconfig == "" {
		if k, ok := os.LookupEnv("KUBECONFIG"); ok {
			cfg.Kubeconfig = k
		}
	}
	slog.Info("Parsed config", Spread(cfg)...)
	return cfg
}

// Spread returns list of struct fields in the format ["field_name", "field_value", /* ... */]
func Spread(val interface{}) []interface{} {
	v := reflect.ValueOf(val)
	t := v.Type()

	var res []interface{}
	for i := 0; i < v.NumField(); i++ {
		res = append(res, strings.ToLower(t.Field(i).Name), v.Field(i).Interface())
	}

	return res
}
