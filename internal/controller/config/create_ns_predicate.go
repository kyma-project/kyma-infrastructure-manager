package config

import (
	"log/slog"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

var _ predicate.TypedPredicate[client.Object] = &createResourcePredicate{}

type createResourcePredicate struct {
	types.NamespacedName
}

func (p createResourcePredicate) slogArgs() []any {
	return []any{
		"name", p.Name,
		"namespace", p.Namespace,
	}
}

func (p createResourcePredicate) match(e event.TypedUpdateEvent[client.Object]) bool {
	return p.Name == e.ObjectNew.GetName() && p.Namespace == e.ObjectOld.GetNamespace()
}

// Create - handles the case of namespace creation (omits events comming from
// the master secret namespace)
func (p createResourcePredicate) Create(e event.TypedCreateEvent[client.Object]) bool {
	return false
}

// Delete - omit event
func (p createResourcePredicate) Delete(event.TypedDeleteEvent[client.Object]) bool {
	return false
}

// Update - omit event
func (p createResourcePredicate) Update(e event.TypedUpdateEvent[client.Object]) bool {
	if !p.match(e) {
		return false
	}

	args := p.slogArgs()
	slog.Debug("resource updated", args...)

	return true
}

// Generic - omit event
func (p createResourcePredicate) Generic(event.TypedGenericEvent[client.Object]) bool {
	return false
}
