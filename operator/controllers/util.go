package controllers

import (
	inlocov1alpha1 "github.com/inloco/kube-actions/operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	runnerResourcesKey = "runner"
)

func HasActionsRunnerRequestedStorage(actionsRunner *inlocov1alpha1.ActionsRunner) bool {
	if actionsRunner == nil {
		return false
	}

	res, ok := actionsRunner.Spec.Resources[runnerResourcesKey]
	if !ok {
		return false
	}

	req := res.Requests
	if req == nil {
		return false
	}

	s := req.Storage()
	if s == nil {
		return false
	}

	return !s.IsZero()
}

type Event interface{}

func EventObject(e Event) client.Object {
	if createEvent, ok := e.(event.CreateEvent); ok {
		return createEvent.Object
	}

	if updateEvent, ok := e.(event.UpdateEvent); ok {
		return updateEvent.ObjectNew
	}

	if deleteEvent, ok := e.(event.DeleteEvent); ok {
		return deleteEvent.Object
	}

	if genericEvent, ok := e.(event.GenericEvent); ok {
		return genericEvent.Object
	}

	return nil
}

type EventPredicate func(Event) bool

var _ predicate.Predicate = EventPredicate(nil)

func (ep EventPredicate) Create(createEvent event.CreateEvent) bool {
	return ep(createEvent)
}

func (ep EventPredicate) Update(updateEvent event.UpdateEvent) bool {
	return ep(updateEvent)
}

func (ep EventPredicate) Delete(deleteEvent event.DeleteEvent) bool {
	return ep(deleteEvent)
}

func (ep EventPredicate) Generic(genericEvent event.GenericEvent) bool {
	return ep(genericEvent)
}
