package main

import (
	"flag"
	"log/slog"
	"os"
	"reflect"
	"strings"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/providers/basicflag"
)

var k = koanf.New("/")

type Config struct {
	Kubeconfig string `koanf:"kubeconfig"`
	Pretend    bool   `koanf:"pretend"`
	Verbose    bool   `koanf:"verbose"`
	Force      bool   `koanf:"force"`
	OldLabel   string `koanf:"oldLabel"`
	NewLabel   string `koanf:"newLabel"`
}

func ParseConfig() Config {
	fs := flag.NewFlagSet("", flag.ExitOnError)
	fs.String("kubeconfig", "", "Kubeconfig file path")
	fs.String("oldLabel", LabelSelectorOld, "Label marking old CRBs")
	fs.String("newLabel", LabelSelectorNew, "Label marking new CRBs")
	fs.Bool("pretend", false, "Don't remove CRBs, write what would be removed  to ./removed.json")
	fs.Bool("verbose", false, "Increase the log level to debug (default: info)")
	fs.Bool("force", false, "Force remove CRBs without checking migration status")

	if err := fs.Parse(os.Args[1:]); err != nil {
		slog.Error("Error parsing flags", "error", err)
		os.Exit(1)
	}

	if err := k.Load(basicflag.Provider(fs, "/"), nil); err != nil {
		slog.Error("Error loading config", "error", err)
		os.Exit(1)
	}

	cfg := Config{
		OldLabel: LabelSelectorOld,
		NewLabel: LabelSelectorNew,
	}
	if err := k.Unmarshal("", &cfg); err != nil {
		slog.Error("Error unmarshalling config", "error", err)
		os.Exit(1)
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
