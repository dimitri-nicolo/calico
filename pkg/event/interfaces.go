package event

import (
	"context"
	"encoding/json"
	"time"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	auditv1 "k8s.io/apiserver/pkg/apis/audit/v1beta1"

	"github.com/tigera/compliance/pkg/resources"
)

type AuditEventResult struct {
	*auditv1.Event
	Err error
}

type Fetcher interface {
	GetAuditEvents(context.Context, *schema.GroupVersionKind, *time.Time, *time.Time) <-chan *AuditEventResult
}

func ExtractResourceFromAuditEvent(event *auditv1.Event) (resources.Resource, error) {
	clog := log.WithField("kind", event.ObjectRef.Resource)
	// Extract group version kind from event response object.
	tm := new(metav1.TypeMeta)
	if err := json.Unmarshal(event.ResponseObject.Raw, tm); err != nil {
		clog.WithError(err).WithField("json", string(event.ResponseObject.Raw)).Error("failed to marshal json")
		return nil, err
	}

	// Extract resource from event response object.
	clog = log.WithField("type", tm)
	if tm.Kind == "Status" {
		return nil, nil
	}
	rh := resources.GetResourceHelper(tm.GroupVersionKind())
	res := rh.NewResource()
	if err := json.Unmarshal(event.ResponseObject.Raw, res); err != nil {
		clog.WithError(err).WithField("json", string(event.ResponseObject.Raw)).Error("failed to marshal json")
		return nil, err
	}

	return res, nil
}
