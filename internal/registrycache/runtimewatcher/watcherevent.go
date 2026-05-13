package runtimewatcher

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/kyma-project/runtime-watcher/listener/pkg/v2/types"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
)

type (
	WatcherListenerEvent = types.GenericEvent
	CtrlRuntimeEvent     = event.GenericEvent
)

var (
	ErrHandlingWatcherEvent   = errors.New("error handling watcher event")
	ErrConvertingWatcherEvent = errors.New("failed to convert event object")
	ErrExtractingRuntimeID    = errors.New("failed to extract runtime ID from event data")
)

// AdaptEvents converts given channel from the type used by runtime-watcher/listener
// module to the type required by the controller-runtime library.
func AdaptEvents(listenerChan func() <-chan WatcherListenerEvent) <-chan CtrlRuntimeEvent {
	dest := make(chan CtrlRuntimeEvent)
	go func() {
		defer close(dest)
		for evt := range listenerChan() {
			dest <- CtrlRuntimeEvent{Object: evt.Object}
		}
	}()
	return dest
}

func CreateSkrEventHandler(l logr.Logger) *handler.Funcs {
	return &handler.Funcs{
		GenericFunc: func(ctx context.Context, evnt event.GenericEvent,
			queue workqueue.TypedRateLimitingInterface[ctrl.Request],
		) {
			runtimeID, err := getRuntimeIDFromEvent(evnt)
			if err != nil {
				l.Error(fmt.Errorf("%w: %w", ErrHandlingWatcherEvent, err), fmt.Sprintf("event: %v", evnt.Object))
				return
			}

			kcpKubeconfigKey := client.ObjectKey{
				Name:      "kubeconfig-" + runtimeID,
				Namespace: "kcp-system",
			}
			req := ctrl.Request{NamespacedName: kcpKubeconfigKey}
			l.Info(fmt.Sprintf("event received from SKR, adding %s to queue", req.NamespacedName))

			queue.Add(req)
		},
	}
}

func getRuntimeIDFromEvent(evnt event.GenericEvent) (string, error) {
	unstruct, ok := evnt.Object.(*unstructured.Unstructured)
	if !ok {
		return "", ErrConvertingWatcherEvent
	}
	runtimeID, ok := extractRuntimeIDFromMap(unstruct)
	if !ok {
		return "", ErrExtractingRuntimeID
	}
	return runtimeID, nil
}

func extractRuntimeIDFromMap(unstructuredEvent *unstructured.Unstructured) (string, bool) {
	runtimeId, ok := unstructuredEvent.Object["runtime-id"]
	if !ok {
		return "", false
	}
	s, ok := runtimeId.(string)
	if !ok {
		return "", false
	}
	return s, true
}
