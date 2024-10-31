// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package audit

import (
	"time"

	authnv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kaudit "k8s.io/apiserver/pkg/apis/audit"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/testutils"
)

const dummyURL = "anyURL"

var event = kaudit.Event{
	TypeMeta:   metav1.TypeMeta{Kind: "Event", APIVersion: "audit.k8s.io/v1"},
	AuditID:    types.UID("some-uuid-most-likely"),
	Stage:      kaudit.StageRequestReceived,
	RequestURI: "/apis/v3/projectcalico.org",
	Verb:       "PUT",
	User: authnv1.UserInfo{
		Username: "user",
		UID:      "uid",
		Extra:    map[string]authnv1.ExtraValue{"extra": authnv1.ExtraValue([]string{"value"})},
	},
	ImpersonatedUser: &authnv1.UserInfo{
		Username: "impuser",
		UID:      "impuid",
		Groups:   []string{"g1"},
	},
	SourceIPs:      []string{"1.2.3.4"},
	UserAgent:      "user-agent",
	ObjectRef:      &kaudit.ObjectReference{},
	ResponseStatus: &metav1.Status{},
	RequestObject: &runtime.Unknown{
		ContentType: runtime.ContentTypeJSON,
	},
	ResponseObject: &runtime.Unknown{
		ContentType: runtime.ContentTypeJSON,
	},
	RequestReceivedTimestamp: metav1.NewMicroTime(time.Now().Add(-5 * time.Second)),
	StageTimestamp:           metav1.NewMicroTime(time.Now()),
	Annotations:              map[string]string{"brick": "red"},
}

var (
	noAuditLogs   []v1.AuditLog
	kubeAuditLogs = []v1.AuditLog{
		{
			Event: event,
			Name:  testutils.StringPtr("ee-any"),
		},
		{Event: event},
	}
	eeAuditLogs = []v1.AuditLog{
		{Event: event},
		{Event: event},
	}
)

var bulkResponseSuccess = &v1.BulkResponse{
	Total:     2,
	Succeeded: 2,
	Failed:    0,
}

var bulkResponsePartialSuccess = &v1.BulkResponse{
	Total:     2,
	Succeeded: 1,
	Failed:    1,
	Errors: []v1.BulkError{
		{
			Resource: "res",
			Type:     "index error",
			Reason:   "I couldn't do it",
		},
	},
}
