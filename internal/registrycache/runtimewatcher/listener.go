package runtimewatcher

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-logr/logr"
	watchertypes "github.com/kyma-project/runtime-watcher/listener/pkg/v2/types"
)

const defaultPort = 8082

type RegistryCacheConfigListener struct {
	Addr          string
	Logger        logr.Logger
	ComponentName string
	events        chan watchertypes.GenericEvent
}

func NewRegistryCacheConfigListener(addr string, componentName string, logger logr.Logger) *RegistryCacheConfigListener {
	return &RegistryCacheConfigListener{
		Addr:          addr,
		ComponentName: componentName,
		Logger:        logger,
	}
}

}

func (l *RegistryCacheConfigListener) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc(fmt.Sprintf("/v2/%s/event", l.ComponentName), l.handleEvent)

	server := &http.Server{
	server := &http.Server{
		Addr:    l.Addr,
		Handler: mux,
	}

		Handler: mux,
	}

	go func() {
		l.Logger.Info("Starting registry cache config listener", "addr", server.Addr, "component", l.ComponentName)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			l.Logger.Error(err, "Error starting registry cache config listener", "component", l.ComponentName)
		}
	}()

	<-ctx.Done()
	l.Logger.Info("Shutting down registry cache config listener", "component", l.ComponentName)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		l.Logger.Error(err, "failed to shut down listener HTTP server")
	}

		l.Logger.Error(err, "failed to shut down listener HTTP server")
	}

	return nil
}

func (l *RegistryCacheConfigListener) handleEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		l.Logger.Error(nil, "method not allowed", "method", r.Method, "path", r.URL.Path)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var watchEvent watchertypes.WatchEvent
	if err := json.NewDecoder(r.Body).Decode(&watchEvent); err != nil {
		l.Logger.Error(err, "failed to decode watch event")
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// TODO: convert watchEvent to a generic type and send it to the channel for processing by the reconciler
	l.Logger.Info("Received watch event for registry cache", "event", watchEvent)

	w.WriteHeader(http.StatusOK)
}
