package rtbootstrapperconfig

import (
	"log/slog"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

var _ predicate.TypedPredicate[client.Object] = &objectUpdatedPredicate{}

type objectUpdatedPredicate struct {
	types.NamespacedName
}

func (p objectUpdatedPredicate) slogArgs() []any {
	return []any{
		"name", p.Name,
		"namespace", p.Namespace,
	}
}

func (p objectUpdatedPredicate) match(e event.TypedUpdateEvent[client.Object]) bool {
	return p.Name == e.ObjectNew.GetName() && p.Namespace == e.ObjectNew.GetNamespace()
}

// Create - handles the case of namespace creation (omits events comming from
// the master secret namespace)
func (p objectUpdatedPredicate) Create(e event.TypedCreateEvent[client.Object]) bool {
	return false
}

// Delete - omit event
func (p objectUpdatedPredicate) Delete(event.TypedDeleteEvent[client.Object]) bool {
	return false
}

// Update - omit event
func (p objectUpdatedPredicate) Update(e event.TypedUpdateEvent[client.Object]) bool {
	if !p.match(e) {
		return false
	}

	args := p.slogArgs()
	slog.Debug("resource updated", args...)
	slog.Info("configuration resource modified ", args...)

	return true
}

// Generic - omit event
func (p objectUpdatedPredicate) Generic(event.TypedGenericEvent[client.Object]) bool {
	return false
}
