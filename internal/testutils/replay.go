package testutils

import (
	"context"
	"encoding/json"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	auditv1 "k8s.io/apiserver/pkg/apis/audit"

	"github.com/tigera/compliance/pkg/event"
	"github.com/tigera/compliance/pkg/list"
	"github.com/tigera/compliance/pkg/resources"
)

type ReplayTester struct {
	lists  map[metav1.TypeMeta]*list.TimestampedResourceList
	events map[metav1.TypeMeta][]*auditv1.Event
}

func (t *ReplayTester) RetrieveList(kind metav1.TypeMeta, from, to *time.Time, sortAscendingTime bool) (*list.TimestampedResourceList, error) {
	l, ok := t.lists[kind]
	if !ok {
		panic(kind.String() + " list does not exist")
	}
	return l, nil
}

func (t *ReplayTester) StoreList(tm metav1.TypeMeta, l *list.TimestampedResourceList) error {
	t.lists[tm] = l
	return nil
}

func (t *ReplayTester) GetAuditEvents(ctx context.Context, tm *metav1.TypeMeta, from, to *time.Time) <-chan *event.AuditEventResult {
	ch := make(chan *event.AuditEventResult)
	go func() {
		defer close(ch)

		// All type metas.
		if tm == nil {
			for _, events := range t.events {
				for _, ev := range events {
					if ev.StageTimestamp.Time.After(*from) && ev.StageTimestamp.Time.Before(*to) {
						ch <- &event.AuditEventResult{ev, nil}
					}
				}
			}
			return
		}

		// Single type meta.
		for _, ev := range t.events[*tm] {
			if ev.StageTimestamp.Time.After(*from) && ev.StageTimestamp.Time.Before(*to) {
				ch <- &event.AuditEventResult{ev, nil}
			}
		}
	}()
	return ch
}

func (t *ReplayTester) SetResourceAuditEvent(verb string, res resources.Resource, timestamp time.Time) {
	ev := new(auditv1.Event)
	ev.Verb = verb

	// Set the objectRef
	ev.ObjectRef = &auditv1.ObjectReference{}

	// Set the response object.
	resJson, err := json.Marshal(res)
	ev.ResponseObject = &runtime.Unknown{Raw: resJson}
	if err != nil {
		panic(err)
	}

	// Set the timestamp.
	ev.StageTimestamp = metav1.MicroTime{timestamp}

	// Append to event array.
	tm := resources.GetTypeMeta(res)
	t.events[tm] = append(t.events[tm], ev)
}

func NewReplayTester(timestamp time.Time) *ReplayTester {
	lists := make(map[metav1.TypeMeta]*list.TimestampedResourceList)
	for _, rh := range resources.GetAllResourceHelpers() {
		lists[rh.TypeMeta()] = &list.TimestampedResourceList{rh.NewResourceList(), metav1.Time{timestamp}, metav1.Time{timestamp}}
	}
	return &ReplayTester{lists: lists, events: make(map[metav1.TypeMeta][]*auditv1.Event)}
}
