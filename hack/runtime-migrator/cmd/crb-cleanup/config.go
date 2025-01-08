package main

import (
	"flag"
	"log/slog"
	"reflect"
	"strings"
)

type Config struct {
	Kubeconfig string
	Pretend    bool
	Verbose    bool
	Force      bool
	OldLabel   string
	NewLabel   string
	Prefix     string
}

func ParseConfig() Config {
	cfg := Config{}
	flag.StringVar(&cfg.Kubeconfig, "kubeconfig", "", "Kubeconfig file path")
	flag.StringVar(&cfg.OldLabel, "oldLabel", LabelSelectorOld, "Label marking old CRBs")
	flag.StringVar(&cfg.NewLabel, "newLabel", LabelSelectorNew, "Label marking new CRBs")
	flag.BoolVar(&cfg.Pretend, "pretend", false, "Don't remove CRBs, write what would be removed  to ./removed.json")
	flag.BoolVar(&cfg.Verbose, "verbose", false, "Increase the log level to debug (default: info)")
	flag.BoolVar(&cfg.Force, "force", false, "Force remove CRBs without checking migration status")
	flag.Parse()
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
