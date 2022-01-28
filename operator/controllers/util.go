package controllers

import (
	"reflect"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	inlocov1alpha1 "github.com/inloco/kube-actions/operator/api/v1alpha1"
)

var (
	CreateOpts = []client.CreateOption{
		client.FieldOwner("kube-actions"),
	}

	PatchOpts = []client.PatchOption{
		client.ForceOwnership,
		client.FieldOwner("kube-actions"),
	}

	UpdateOpts = []client.UpdateOption{
		client.FieldOwner("kube-actions"),
	}

	DeleteOpts = []client.DeleteOption{
		client.PropagationPolicy(metav1.DeletePropagationForeground),
	}
)

func HasActionsRunnerRequestedStorage(actionsRunner *inlocov1alpha1.ActionsRunner) bool {
	if actionsRunner == nil {
		return false
	}

	res, ok := actionsRunner.Spec.Resources["runner"]
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

type ReconciliationAction byte

const (
	ReconciliationActionNothing = iota
	ReconciliationActionCreate
	ReconciliationActionUpdate
	ReconciliationActionDelete
)

func CalculateReconciliationAction(actual client.Object, desired client.Object) ReconciliationAction {
	actualIsZero := IsZero(actual)
	desiredIsZero := IsZero(desired)

	if actualIsZero && !desiredIsZero {
		return ReconciliationActionCreate
	}

	if !actualIsZero && !desiredIsZero {
		return ReconciliationActionUpdate
	}

	if !actualIsZero && desiredIsZero {
		return ReconciliationActionDelete
	}

	return ReconciliationActionNothing
}

func IsZero(i interface{}) bool {
	v := reflect.ValueOf(i)
	if v.Kind() == reflect.Invalid {
		return true
	}

	v = reflect.Indirect(v)
	if v.Kind() == reflect.Invalid {
		return true
	}

	return v.IsZero()
}

func IgnoreAlreadyExists(err error) error {
	if apierrors.IsAlreadyExists(err) {
		return nil
	}

	return err
}
