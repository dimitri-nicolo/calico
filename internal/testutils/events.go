package testutils

import (
	"encoding/json"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	auditv1 "k8s.io/apiserver/pkg/apis/audit"

	"github.com/tigera/compliance/pkg/resources"
)

func NewAuditEvent(verb string, res resources.Resource, timestamp time.Time) *auditv1.Event {
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

	return ev
}
