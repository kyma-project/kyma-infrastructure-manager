package runtimewatcher

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func TestExtractRuntimeIDFromMap(t *testing.T) {
	tests := []struct {
		name        string
		object      map[string]any
		expectedRID string
		expectedOk  bool
	}{
		{
			name:        "valid runtime-id",
			object:      map[string]any{"runtime-id": "rid-123"},
			expectedRID: "rid-123",
			expectedOk:  true,
		},
		{
			name:       "missing runtime-id key",
			object:     map[string]any{"other": "x"},
			expectedOk: false,
		},
		{
			name:       "runtime-id wrong type",
			object:     map[string]any{"runtime-id": 123},
			expectedOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &unstructured.Unstructured{Object: tt.object}
			rid, ok := extractRuntimeIDFromMap(u)
			assert.Equal(t, tt.expectedOk, ok)
			if tt.expectedOk {
				assert.Equal(t, tt.expectedRID, rid)
			}
		})
	}
}

func TestGetRuntimeIDFromEvent(t *testing.T) {
	tests := []struct {
		name        string
		object      map[string]any
		expectedRID string
		expectedErr error
	}{
		{
			name:        "valid event",
			object:      map[string]any{"runtime-id": "rid-456"},
			expectedRID: "rid-456",
		},
		{
			name:        "missing runtime-id",
			object:      map[string]any{"other": "x"},
			expectedErr: ErrExtractingRuntimeID,
		},
		{
			name:        "runtime-id wrong type",
			object:      map[string]any{"runtime-id": 123},
			expectedErr: ErrExtractingRuntimeID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evnt := event.GenericEvent{Object: &unstructured.Unstructured{Object: tt.object}}
			rid, err := getRuntimeIDFromEvent(evnt)
			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expectedRID, rid)
		})
	}
}

func TestAdaptEvents_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	src := make(chan WatcherListenerEvent, 1)
	dest := AdaptEvents(ctx, func() <-chan WatcherListenerEvent { return src })

	cancel()

	_, open := <-dest
	assert.False(t, open, "dest channel should be closed after context cancellation")
}

func TestAdaptEvents_SourceClose(t *testing.T) {
	src := make(chan WatcherListenerEvent, 1)
	dest := AdaptEvents(context.Background(), func() <-chan WatcherListenerEvent { return src })

	close(src)

	_, open := <-dest
	assert.False(t, open, "dest channel should be closed when source channel is closed")
}

func TestSkrEventHandler_GenericFunc(t *testing.T) {
	tests := []struct {
		name              string
		object            map[string]any
		expectedQueueLen  int
		expectedName      string
		expectedNamespace string
	}{
		{
			name:              "valid event adds to queue",
			object:            map[string]any{"runtime-id": "rid-789"},
			expectedQueueLen:  1,
			expectedName:      "kubeconfig-rid-789",
			expectedNamespace: "test-namespace",
		},
		{
			name:             "missing runtime-id does not add to queue",
			object:           map[string]any{"other": "x"},
			expectedQueueLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := CreateSkrEventHandler(logr.Discard(), "test-namespace")
			queue := workqueue.NewTypedRateLimitingQueue[ctrl.Request](
				workqueue.DefaultTypedControllerRateLimiter[ctrl.Request](),
			)
			defer queue.ShutDown()

			evnt := event.GenericEvent{Object: &unstructured.Unstructured{Object: tt.object}}
			h.GenericFunc(context.Background(), evnt, queue)

			require.Equal(t, tt.expectedQueueLen, queue.Len())
			if tt.expectedQueueLen > 0 {
				item, shutdown := queue.Get()
				require.False(t, shutdown)
				queue.Done(item)
				assert.Equal(t, tt.expectedName, item.Name)
				assert.Equal(t, tt.expectedNamespace, item.Namespace)
			}
		})
	}
}
