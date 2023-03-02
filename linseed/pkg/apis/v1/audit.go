// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package v1

import (
	"fmt"

	authnv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apiserver/pkg/apis/audit"

	"github.com/projectcalico/calico/libcalico-go/lib/json"
)

type Kind string

const (
	KindNetworkPolicy       Kind = "networkpolicies"
	KindGlobalNetworkPolicy Kind = "globalnetworkpolicies"
	KindNetworkSet          Kind = "networksets"
	KindGlobalNetworkSet    Kind = "globalnetworksets"
)

type Verb string

const (
	Create Verb = "create"
	Delete Verb = "delete"
	Patch  Verb = "patch"
	Update Verb = "update"
)

// AuditLogParams provide query options for listing audit logs.
type AuditLogParams struct {
	QueryParams `json:",inline" validate:"required"`
	Type        AuditLogType `json:"type"`

	// Configure filtering based on object kind. Response will include
	// any objects that match any of the given kinds.
	Kinds []Kind `json:"kinds"`

	// Match specific object fields.
	ObjectRef *ObjectReference `json:"object_ref"`

	// Match the action taken on the resource
	Verbs []Verb `json:"verbs"`

	// Response code match.
	ResponseCodes []int32 `json:"response_codes"`
}

// ObjectReference is the set of fields we support in query requests
// to filter audit logs based on their object.
type ObjectReference struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type AuditLogType string

const (
	AuditLogTypeKube AuditLogType = "kube"
	AuditLogTypeEE   AuditLogType = "ee"
)

// AuditLog is the model used to ingest audit data
type AuditLog struct {
	// Composition of K8S Audit Event
	audit.Event
	// Name field is populated by FluentD
	Name *string `json:"name,omitempty"`
}

// internalEvent is a copy of the K8S Audit Event in order to perform JSON marshaling/unmarshalling
// to a representation that allows the addition of new fields and camel case for json keys
type internalEvent struct {
	// K8S Audit fields with proper json tags in order to represent
	// the json fields with camel case instead of capital letters
	metav1.TypeMeta
	Level                    audit.Level              `json:"level"`
	AuditID                  types.UID                `json:"auditID"`
	Stage                    audit.Stage              `json:"stage"`
	RequestURI               string                   `json:"requestURI"`
	Verb                     string                   `json:"verb"`
	User                     authnv1.UserInfo         `json:"user"`
	ImpersonatedUser         *authnv1.UserInfo        `json:"impersonatedUser,omitempty"`
	SourceIPs                []string                 `json:"sourceIPs"`
	UserAgent                string                   `json:"userAgent"`
	ObjectRef                *internalObjectReference `json:"objectRef,omitempty"`
	ResponseStatus           *metav1.Status           `json:"responseStatus,omitempty"`
	RequestObject            *runtime.Unknown         `json:"requestObject,omitempty"`
	ResponseObject           *runtime.Unknown         `json:"responseObject,omitempty"`
	RequestReceivedTimestamp metav1.MicroTime         `json:"requestReceivedTimestamp"`
	StageTimestamp           metav1.MicroTime         `json:"stageTimestamp"`
	Annotations              map[string]string        `json:"annotations"`

	// Additional field added by FluentD
	Name *string `json:"name,,omitempty"`
}

type internalObjectReference struct {
	Resource        string    `json:"resource,omitempty"`
	Namespace       string    `json:"namespace,omitempty"`
	Name            string    `json:"name,omitempty"`
	UID             types.UID `json:"uid,omitempty"`
	APIGroup        string    `json:"apiGroup,omitempty"`
	APIVersion      string    `json:"apiVersion,omitempty"`
	ResourceVersion string    `json:"resourceVersion,omitempty"`
	Subresource     string    `json:"subresource,omitempty"`
}

func (auditLog *AuditLog) MarshalJSON() ([]byte, error) {
	if auditLog == nil {
		return []byte{}, fmt.Errorf("cannot marshal nil value into JSON")
	}

	// Create an internal representation of the K8S event
	// We are doing this because K8S Audit event is currently
	// stored in ES with camel case for fields instead of
	// using capital case for fields missing json tags
	val := internalEvent{}
	val.TypeMeta = auditLog.TypeMeta
	val.AuditID = auditLog.AuditID
	val.Level = auditLog.Level
	val.Stage = auditLog.Stage
	val.RequestURI = auditLog.RequestURI
	val.Verb = auditLog.Verb
	val.User = auditLog.User
	if auditLog.ImpersonatedUser != nil {
		val.ImpersonatedUser = auditLog.ImpersonatedUser.DeepCopy()
	}
	val.SourceIPs = auditLog.SourceIPs
	val.UserAgent = auditLog.UserAgent
	val.RequestReceivedTimestamp = auditLog.RequestReceivedTimestamp
	if auditLog.ResponseStatus != nil {
		val.ResponseStatus = auditLog.ResponseStatus.DeepCopy()
	}
	val.RequestReceivedTimestamp = auditLog.RequestReceivedTimestamp
	val.StageTimestamp = auditLog.StageTimestamp
	val.Annotations = auditLog.Annotations

	// Copy ObjectReference in an internal representation
	// in order to JSON marshal with camelCase instead of
	// using capital letters
	if auditLog.ObjectRef != nil {
		val.ObjectRef = &internalObjectReference{
			Resource:        auditLog.ObjectRef.Resource,
			Namespace:       auditLog.ObjectRef.Namespace,
			Name:            auditLog.ObjectRef.Name,
			UID:             auditLog.ObjectRef.UID,
			APIGroup:        auditLog.ObjectRef.APIGroup,
			APIVersion:      auditLog.ObjectRef.APIVersion,
			ResourceVersion: auditLog.ObjectRef.ResourceVersion,
			Subresource:     auditLog.ObjectRef.Subresource,
		}
	}
	if auditLog.RequestObject != nil {
		val.RequestObject = auditLog.RequestObject.DeepCopy()
	}
	if auditLog.ResponseObject != nil {
		val.ResponseObject = auditLog.ResponseObject.DeepCopy()
	}

	if auditLog.Name != nil {
		val.Name = auditLog.Name
	}

	return json.Marshal(val)
}

func (auditLog *AuditLog) UnmarshalJSON(data []byte) error {
	if auditLog == nil {
		return fmt.Errorf("cannot unmarshal nil value from JSON")
	}

	// Unmarshal the data received from FluentD
	// as an internal representation of K8S Audit
	// since FluentD add additional fields like "name"
	var internalEvent internalEvent
	err := json.Unmarshal(data, &internalEvent)
	if err != nil {
		return err
	}

	// Populate the name field that is filled in by FluentD
	// and is not available in the K8S model
	if internalEvent.Name != nil {
		auditLog.Name = internalEvent.Name
	}

	// Create an empty KS8 audit k8sEvent and
	// populate the K8S k8sEvent model with all the necessary fields
	k8sEvent := audit.Event{}
	k8sEvent.TypeMeta = internalEvent.TypeMeta
	k8sEvent.AuditID = internalEvent.AuditID
	k8sEvent.Level = internalEvent.Level
	k8sEvent.Stage = internalEvent.Stage
	k8sEvent.RequestURI = internalEvent.RequestURI
	k8sEvent.Verb = internalEvent.Verb
	k8sEvent.User = internalEvent.User
	if internalEvent.ImpersonatedUser != nil {
		k8sEvent.ImpersonatedUser = internalEvent.ImpersonatedUser.DeepCopy()
	}
	k8sEvent.SourceIPs = internalEvent.SourceIPs
	k8sEvent.UserAgent = internalEvent.UserAgent
	k8sEvent.RequestReceivedTimestamp = internalEvent.RequestReceivedTimestamp
	k8sEvent.RequestReceivedTimestamp = internalEvent.RequestReceivedTimestamp
	k8sEvent.StageTimestamp = internalEvent.StageTimestamp
	k8sEvent.Annotations = internalEvent.Annotations
	if internalEvent.ResponseStatus != nil {
		k8sEvent.ResponseStatus = internalEvent.ResponseStatus.DeepCopy()
	}

	// Populating ObjectRef
	if internalEvent.ObjectRef != nil {
		// Copy ObjectReference from the internal representation
		// to have access to the K8S Audit ObjectReference
		k8sEvent.ObjectRef = &audit.ObjectReference{
			Resource:        internalEvent.ObjectRef.Resource,
			Namespace:       internalEvent.ObjectRef.Namespace,
			Name:            internalEvent.ObjectRef.Name,
			UID:             internalEvent.ObjectRef.UID,
			APIGroup:        internalEvent.ObjectRef.APIGroup,
			APIVersion:      internalEvent.ObjectRef.APIVersion,
			ResourceVersion: internalEvent.ObjectRef.ResourceVersion,
			Subresource:     internalEvent.ObjectRef.Subresource,
		}
	}

	// Populating RequestObject
	if internalEvent.RequestObject != nil {
		// Copy Object reference on K8S Audit Event
		k8sEvent.RequestObject = internalEvent.RequestObject.DeepCopy()
	}

	// Populating ResponseObject
	if internalEvent.ResponseObject != nil {
		// Copy Object reference on K8S Audit Event
		k8sEvent.ResponseObject = internalEvent.ResponseObject.DeepCopy()
	}

	// Replace the K8S k8sEvent on the AuditLog
	auditLog.Event = k8sEvent

	return nil
}
