package util

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type PredicateEvent string

const (
	PredicateEventCreate  PredicateEvent = "Create"
	PredicateEventUpdate  PredicateEvent = "Update"
	PredicateEventDelete  PredicateEvent = "Delete"
	PredicateEventGeneric PredicateEvent = "Generic"
)

func PreficateOfFunction(filter func(client.Object, PredicateEvent) bool) predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return filter(e.Object, PredicateEventCreate)
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return filter(e.ObjectNew, PredicateEventUpdate)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return filter(e.Object, PredicateEventDelete)
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return filter(e.Object, PredicateEventGeneric)
		},
	}
}
