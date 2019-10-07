// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package query

import (
	"fmt"
)

var (
	// auditKeys omits disabled properties in the mappings
	auditKeys = map[string]Validator{
		"apiVersion":                URLValidator,
		"auditID":                   NullValidator,
		"kind":                      NullValidator,
		"level":                     SetValidator("None", "Metadata", "Request", "RequestResponse"),
		"name":                      DomainValidator,
		"objectRef.apiGroup":        NullValidator,
		"objectRef.apiVersion":      URLValidator,
		"objectRef.name":            DomainValidator,
		"objectRef.resource":        NullValidator,
		"objectRef.namespace":       DomainValidator,
		"requestReceivedTimestamp":  DateValidator,
		"requestURI":                URLValidator,
		"responseObject.apiVersion": NullValidator,
		"responseObject.kind":       NullValidator,
		"responseStatus.code":       IntRangeValidator(100, 599),
		"sourceIPs":                 IPValidator,
		"stage":                     SetValidator("RequestReceived", "ResponseStarted", "ResponseComplete", "Panic"),
		"stageTimestamp":            DateValidator,
		"timestamp":                 DateValidator,
		"user.groups":               NullValidator,
		"user.username":             NullValidator,
		"verb":                      SetValidator("get", "list", "watch", "create", "update", "patch", "delete"),
	}
)

func IsValidAuditAtom(a *Atom) error {
	if validator, ok := auditKeys[a.Key]; ok {
		return validator(a)
	}

	return fmt.Errorf("invalid key: %s", a.Key)
}
