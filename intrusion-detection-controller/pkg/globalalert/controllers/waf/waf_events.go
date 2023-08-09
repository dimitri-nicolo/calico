// Copyright 2023 Tigera Inc. All rights reserved.

package waf

import (
	"github.com/projectcalico/calico/libcalico-go/lib/validator/v3/query"
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
)

func NewWafEvent(l v1.WAFLog) v1.Event {

	return v1.Event{
		Type: query.WafEventType,
		// Deviating from original implementation here
		Origin: "Web Application Firewall",

		// GlobalAlert use time.Now() but it makes more sense to use the
		// timestamp from the WAF log...
		Time:        v1.NewEventDate(l.Timestamp),
		Name:        "WAF Event",
		Description: "Some traffic inside your cluster triggered some Web Application Firewall rules",
		// Bad but not too bad :) Open for feedback
		Severity:     80,
		Host:         l.Host,
		Protocol:     l.Protocol,
		SourceIP:     &l.Source.IP,
		SourceName:   l.Source.Hostname,
		DestIP:       &l.Destination.IP,
		DestName:     l.Destination.Hostname,
		MitreIDs:     &[]string{"T1190"},
		Mitigations:  &[]string{"Review the source of this event - an attacker could be inside your cluster attempting to exploit your web application. Calico network policy can be used to block the connection if the activity is not expected"},
		AttackVector: "Network",
		MitreTactic:  "Initial Access",

		Record: l,
	}
}
