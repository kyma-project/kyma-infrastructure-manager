package runtime

import (
	"fmt"
	"sync"

	gardener_api "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clienttesting "k8s.io/client-go/testing"
)

// CustomTracker implements ObjectTracker with a sequence of Shoot objects
// it will be updated with a different shoot sequence for each test case
type CustomTracker struct {
	clienttesting.ObjectTracker
	shootSequence []*gardener_api.Shoot
	callCnt       int
	mu            sync.Mutex
}

func NewCustomTracker(tracker clienttesting.ObjectTracker, shoots []*gardener_api.Shoot) *CustomTracker {
	return &CustomTracker{
		ObjectTracker: tracker,
		shootSequence: shoots,
	}
}

func (t *CustomTracker) GetCallCnt() int {
	return t.callCnt
}

func (t *CustomTracker) IsSequenceFullyUsed() bool {
	return t.callCnt == len(t.shootSequence) && len(t.shootSequence) > 0
}

func (t *CustomTracker) Get(gvr schema.GroupVersionResource, ns, name string) (runtime.Object, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if gvr.Resource == "shoots" {
		if t.callCnt < len(t.shootSequence) {
			shoot := t.shootSequence[t.callCnt]
			t.callCnt++

			if shoot == nil {
				return nil, k8serrors.NewNotFound(schema.GroupResource{}, "")
			}
			return shoot, nil
		}
		return nil, fmt.Errorf("no more Shoot objects in sequence")
	}
	return t.ObjectTracker.Get(gvr, ns, name)
}

func (t *CustomTracker) Delete(gvr schema.GroupVersionResource, ns, name string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if gvr.Resource == "shoots" {
		for index, shoot := range t.shootSequence {
			if shoot != nil && shoot.Name == name {
				t.shootSequence[index] = nil
				return nil
			}
		}
		return k8serrors.NewNotFound(schema.GroupResource{}, "")
	}
	return t.ObjectTracker.Delete(gvr, ns, name)
}
