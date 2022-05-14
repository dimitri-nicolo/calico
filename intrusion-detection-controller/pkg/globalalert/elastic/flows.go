// Copyright (c) 2019 Tigera Inc. All rights reserved.

package elastic

import (
	"strings"

	"github.com/projectcalico/calico/libcalico-go/lib/validator/v3/query"
)

func NewFlowsConverter() ElasticQueryConverter {
	return &converter{flowsAtomToElastic}
}

func flowsAtomToElastic(a *query.Atom) JsonObject {
	switch a.Key {
	case "dest_labels.labels", "policies.all_policies", "source_labels.labels":
		path := a.Key[:strings.Index(a.Key, ".")]
		return JsonObject{
			"nested": JsonObject{
				"path":  path,
				"query": basicAtomToElastic(a),
			},
		}
	default:
		return basicAtomToElastic(a)
	}
}
