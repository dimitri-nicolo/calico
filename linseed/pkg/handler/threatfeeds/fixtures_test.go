// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package threatfeeds

import (
	"time"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
)

const dummyURL = "anyURL"

var (
	noIpSet     []v1.IPSetThreatFeed
	noDomainSet []v1.DomainNameSetThreatFeed
	ipSets      = []v1.IPSetThreatFeed{
		{
			ID: "a",
			Data: &v1.IPSetThreatFeedData{
				CreatedAt: time.Unix(0, 0),
				IPs:       []string{"1.2.3.4/32"},
			},
		},
		{
			ID: "b",
			Data: &v1.IPSetThreatFeedData{
				CreatedAt: time.Unix(0, 0),
				IPs:       []string{"1.2.3.5/32"},
			},
		},
	}

	domainSets = []v1.DomainNameSetThreatFeed{
		{
			ID: "a",
			Data: &v1.DomainNameSetThreatFeedData{
				CreatedAt: time.Unix(0, 0),
				Domains:   []string{"a.b.c.d"},
			},
		},
		{
			ID: "b",
			Data: &v1.DomainNameSetThreatFeedData{
				CreatedAt: time.Unix(0, 0),
				Domains:   []string{"x.y.z"},
			},
		},
	}

	bulkResponseSuccess = &v1.BulkResponse{
		Total:     2,
		Succeeded: 2,
		Failed:    0,
	}

	bulkResponsePartialSuccess = &v1.BulkResponse{
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
)
