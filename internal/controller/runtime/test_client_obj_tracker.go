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

const shootType = "shoots"

// CustomTracker implements ObjectTracker with a sequence of Shoot objects
// it will be updated with a different shoot sequence for each test case
type CustomTracker struct {
	clienttesting.ObjectTracker
	shootSequence []*gardener_api.Shoot
	seedSequence  []*gardener_api.Seed
	shootCallCnt  int
	seedCallCnt   int
	mu            sync.Mutex
}

func NewCustomTracker(tracker clienttesting.ObjectTracker, shoots []*gardener_api.Shoot, seeds []*gardener_api.Seed) *CustomTracker {
	return &CustomTracker{
		ObjectTracker: tracker,
		shootSequence: shoots,
		seedSequence:  seeds,
	}
}

func (t *CustomTracker) IsSequenceFullyUsed() bool {
	return (t.shootCallCnt == len(t.shootSequence) && len(t.shootSequence) > 0) && t.seedCallCnt == len(t.seedSequence)
}

func (t *CustomTracker) Get(gvr schema.GroupVersionResource, ns, name string) (runtime.Object, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if gvr.Resource == shootType {
		return getNextObject(t.shootSequence, &t.shootCallCnt)
	} else if gvr.Resource == "seeds" {
		return getNextObject(t.seedSequence, &t.seedCallCnt)
	}

	return t.ObjectTracker.Get(gvr, ns, name)
}

func getNextObject[T any](sequence []*T, counter *int) (*T, error) {
	if *counter < len(sequence) {
		obj := sequence[*counter]
		*counter++

		if obj == nil {
			return nil, k8serrors.NewNotFound(schema.GroupResource{}, "")
		}
		return obj, nil
	}
	return nil, fmt.Errorf("no more objects in sequence")
}

func (t *CustomTracker) Update(gvr schema.GroupVersionResource, obj runtime.Object, ns string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if gvr.Resource == shootType {
		shoot, ok := obj.(*gardener_api.Shoot)
		if !ok {
			return fmt.Errorf("object is not of type Gardener Shoot")
		}
		for index, existingShoot := range t.shootSequence {
			if existingShoot != nil && existingShoot.Name == shoot.Name {
				t.shootSequence[index] = shoot
				return nil
			}
		}
		return k8serrors.NewNotFound(schema.GroupResource{}, shoot.Name)
	}
	return t.ObjectTracker.Update(gvr, obj, ns)
}

func (t *CustomTracker) Delete(gvr schema.GroupVersionResource, ns, name string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if gvr.Resource == shootType {
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
