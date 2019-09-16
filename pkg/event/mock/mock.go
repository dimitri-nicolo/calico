package mock

import (
	"context"
	"encoding/json"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	auditv1 "k8s.io/apiserver/pkg/apis/audit"

	"github.com/projectcalico/libcalico-go/lib/resources"
	api "github.com/tigera/lma/pkg/api"
)

type Fetcher struct {
	data map[metav1.TypeMeta][]*api.AuditEventResult
}

func NewEventFetcher() *Fetcher {
	return &Fetcher{data: make(map[metav1.TypeMeta][]*api.AuditEventResult)}
}

func (f *Fetcher) GetAuditEvents(ctx context.Context, from, to *time.Time) <-chan *api.AuditEventResult {
	ch := make(chan *api.AuditEventResult)
	go func() {
		defer close(ch)

		for _, events := range f.data {
			for _, ev := range events {
				if ev.StageTimestamp.Time.After(*from) && ev.StageTimestamp.Time.Before(*to) {
					ch <- ev
				}
			}
		}
	}()
	return ch
}

func (f *Fetcher) LoadAuditEvent(verb string, stage auditv1.Stage, objRef resources.Resource, respObj interface{}, timestamp time.Time, resVer string) {
	// Get the resource helper.
	tm := resources.GetTypeMeta(objRef)
	rh := resources.GetResourceHelperByTypeMeta(tm)

	// Create the audit event.
	ev := &auditv1.Event{
		Verb:  verb,
		Stage: stage,
		ObjectRef: &auditv1.ObjectReference{
			Name:       objRef.GetObjectMeta().GetName(),
			Namespace:  objRef.GetObjectMeta().GetNamespace(),
			APIGroup:   objRef.GetObjectKind().GroupVersionKind().Group,
			APIVersion: objRef.GetObjectKind().GroupVersionKind().Version,
			Resource:   rh.Plural(),
		},
		StageTimestamp: metav1.MicroTime{timestamp},
	}

	// Set the response object if this is a response complete stage event.
	if stage == auditv1.StageResponseComplete {
		if obj, ok := respObj.(resources.Resource); ok {
			obj.GetObjectMeta().SetResourceVersion(resVer)
		}
		resJson, err := json.Marshal(respObj)
		ev.ResponseObject = &runtime.Unknown{Raw: resJson}
		if err != nil {
			panic(err)
		}
	}

	// Append to event array
	f.data[tm] = append(f.data[tm], &api.AuditEventResult{ev, nil})
}
