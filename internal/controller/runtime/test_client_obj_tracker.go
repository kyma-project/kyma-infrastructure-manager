package runtime

import (
	"fmt"
	"sync"

	gardener_api "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	apimachinery "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clienttesting "k8s.io/client-go/testing"
)

const shootType = "shoots"
const seedType = "seeds"

// CustomTracker implements ObjectTracker with a sequence of Shoot objects
// it will be updated with a different shoot sequence for each test case
type CustomTracker struct {
	clienttesting.ObjectTracker
	shootSequence    []*gardener_api.Shoot
	seedListSequence []*gardener_api.SeedList
	shootCallCnt     int
	seedCallCnt      int
	mu               sync.Mutex
}

func NewCustomTracker(tracker clienttesting.ObjectTracker, shoots []*gardener_api.Shoot, seedLists []*gardener_api.SeedList) *CustomTracker {
	return &CustomTracker{
		ObjectTracker:    tracker,
		shootSequence:    shoots,
		seedListSequence: seedLists,
	}
}

func (t *CustomTracker) IsSequenceFullyUsed() bool {
	return t.shootCallCnt == len(t.shootSequence) && t.seedCallCnt == len(t.seedListSequence)
}

func (t *CustomTracker) Get(gvr schema.GroupVersionResource, ns, name string, opts ...apimachinery.GetOptions) (runtime.Object, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if gvr.Resource == shootType {
		return getNextObject(t.shootSequence, &t.shootCallCnt)
	}
	return t.ObjectTracker.Get(gvr, ns, name, opts...)
}

func (t *CustomTracker) List(gvr schema.GroupVersionResource, gvk schema.GroupVersionKind, ns string, opts ...apimachinery.ListOptions) (runtime.Object, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if gvr.Resource == seedType {
		return getNextObject(t.seedListSequence, &t.seedCallCnt)
	}

	return t.ObjectTracker.List(gvr, gvk, ns, opts...)
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

func (t *CustomTracker) Update(gvr schema.GroupVersionResource, obj runtime.Object, ns string, opts ...apimachinery.UpdateOptions) error {
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
	return t.ObjectTracker.Update(gvr, obj, ns, opts...)
}

func (t *CustomTracker) Delete(gvr schema.GroupVersionResource, ns, name string, opts ...apimachinery.DeleteOptions) error {
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
	return t.ObjectTracker.Delete(gvr, ns, name, opts...)
}
