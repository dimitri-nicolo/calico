// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package middleware

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
	authzv1 "k8s.io/api/authorization/v1"
)

// Request properties to indicate the cluster used for proxying and RBAC.
const (
	clusterParam       = "cluster"
	clusterIdHeader    = "x-cluster-id"
	defaultClusterName = "cluster"
)

var legacyURLPath, extractIndexPattern, datelessIndexPattern *regexp.Regexp

var queryResourceMap map[string]string

func init() {
	// This regexp matches legacy queries, for example: "/tigera-elasticsearch/tigera_secure_ee_flows.cluster.*/_search"
	legacyURLPath = regexp.MustCompile(`^.*/tigera_secure_ee_.*/_search$`)
	// This regexp extracts the index from a legacy query URL path
	extractIndexPattern = regexp.MustCompile(`/(tigera_secure_ee_[_a-z]*)[.*]?.*/_search`)
	datelessIndexPattern = regexp.MustCompile(`^tigera_secure_ee_events\*?$`)

	queryResourceMap = map[string]string{
		"tigera_secure_ee_flows":      "flows",
		"tigera_secure_ee_audit_":     "audit*", // support both audit_*
		"tigera_secure_ee_audit":      "audit*", // and audit*
		"tigera_secure_ee_audit_ee":   "audit_ee",
		"tigera_secure_ee_audit_kube": "audit_kube",
		"tigera_secure_ee_events":     "events",
		"tigera_secure_ee_dns":        "dns",
		"flowLogNames":                "flows",
		"flowLogNamespaces":           "flows",
		"flowLogs":                    "flows",
	}
}

// The handler returned by this will add a ResourceAttribute to the context
// of the request based on the request.URL.Path. The ResourceAttribute
// is intended to be used with a SelfSubjectAccessReview or SubjectAccessReview
// to check if a user has access to the resource.
// Upon successful conversion/context update, the handler passed in will be
// called, otherwise the ResponseWriter will be updated with the appropriate
// status and a message with details.
func RequestToResource(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		cluster, resourceName, urlPath, err := getResourcesFromReq(req)
		if err != nil {
			log.WithError(err).Debugf("Unable to convert request URL '%+v' to resource", req.URL)
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		newReq := req.WithContext(NewContextWithReviewResource(req.Context(), getResourceAttributes(cluster, resourceName)))
		newReq.URL.Path = urlPath
		newReq.URL.RawPath = urlPath
		h.ServeHTTP(w, newReq)
	})
}

type contextKey int

const (
	ResourceAttributeKey contextKey = iota
	NonResourceAttributeKey
)

func NewContextWithReviewResource(
	ctx context.Context,
	ra *authzv1.ResourceAttributes,
) context.Context {
	return context.WithValue(ctx, ResourceAttributeKey, ra)
}

func NewContextWithReviewNonResource(
	ctx context.Context,
	ra *authzv1.NonResourceAttributes,
) context.Context {
	return context.WithValue(ctx, NonResourceAttributeKey, ra)
}

func FromContextGetReviewResource(ctx context.Context) (*authzv1.ResourceAttributes, bool) {
	ra, ok := ctx.Value(ResourceAttributeKey).(*authzv1.ResourceAttributes)
	return ra, ok
}

func FromContextGetReviewNonResource(ctx context.Context) (*authzv1.NonResourceAttributes, bool) {
	nra, ok := ctx.Value(NonResourceAttributeKey).(*authzv1.NonResourceAttributes)
	return nra, ok
}

func getResourceAttributes(cluster, resourceName string) *authzv1.ResourceAttributes {
	return &authzv1.ResourceAttributes{
		Verb:     "get",
		Group:    "lma.tigera.io",
		Resource: cluster,
		Name:     resourceName,
	}
}

// getResourcesFromReq parses the req.URL.Path, returns the cluster and resourceName used for RBAC
func getResourcesFromReq(req *http.Request) (string, string, string, error) {
	var resource string
	var cluster string

	if req.URL == nil {
		return cluster, resource, "", fmt.Errorf("no URL in request")
	}

	urlPath := req.URL.Path
	if legacyURLPath.MatchString(req.URL.Path) {
		// This is a legacy Elasticsearch query
		cluster, resource, urlPath = parseLegacyURLPath(req)
	} else {
		// This must be a query according to the flowLog api spec
		var err error
		cluster, resource, err = parseFlowLogURLPath(req)
		if err != nil {
			return cluster, resource, urlPath, err
		}
	}

	if resource == "" {
		return cluster, resource, urlPath, fmt.Errorf("invalid resource in path '%s'", req.URL.Path)
	}

	if cluster == "" {
		cluster = defaultClusterName
	}

	return cluster, resource, urlPath, nil
}

// FlogLog api, see: https://docs.google.com/document/d/1kUPDVn_tcehRrHn_nhm8GFILCaeOZ7u7pLm68Zv3Yng
// A request might look like /flowLogs?cluster=my-cluster
// returns <cluster>, <index>, err
func parseFlowLogURLPath(req *http.Request) (string, string, error) {
	path := req.URL.Path
	pathSlice := strings.Split(path, "/")
	pathSliceLen := len(pathSlice)

	if pathSliceLen < 2 {
		return "", "", nil
	}

	path = pathSlice[pathSliceLen-1] // Keep only the last part of the path

	values, err := url.ParseQuery(req.URL.RawQuery)
	if err != nil {
		return "", "", fmt.Errorf("unable to parse query parameters of request %s", req.URL.RawQuery)
	}
	clusters := values[clusterParam]
	res, _ := queryToResource(path)
	if len(clusters) > 0 {
		return clusters[0], res, nil
	}
	return defaultClusterName, res, nil
}

// This is a legacy request with a path such as: "some/path/<index>.<cluster>.*/_search".
// We return a (corrected) url path that does not query unauthorized clusters.
// returns <cluster>, <index>, urlPath, err
func parseLegacyURLPath(req *http.Request) (string, string, string) {
	// Extract groups such that:
	// - group 0 would match "/<index>.<cluster>.*/_search"
	// - group 1 would match "<index>"
	match := extractIndexPattern.FindStringSubmatch(req.URL.Path)
	if len(match) != 2 {
		return "", "", ""
	}
	idx := match[1]
	res, _ := queryToResource(idx)

	clu := defaultClusterName
	if req.Header != nil {
		xclusterid := req.Header.Get(clusterIdHeader)
		if xclusterid != "" {
			clu = xclusterid
		}
	}

	// path would be a replacement for match[1]
	// This lets us create the actual ES query that always includes the cluster name.
	var path string
	if strings.Contains(match[0], "*") {
		// certain indices don't have date suffix and adding .* to the end will not match the index we need,
		// as the . is considered mandatory.
		if datelessIndexPattern.MatchString(idx) {
			path = fmt.Sprintf("/%s.%s/_search", idx, clu)
		} else {
			path = fmt.Sprintf("/%s.%s.*/_search", idx, clu)
		}
	} else {
		path = fmt.Sprintf("/%s.%s/_search", idx, clu)
	}

	return clu, res, strings.Replace(req.URL.Path, match[0], path, 1)
}

// queryToResource maps indexes into resource names used in RBAC
// implements the table located in
// https://docs.google.com/document/d/1wFrbjLydsdz0NfxVk-_iW7eqx4ZIZWfgj5SzcsRmTwo/edit#heading=h.pva3ex6ffysc
func queryToResource(query string) (string, bool) {
	str, ok := queryResourceMap[query]
	return str, ok
}
