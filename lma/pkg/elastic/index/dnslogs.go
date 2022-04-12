// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package index

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/olivere/elastic/v7"

	"github.com/projectcalico/calico/libcalico-go/lib/validator/v3/query"

	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"github.com/projectcalico/calico/lma/pkg/httputils"
)

const esDnsLogsIndexPrefix = "tigera_secure_ee_dns"

// dnsLogsIndexHelper implements the Helper interface for dns logs.
type dnsLogsIndexHelper struct{}

// DnsLogs returns an instance of the dns logs index helper.
func DnsLogs() Helper {
	return dnsLogsIndexHelper{}
}

// NewDnsLogsConverter returns a converter instance defined for dns logs.
func NewDnsLogsConverter() converter {
	return converter{dnsAtomToElastic}
}

// dnsAtomToElastic returns a dns log atom as an elastic JsonObject.
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

// Helper.

func (h dnsLogsIndexHelper) NewSelectorQuery(selector string) (elastic.Query, error) {
	q, err := query.ParseQuery(selector)
	if err != nil {
		return nil, &httputils.HttpStatusError{
			Status: http.StatusBadRequest,
			Err:    err,
			Msg:    fmt.Sprintf("Invalid selector (%s) in request: %v", selector, err),
		}
	} else if err := query.Validate(q, query.IsValidDNSAtom); err != nil {
		return nil, &httputils.HttpStatusError{
			Status: http.StatusBadRequest,
			Err:    err,
			Msg:    fmt.Sprintf("Invalid selector (%s) in request: %v", selector, err),
		}
	}
	converter := NewDnsLogsConverter()
	return JsonObjectElasticQuery(converter.Convert(q)), nil
}

func (h dnsLogsIndexHelper) NewRBACQuery(
	resources []apiv3.AuthorizedResourceVerbs,
) (elastic.Query, error) {
	// Convert the permissions into a query that each flow must satisfy - essentially a source or
	// destination must be listable by the user to be included in the response.
	var should []elastic.Query
	for _, r := range resources {
		for _, v := range r.Verbs {
			if v.Verb != "list" {
				// Only interested in the list verbs.
				continue
			}
			for _, rg := range v.ResourceGroups {
				switch r.Resource {
				case "pods":
					if rg.Namespace == "" {
						// User can list all namespaces.
						return nil, nil
					}
					// Can list Pods in a specific namespace.
					should = append(should,
						elastic.NewTermQuery("client_namespace", rg.Namespace),
					)
				}
			}
			break
		}
	}

	if len(should) == 0 {
		return nil, &httputils.HttpStatusError{
			Status: http.StatusForbidden,
			Msg:    "Forbidden",
			Err:    errors.New("user is not permitted to access any documents for this index"),
		}
	} else if len(should) == 1 {
		return should[0], nil
	}

	return elastic.NewBoolQuery().Should(should...), nil
}

func (h dnsLogsIndexHelper) NewTimeRangeQuery(from, to time.Time) elastic.Query {
	return elastic.NewRangeQuery("end_time").Gt(from).Lte(to)
}

func (h dnsLogsIndexHelper) GetTimeField() string {
	return "end_time"
}

func (h dnsLogsIndexHelper) GetIndex(cluster string) string {
	return fmt.Sprintf("%s.%s.*", esDnsLogsIndexPrefix, cluster)
}
