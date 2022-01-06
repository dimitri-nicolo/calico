// Copyright (c) 2019 Tigera Inc. All rights reserved.

package elastic

import (
	"strings"

	"github.com/projectcalico/calico/libcalico-go/lib/validator/v3/query"
)

func NewDNSConverter() ElasticQueryConverter {
	return &converter{dnsAtomToElastic}
}

func dnsAtomToElastic(a *query.Atom) JsonObject {
	switch a.Key {
	case "servers.name", "servers.name_aggr", "servers.namespace", "servers.ip",
		"rrsets.name", "rrsets.type", "rrsets.class", "rrsets.rdata":

		path := a.Key[:strings.Index(a.Key, ".")]
		return JsonObject{
			"nested": JsonObject{
				"path":  path,
				"query": basicAtomToElastic(a),
			},
		}
	}

	switch {
	case strings.HasPrefix(a.Key, "servers.labels."):
		return JsonObject{
			"nested": JsonObject{
				"path":  "servers",
				"query": basicAtomToElastic(a),
			},
		}
	case strings.HasPrefix(a.Key, "client_labels."):
		return basicAtomToElastic(a)
	default:
		return basicAtomToElastic(a)
	}
}
