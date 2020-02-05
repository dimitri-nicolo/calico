// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package middleware

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
	authzv1 "k8s.io/api/authorization/v1"
	"github.com/tigera/lma/pkg/auth"
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
		cluster, resourceName, urlPath, err := parseURLPath(req)
		if err != nil {
			log.WithError(err).Debugf("Unable to convert request URL '%+v' to resource", req.URL)
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		newReq := req.WithContext(auth.NewContextWithReviewResource(req.Context(), getResourceAttributes(cluster, resourceName)))
		newReq.URL.Path = urlPath
		newReq.URL.RawPath = urlPath
		h.ServeHTTP(w, newReq)
	})
}

// Get the resource attributes for an RBAC request based on the cluster you want to access.
func getResourceAttributes(cluster, resourceName string) *authzv1.ResourceAttributes {
	return &authzv1.ResourceAttributes{
		Verb:     "get",
		Group:    "lma.tigera.io",
		Resource: cluster,
		Name:     resourceName,
	}
}

// parseURLPath is compatible with the new flow log api, as well as the soon deprecated legacy api. If the request
// is made to a legacy resource, we inspect the request header in addition to the req.URL.Path.
// returns <cluster>, <index>, <req.url.path>, err
func parseURLPath(req *http.Request) (cluster, index, urlPath string, err error) {
	if req.URL == nil {
		return cluster, index, urlPath, fmt.Errorf("no URL in request")
	}

	if legacyURLPath.MatchString(req.URL.Path) {
		// This is a legacy Elasticsearch query
		cluster, index, urlPath = parseLegacyURLPath(req)
	} else {
		// This must be a query according to the flowLog api spec
		var err error
		cluster, index, err = parseFlowLogURLPath(req)
		if err != nil {
			return cluster, index, urlPath, err
		}
	}

	if index == "" {
		return cluster, index, urlPath, fmt.Errorf("invalid resource in path '%s'", req.URL.Path)
	}

	if cluster == "" {
		cluster = defaultClusterName
	}

	return cluster, index, urlPath, nil
}

// FlogLog api, see: https://docs.google.com/document/d/1kUPDVn_tcehRrHn_nhm8GFILCaeOZ7u7pLm68Zv3Yng
// A request might look like /flowLogs?cluster=my-cluster
// returns <cluster>, <index>, err
func parseFlowLogURLPath(req *http.Request) (cluster, index string, err error) {
	path := req.URL.Path
	pathSlice := strings.Split(path, "/")
	pathSliceLen := len(pathSlice)

	if pathSliceLen < 2 {
		return cluster, index, nil
	}

	path = pathSlice[pathSliceLen-1] // Keep only the last part of the path

	values, err := url.ParseQuery(req.URL.RawQuery)
	if err != nil {
		return cluster, index, fmt.Errorf("unable to parse query parameters of request %s", req.URL.RawQuery)
	}
	clusters := values[clusterParam]
	index, _ = queryToResource(path)
	if len(clusters) > 0 {
		return clusters[0], index, nil
	}
	return defaultClusterName, index, nil
}

// This is a legacy request with a path such as: "some/path/<index>.<cluster>.*/_search".
// We return a (corrected) url path that does not query unauthorized clusters.
// returns <cluster>, <index>, urlPath, err
func parseLegacyURLPath(req *http.Request) (cluster, index, err string) {
	// Extract groups such that:
	// - group 0 would match "/<index>.<cluster>.*/_search"
	// - group 1 would match "<index>"
	match := extractIndexPattern.FindStringSubmatch(req.URL.Path)
	if len(match) != 2 {
		return cluster, index, err
	}
	idx := match[1]
	index, _ = queryToResource(idx)

	cluster = defaultClusterName
	if req.Header != nil {
		xclusterid := req.Header.Get(clusterIdHeader)
		if xclusterid != "" {
			cluster = xclusterid
		}
	}

	// path would be a replacement for match[1]
	// This lets us create the actual ES query that always includes the cluster name.
	var path string
	if strings.Contains(match[0], "*") {
		// certain indices don't have date suffix and adding .* to the end will not match the index we need,
		// as the . is considered mandatory.
		if datelessIndexPattern.MatchString(idx) {
			path = fmt.Sprintf("/%s.%s/_search", idx, cluster)
		} else {
			path = fmt.Sprintf("/%s.%s.*/_search", idx, cluster)
		}
	} else {
		path = fmt.Sprintf("/%s.%s/_search", idx, cluster)
	}

	return cluster, index, strings.Replace(req.URL.Path, match[0], path, 1)
}

// queryToResource maps indexes into resource names used in RBAC
// implements the table located in
// https://docs.google.com/document/d/1wFrbjLydsdz0NfxVk-_iW7eqx4ZIZWfgj5SzcsRmTwo/edit#heading=h.pva3ex6ffysc
func queryToResource(query string) (string, bool) {
	str, ok := queryResourceMap[query]
	return str, ok
}
