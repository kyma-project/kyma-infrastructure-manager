package runtime

import (
	"testing"

	gardener_api "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestCustomTracker_Get(t *testing.T) {
	seedSequence := []*gardener_api.Seed{{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "seed",
			Namespace: "test",
		},
	}}

	shootSequence := []*gardener_api.Shoot{{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "shoot",
			Namespace: "test",
		},
	}}

	t.Run("should return shoot object", func(t *testing.T) {
		// given
		tracker := NewCustomTracker(nil, shootSequence, seedSequence)
		gvr := schema.GroupVersionResource{
			Resource: "shoots",
		}

		// when
		obj, err := tracker.Get(gvr, "test", "shoot")

		// then
		require.NoError(t, err)
		require.NotNil(t, obj)
		require.Equal(t, shootSequence[0], obj)
	})

	t.Run("should return seed object", func(t *testing.T) {
		// given
		tracker := NewCustomTracker(nil, shootSequence, seedSequence)
		gvr := schema.GroupVersionResource{
			Resource: "seeds",
		}

		// when
		obj, err := tracker.Get(gvr, "test", "seed")

		// then
		require.NoError(t, err)
		require.NotNil(t, obj)
		require.Equal(t, seedSequence[0], obj)
	})

	t.Run("should return not found error", func(t *testing.T) {
		// given
		tracker := NewCustomTracker(nil, shootSequence, seedSequence)
		gvr := schema.GroupVersionResource{
			Resource: "shoots",
		}

		// when
		obj, err := tracker.Get(gvr, "test", "shoot")
		objOutOfRange, errOutOfRange := tracker.Get(gvr, "test", "outOfRange")

		// then
		require.NoError(t, err)
		require.NotNil(t, obj)
		require.Error(t, errOutOfRange)
		require.Nil(t, objOutOfRange)
	})
}

func TestCustomTracker_Update(t *testing.T) {
	t.Run("should update shoot object", func(t *testing.T) {
		// given
		shootSequence := []*gardener_api.Shoot{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "shoot",
				Namespace: "test",
			},
		}}
		tracker := NewCustomTracker(nil, shootSequence, nil)
		gvr := schema.GroupVersionResource{
			Resource: "shoots",
		}
		newShoot := &gardener_api.Shoot{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "shoot",
				Namespace: "test",
			},
			Spec: gardener_api.ShootSpec{
				Region: "afterUpdate",
			},
		}

		// when
		err := tracker.Update(gvr, newShoot, "test")

		// then
		require.NoError(t, err)
		require.Equal(t, newShoot, shootSequence[0])
	})

	t.Run("should return not found error", func(t *testing.T) {
		// given
		shootSequence := []*gardener_api.Shoot{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "shoot",
				Namespace: "test",
			},
		}}
		tracker := NewCustomTracker(nil, shootSequence, nil)
		gvr := schema.GroupVersionResource{
			Resource: "shoots",
		}
		newShoot := &gardener_api.Shoot{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "shoot2",
				Namespace: "test",
			},
			Spec: gardener_api.ShootSpec{
				Region: "afterUpdate",
			},
		}

		// when
		err := tracker.Update(gvr, newShoot, "test")

		// then
		require.Error(t, err)
	})
}

func TestCustomTracker_Delete(t *testing.T) {
	t.Run("should delete shoot object.", func(t *testing.T) {
		// given
		shootSequence := []*gardener_api.Shoot{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "shoot",
				Namespace: "test",
			},
		}}
		tracker := NewCustomTracker(nil, shootSequence, nil)
		gvr := schema.GroupVersionResource{
			Resource: "shoots",
		}

		// when
		err := tracker.Delete(gvr, "test", "shoot")

		// then
		require.NoError(t, err)
		require.Nil(t, shootSequence[0])
	})

	t.Run("should return not found error", func(t *testing.T) {
		// given
		shootSequence := []*gardener_api.Shoot{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "shoot",
				Namespace: "test",
			},
		}}
		tracker := NewCustomTracker(nil, shootSequence, nil)
		gvr := schema.GroupVersionResource{
			Resource: "shoots",
		}

		// when
		err := tracker.Delete(gvr, "test", "shoot2")

		// then
		require.Error(t, err)
	})
}
