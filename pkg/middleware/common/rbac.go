// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package common

import (
	"github.com/olivere/elastic/v7"

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
)

// GetRBACQuery converts the permissions specified in the authroized resource verbs into an elastic search query
// that limits results to resources the user may list.
func GetRBACQuery(verbs []apiv3.AuthorizedResourceVerbs) elastic.Query {
	// Convert the permissions into a query that each flow must satisfy - essentially a source or destination must
	// be listable by the user to be included in the response.
	var should []elastic.Query
	for _, r := range verbs {
		for _, v := range r.Verbs {
			if v.Verb != "list" {
				// Only interested in the list verbs.
				continue
			}
			for _, rg := range v.ResourceGroups {
				var query elastic.Query
				switch r.Resource {
				case "hostendpoints":
					// HostEndpoints are neither tiered nor namespaced, and AuthorizationReview does not determine
					// RBAC at the instance level, so must be able to list all HostEndpoints.
					query = elastic.NewBoolQuery().Should(
						elastic.NewTermQuery("source_type", "hep"),
						elastic.NewTermQuery("dest_type", "hep"),
					)
				case "networksets":
					if rg.Namespace == "" {
						// Can list all NetworkSets. Check type is "ns" and namespace is not "-" (which would be a
						// GlobalNetworkSet).
						query = elastic.NewBoolQuery().Should(
							elastic.NewBoolQuery().Must(
								elastic.NewTermQuery("source_type", "ns"),
							).MustNot(
								elastic.NewTermQuery("source_namespace", "-"),
							),
							elastic.NewBoolQuery().Must(
								elastic.NewTermQuery("dest_type", "ns"),
							).MustNot(
								elastic.NewTermQuery("dest_namespace", "-"),
							),
						)
					} else {
						// Can list NetworkSets in a specific namespace. Check type is "ns" and namespace matches.
						query = elastic.NewBoolQuery().Should(
							elastic.NewBoolQuery().Must(
								elastic.NewTermQuery("source_type", "ns"),
								elastic.NewTermQuery("source_namespace", rg.Namespace),
							),
							elastic.NewBoolQuery().Must(
								elastic.NewTermQuery("dest_type", "ns"),
								elastic.NewTermQuery("dest_namespace", rg.Namespace),
							),
						)
					}
				case "globalnetworksets":
					// GlobalNetworkSets are neither tiered nor namespaced, and AuthorizationReview does not determine
					// RBAC at the instance level, so must be able to list all GlobalNetworkSets. Check type is "ns"
					// and namespace is "-".
					query = elastic.NewBoolQuery().Should(
						elastic.NewBoolQuery().Must(
							elastic.NewTermQuery("source_type", "ns"),
							elastic.NewTermQuery("source_namespace", "-"),
						),
						elastic.NewBoolQuery().Must(
							elastic.NewTermQuery("dest_type", "ns"),
							elastic.NewTermQuery("dest_namespace", "-"),
						),
					)
				case "pods":
					if rg.Namespace == "" {
						// Can list all Pods. Check type is "wep".
						query = elastic.NewBoolQuery().Should(
							elastic.NewTermQuery("source_type", "wep"),
							elastic.NewTermQuery("dest_type", "wep"),
						)
					} else {
						// Can list Pods in a specific namespace. Check type is "wep" and namespace matches.
						query = elastic.NewBoolQuery().Should(
							elastic.NewBoolQuery().Must(
								elastic.NewTermQuery("source_type", "wep"),
								elastic.NewTermQuery("source_namespace", rg.Namespace),
							),
							elastic.NewBoolQuery().Must(
								elastic.NewTermQuery("dest_type", "wep"),
								elastic.NewTermQuery("dest_namespace", rg.Namespace),
							),
						)
					}
				}

				if query != nil {
					should = append(should, query)
				}
			}
			break
		}
	}

	return elastic.NewBoolQuery().Should(should...)
}
