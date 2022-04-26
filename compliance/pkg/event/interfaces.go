package event

import (
	"encoding/json"
	"errors"
	"fmt"

	log "github.com/sirupsen/logrus"
	auditv1 "k8s.io/apiserver/pkg/apis/audit"

	"github.com/projectcalico/calico/libcalico-go/lib/resources"

	api "github.com/tigera/lma/pkg/api"
)

// ExtractResourceFromAuditEvent determines the resource kind located within an audit event
// and coerces the response object into the appropriate type. This may return a nil resource with
// no error if the resource is not handled by the reporter code.
func ExtractResourceFromAuditEvent(event *auditv1.Event) (resources.Resource, error) {
	// We only need to extract successful configuration updates.
	if event.ResponseStatus != nil && (event.ResponseStatus.Code < 200 || event.ResponseStatus.Code > 299) {
		log.Debugf("Skipping unsuccessful event, code: %d", event.ResponseStatus.Code)
		return nil, nil
	}

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
	case api.EventVerbCreate, api.EventVerbUpdate, api.EventVerbPatch, api.EventVerbDelete:
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

	if event.Verb == api.EventVerbDelete {
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
		// The response object is missing from the audit log. If this is a subresource then we can ignore this as
		// the subresource contains information that is not required for the compliance reports to be generated,
		// although it may under some circumstances provide more accurate results, so if we have it we'll use it.
		if event.ObjectRef.Subresource != "" {
			// We can skip sub-resources since we don't generally collect the audit logs for these.
			clog.Debugf("Skipping event for subresource %s with no ResponseObject", event.ObjectRef.Subresource)
			return nil, nil
		}
		logEventError(event, "responseObject is missing from audit log - audit policy must be incorrect")
		return nil, errors.New("responseObject is missing from audit log - audit policy must be incorrect")
	}

	if err := json.Unmarshal(event.ResponseObject.Raw, res); err != nil {
		logEventError(event, fmt.Sprintf("Failed to unmarshal responseObject: %v", err))
		return nil, err
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
