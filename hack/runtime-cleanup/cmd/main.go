package main

import (
	"github.com/kyma-project/infrastructure-manager/hack/runtime-cleanup/cmd/cleaner"
	"log/slog"
	"os"
)

func main() {
	textHandler := slog.NewTextHandler(os.Stdout, nil)
	log := slog.New(textHandler)
	log.Info("Starting runtime cleanup")
	err := cleaner.Execute()
	if err != nil {
		log.Error("Error during running runtime cleanup ", err)
	}
	return
}
