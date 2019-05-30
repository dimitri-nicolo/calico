package event

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	auditv1 "k8s.io/apiserver/pkg/apis/audit"

	"github.com/tigera/compliance/pkg/resources"
)

const (
	VerbCreate = "create"
	VerbUpdate = "update"
	VerbPatch  = "patch"
	VerbDelete = "delete"
)

var (
	ConfigurationVerbs = []string{
		VerbCreate, VerbUpdate, VerbPatch, VerbDelete,
	}
)

type AuditEventResult struct {
	*auditv1.Event
	Err error
}

type Fetcher interface {
	GetAuditEvents(context.Context, *time.Time, *time.Time) <-chan *AuditEventResult
}

// ExtractResourceFromAuditEvent determines the resource kind located within an audit event
// and coerces the response object into the appropriate type. This may return a nil resource with
// no error if the resource is not handled by the reporter code.
func ExtractResourceFromAuditEvent(event *auditv1.Event) (resources.Resource, error) {
	// Check this is a ResponseComplete stage.
	switch event.Stage {
	case auditv1.StageResponseComplete:
		log.Debug("Stage is ResponseComplete - continue processing")
	default:
		log.Debugf("Event stage is %s - skipping", event.Stage)
		return nil, nil
	}

	// Check that the event is configuration event.
	switch event.Verb {
	case VerbCreate, VerbUpdate, VerbPatch, VerbDelete:
		log.Debug("Event is a configuration event - continue processing")
	default:
		log.Debugf("Event verb is %s - skipping", event.Verb)
		return nil, nil
	}

	// Extract the Object reference and use that to instantiate a new instance of the resource. We always expect
	// this to be set for a configuration event.
	if event.ObjectRef == nil {
		logEventError(event, "No objectRef specified in audit log")
		return nil, errors.New("no objectRef specified in audit log")
	} else if event.ObjectRef.Resource == "" {
		logEventError(event, "No objectRef.Resource specified in audit log")
		return nil, errors.New("no objectRef.Resource specified in audit log")
	}

	// Set up a context logger.
	clog := log.WithField("resource", event.ObjectRef.Resource)

	// We only have resource helpers for events that we need to explicitly process. However, the audit and replay
	// processing may read audit logs that are for different resource types - therefore it's perfectly reasonable
	// to not have an associated resource helper, just skip.
	rh := resources.GetResourceHelperByObjectRef(*event.ObjectRef)
	if rh == nil {
		clog.Debug("Object type is not required for report processing - skipping")
		return nil, nil
	}

	// Create a new resource to unmarshal the event into.
	res := rh.NewResource()

	if event.Verb == VerbDelete {
		// This is a delete event, the response object will not be extractable so just return what we can.
		//
		// Sanity check that we have a name specified. It must be specified in the ObjectRef for a delete event although
		// the same cannot be said for create events where the name may not necessarily be specified.
		if event.ObjectRef.Name == "" {
			logEventError(event, "No objectRef.Name specified for delete event")
			return nil, errors.New("no objectReference.Name specified in audit log")
		}
		res.GetObjectMeta().SetNamespace(event.ObjectRef.Namespace)
		res.GetObjectMeta().SetName(event.ObjectRef.Name)
		return res, nil
	}

	if event.ResponseObject == nil {
		logEventError(event, "responseObject is missing from audit log - audit policy must be incorrect")
		return nil, errors.New("responseObject is missing from audit log - audit policy must be incorrect")
	}

	if err := json.Unmarshal(event.ResponseObject.Raw, res); err != nil {
		logEventError(event, fmt.Sprintf("Failed to unmarshal responseObject: %v", err))
		return nil, err
	}

	// Ensure that we haven't received a status audit log or some other invalid type.
	if tm := resources.GetTypeMeta(res); tm == resources.TypeK8sStatus {
		clog.Debug("Skipping status audit event")
		return nil, nil
	}

	return res, nil
}

// logEventError logs the audit event with an error message, or just the AuditID if marshaling fails for some
// reason.
func logEventError(event *auditv1.Event, txt string) {
	if b, err := json.Marshal(event); err == nil {
		log.WithField("Event", string(b)).Error(txt)
	} else {
		log.WithField("AuditID", event.AuditID).Error(txt)
	}
}
