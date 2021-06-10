// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package elastic

import (
	"errors"
	"net/http"

	"github.com/olivere/elastic/v7"

	"github.com/tigera/lma/pkg/httputils"

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
)

// GetRBACQueryForFlowLogs converts the permissions specified in the authorized resource verbs into an elastic search
// query for flow logs that limits results to resources the user may list.
//
// Returns the query used to limit the responses based on RBAC.
func GetRBACQueryForFlowLogs(resources []apiv3.AuthorizedResourceVerbs) (elastic.Query, error) {
	// Convert the permissions into a query that each flow must satisfy - essentially a source or destination must
	// be listable by the user to be included in the response.
	var should []elastic.Query
	for _, r := range resources {
		for _, v := range r.Verbs {
			if v.Verb != "list" {
				// Only interested in the list verbs.
				continue
			}
			for _, rg := range v.ResourceGroups {
				switch r.Resource {
				case "hostendpoints":
					// HostEndpoints are neither tiered nor namespaced, and AuthorizationReview does not determine
					// RBAC at the instance level, so must be able to list all HostEndpoints.
					should = append(should,
						elastic.NewTermQuery("source_type", "hep"),
						elastic.NewTermQuery("dest_type", "hep"),
					)
				case "networksets":
					if rg.Namespace == "" {
						// Can list all NetworkSets. Check type is "ns" and namespace is not "-" (which would be a
						// GlobalNetworkSet).
						should = append(should,
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
						should = append(should,
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
					should = append(should,
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
						should = append(should,
							elastic.NewTermQuery("source_type", "wep"),
							elastic.NewTermQuery("dest_type", "wep"),
						)
					} else {
						// Can list Pods in a specific namespace. Check type is "wep" and namespace matches.
						should = append(should,
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
	}

	return elastic.NewBoolQuery().Should(should...), nil
}

// GetRBACQueryForL7Logs converts the permissions specified in the authorized resource verbs into an elastic search
// query for L7 logs that limits results to resources the user may list.
//
// Returns the query used to limit the responses based on RBAC.
func GetRBACQueryForL7Logs(resources []apiv3.AuthorizedResourceVerbs) (elastic.Query, error) {
	// Convert the permissions into a query that each flow must satisfy - essentially a source or destination must
	// be listable by the user to be included in the response.
	var should []elastic.Query
	for _, r := range resources {
		for _, v := range r.Verbs {
			if v.Verb != "list" {
				// Only interested in the list verbs.
				continue
			}
			for _, rg := range v.ResourceGroups {
				switch r.Resource {
				case "hostendpoints":
					// HostEndpoints are neither tiered nor namespaced, and AuthorizationReview does not determine
					// RBAC at the instance level, so must be able to list all HostEndpoints.
					should = append(should,
						elastic.NewTermQuery("src_type", "hep"),
						elastic.NewTermQuery("dest_type", "hep"),
					)
				case "networksets":
					if rg.Namespace == "" {
						// Can list all NetworkSets. Check type is "ns" and namespace is not "-" (which would be a
						// GlobalNetworkSet).
						should = append(should,
							elastic.NewBoolQuery().Must(
								elastic.NewTermQuery("src_type", "ns"),
							).MustNot(
								elastic.NewTermQuery("src_namespace", "-"),
							),
							elastic.NewBoolQuery().Must(
								elastic.NewTermQuery("dest_type", "ns"),
							).MustNot(
								elastic.NewTermQuery("dest_namespace", "-"),
							),
						)
					} else {
						// Can list NetworkSets in a specific namespace. Check type is "ns" and namespace matches.
						should = append(should,
							elastic.NewBoolQuery().Must(
								elastic.NewTermQuery("src_type", "ns"),
								elastic.NewTermQuery("src_namespace", rg.Namespace),
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
					should = append(should,
						elastic.NewBoolQuery().Must(
							elastic.NewTermQuery("src_type", "ns"),
							elastic.NewTermQuery("src_namespace", "-"),
						),
						elastic.NewBoolQuery().Must(
							elastic.NewTermQuery("dest_type", "ns"),
							elastic.NewTermQuery("dest_namespace", "-"),
						),
					)
				case "pods":
					if rg.Namespace == "" {
						// Can list all Pods. Check type is "wep".
						should = append(should,
							elastic.NewTermQuery("src_type", "wep"),
							elastic.NewTermQuery("dest_type", "wep"),
						)
					} else {
						// Can list Pods in a specific namespace. Check type is "wep" and namespace matches.
						should = append(should,
							elastic.NewBoolQuery().Must(
								elastic.NewTermQuery("src_type", "wep"),
								elastic.NewTermQuery("src_namespace", rg.Namespace),
							),
							elastic.NewBoolQuery().Must(
								elastic.NewTermQuery("dest_type", "wep"),
								elastic.NewTermQuery("dest_namespace", rg.Namespace),
							),
						)
					}
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
	}

	return elastic.NewBoolQuery().Should(should...), nil
}

// GetRBACQueryForDNSLogs converts the permissions specified in the authorized resource verbs into an elastic search
// query for DNS logs that limits results to resources the user may list.
//
// Returns the query used to limit the responses based on RBAC, if nil then user has full permissions.
func GetRBACQueryForDNSLogs(resources []apiv3.AuthorizedResourceVerbs) (elastic.Query, error) {
	// Convert the permissions into a query that each flow must satisfy - essentially a source or destination must
	// be listable by the user to be included in the response.
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
