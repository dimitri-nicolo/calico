// Copyright (c) 2018-2020 Tigera, Inc. All rights reserved.
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"

	"fmt"

	libapi "github.com/projectcalico/calico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/calico/libcalico-go/lib/backend/model"
	cerrors "github.com/projectcalico/calico/libcalico-go/lib/errors"
	"github.com/projectcalico/calico/ts-queryserver/pkg/querycache/client"

	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
)

const (
	QueryLabelPrefix         = "label_"
	QuerySelector            = "selector"
	QueryPolicy              = "policy"
	QueryNode                = "node"
	QueryRuleDirection       = "ruleDirection"
	QueryRuleIndex           = "ruleIndex"
	QueryRuleEntity          = "ruleEntity"
	QueryRuleNegatedSelector = "ruleNegatedSelector"
	QueryPageNum             = "page"
	QueryNumPerPage          = "maxItems"
	QuerySortBy              = "sortBy"
	QueryReverseSort         = "reverseSort"
	QueryEndpoint            = "endpoint"
	QueryUnmatched           = "unmatched"
	QueryUnprotected         = "unprotected"
	QueryUnlabelled          = "unlabelled"
	QueryTier                = "tier"
	QueryNetworkSet          = "networkset"

	AllResults     = "all"
	resultsPerPage = 100

	numURLSegmentsWithName = 3
)

var (
	errorPolicyMultiParm = errors.New("invalid query: specify only one of " + QueryEndpoint +
		" or " + QueryUnmatched)
	errorEndpointMultiParm = errors.New("invalid query: specify only one of " + QuerySelector +
		" or " + QueryPolicy + ", or specify one of " + QueryPolicy + " or " + QueryUnprotected)
	errorInvalidEndpointName = errors.New("invalid query: the endpoint name is not valid; it should be of the format " +
		"<HostEndpoint name> or <namespace>/<WorkloadEndpoint name>")
	errorInvalidPolicyName = errors.New("invalid query: the policy name is not valid; it should be of the format " +
		"<GlobalNetworkPolicy name> or <namespace>/<NetworkPolicy name>")
	errorInvalidNetworkSetName = errors.New("invalid query: the networkset name is not valid; it should be of the format " +
		"<GlobalNetworkSet name> or <namespace>/<NetworkSet name>")
	errorInvalidEndpointURL = errors.New("the URL does not contain a valid endpoint name; the final URL segments should " +
		"be of the format <HostEndpoint name> or <namespace>/<WorkloadEndpoint name>")
	errorInvalidPolicyURL = errors.New("the URL does not contain a valid policy name; the final URL segments should " +
		"be of the format <GlobalNetworkPolicy name> or <namespace>/<NetworkPolicy name>")
	errorInvalidNodeURL = errors.New("the URL does not contain a valid node name; the final URL segments should " +
		"be of the format <Node name>")
)

type Query interface {
	Policy(w http.ResponseWriter, r *http.Request)
	Policies(w http.ResponseWriter, r *http.Request)
	Node(w http.ResponseWriter, r *http.Request)
	Nodes(w http.ResponseWriter, r *http.Request)
	Endpoint(w http.ResponseWriter, r *http.Request)
	Endpoints(w http.ResponseWriter, r *http.Request)
	Summary(w http.ResponseWriter, r *http.Request)
}

func NewQuery(qi client.QueryInterface) Query {
	return &query{qi: qi}
}

type query struct {
	qi client.QueryInterface
}

func (q *query) Summary(w http.ResponseWriter, r *http.Request) {
	q.runQuery(w, r, client.QueryClusterReq{}, false)
}

func (q *query) Endpoints(w http.ResponseWriter, r *http.Request) {
	selector := r.URL.Query().Get(QuerySelector)
	policies, err := q.getPolicies(r)
	if err != nil {
		q.writeError(w, err, http.StatusBadRequest)
		return
	}
	unprotected := q.getBool(r, QueryUnprotected)
	if (selector != "" && len(policies) > 0) || (unprotected && len(policies) > 0) || len(policies) > 1 {
		q.writeError(w, errorEndpointMultiParm, http.StatusBadRequest)
		return
	}
	var policy model.Key
	if len(policies) > 0 {
		policy = policies[0]
	}
	page, err := q.getPage(r)
	if err != nil {
		q.writeError(w, err, http.StatusBadRequest)
		return
	}

	q.runQuery(w, r, client.QueryEndpointsReq{
		Selector:            selector,
		Policy:              policy,
		Unprotected:         unprotected,
		RuleDirection:       r.URL.Query().Get(QueryRuleDirection),
		RuleIndex:           q.getInt(r, QueryRuleIndex, 0),
		RuleEntity:          r.URL.Query().Get(QueryRuleEntity),
		RuleNegatedSelector: q.getBool(r, QueryRuleNegatedSelector),
		Unlabelled:          q.getBool(r, QueryUnlabelled),
		Page:                page,
		Sort:                q.getSort(r),
		Node:                r.URL.Query().Get(QueryNode),
	}, false)
}

func (q *query) Endpoint(w http.ResponseWriter, r *http.Request) {
	urlParts := strings.SplitN(r.URL.Path, "/", numURLSegmentsWithName)
	if len(urlParts) != numURLSegmentsWithName {
		q.writeError(w, errorInvalidEndpointURL, http.StatusBadRequest)
		return
	}
	endpoint, ok := q.getEndpointKeyFromCombinedName(urlParts[numURLSegmentsWithName-1])
	if !ok {
		q.writeError(w, errorInvalidEndpointURL, http.StatusBadRequest)
		return
	}
	q.runQuery(w, r, client.QueryEndpointsReq{
		Endpoint: endpoint,
	}, true)
}

func (q *query) Policies(w http.ResponseWriter, r *http.Request) {
	endpoints, err := q.getEndpoints(r)
	if err != nil {
		q.writeError(w, err, http.StatusBadRequest)
		return
	}
	networksets, err := q.getNetworkSets(r)
	if err != nil {
		q.writeError(w, err, http.StatusBadRequest)
		return
	}

	unmatched := q.getBool(r, QueryUnmatched)
	if (unmatched && (len(endpoints) > 0 || len(networksets) > 0)) || len(endpoints) > 1 || len(networksets) > 1 {
		q.writeError(w, errorPolicyMultiParm, http.StatusBadRequest)
		return
	}

	var endpoint model.Key
	if len(endpoints) > 0 {
		endpoint = endpoints[0]
	}
	var networkset model.Key
	if len(networksets) > 0 {
		networkset = networksets[0]
	}

	page, err := q.getPage(r)
	if err != nil {
		q.writeError(w, err, http.StatusBadRequest)
		return
	}
	q.runQuery(w, r, client.QueryPoliciesReq{
		Tier:       r.URL.Query().Get(QueryTier),
		Labels:     q.getLabels(r),
		Endpoint:   endpoint,
		NetworkSet: networkset,
		Unmatched:  unmatched,
		Page:       page,
		Sort:       q.getSort(r),
	}, false)
}

func (q *query) Policy(w http.ResponseWriter, r *http.Request) {
	urlParts := strings.SplitN(r.URL.Path, "/", numURLSegmentsWithName)
	if len(urlParts) != numURLSegmentsWithName {
		q.writeError(w, errorInvalidPolicyURL, http.StatusBadRequest)
		return
	}
	policy, ok := q.getPolicyKeyFromCombinedName(urlParts[numURLSegmentsWithName-1])
	if !ok {
		q.writeError(w, errorInvalidPolicyURL, http.StatusBadRequest)
		return
	}
	q.runQuery(w, r, client.QueryPoliciesReq{
		Policy: policy,
	}, true)
}

func (q *query) Nodes(w http.ResponseWriter, r *http.Request) {
	page, err := q.getPage(r)
	if err != nil {
		q.writeError(w, err, http.StatusBadRequest)
		return
	}
	q.runQuery(w, r, client.QueryNodesReq{
		Page: page,
		Sort: q.getSort(r),
	}, false)
}

func (q *query) Node(w http.ResponseWriter, r *http.Request) {
	urlParts := strings.SplitN(r.URL.Path, "/", numURLSegmentsWithName)
	if len(urlParts) != numURLSegmentsWithName {
		q.writeError(w, errorInvalidNodeURL, http.StatusBadRequest)
		return
	}
	node, ok := q.getNodeKeyFromCombinedName(urlParts[numURLSegmentsWithName-1])
	if !ok {
		q.writeError(w, errorInvalidNodeURL, http.StatusBadRequest)
		return
	}
	q.runQuery(w, r, client.QueryNodesReq{
		Node: node,
	}, true)
}

// Convert a query parameter to a uint. We are pretty relaxed about what we accept, using the
// default or min value when the requested value is bogus.
func (q *query) getInt(r *http.Request, queryParm string, def int) int {
	qp := r.URL.Query().Get(queryParm)
	if len(qp) == 0 {
		return def
	}
	val, err := strconv.ParseInt(qp, 0, 0)
	if err != nil {
		return def
	}

	return int(val)
}

func (q *query) getBool(r *http.Request, queryParm string) bool {
	qp := strings.ToLower(r.URL.Query().Get(queryParm))
	return qp == "true" || qp == "t" || qp == "1" || qp == "y" || qp == "yes"
}

func (q *query) getLabels(r *http.Request) map[string]string {
	parms := r.URL.Query()
	labels := make(map[string]string)
	for k, pvs := range parms {
		if strings.HasPrefix(k, QueryLabelPrefix) {
			labels[strings.TrimPrefix(k, QueryLabelPrefix)] = pvs[0]
		}
	}
	return labels
}

func (q *query) getEndpoints(r *http.Request) ([]model.Key, error) {
	eps := r.URL.Query()[QueryEndpoint]
	reps := make([]model.Key, 0, len(eps))
	for _, ep := range eps {
		rep, ok := q.getEndpointKeyFromCombinedName(ep)
		if !ok {
			return nil, errorInvalidEndpointName
		}
		reps = append(reps, rep)
	}
	return reps, nil
}

func (q *query) getPolicies(r *http.Request) ([]model.Key, error) {
	pols := r.URL.Query()[QueryPolicy]
	rpols := make([]model.Key, 0, len(pols))
	for _, pol := range pols {
		rpol, ok := q.getPolicyKeyFromCombinedName(pol)
		if !ok {
			return nil, errorInvalidPolicyName
		}
		rpols = append(rpols, rpol)
	}
	return rpols, nil
}

func (q *query) getNetworkSets(r *http.Request) ([]model.Key, error) {
	netsets := r.URL.Query()[QueryNetworkSet]
	rnetsets := make([]model.Key, 0, len(netsets))
	for _, netset := range netsets {
		rnetset, ok := q.getNetworkSetKeyFromCombinedName(netset)
		if !ok {
			return nil, errorInvalidNetworkSetName
		}
		rnetsets = append(rnetsets, rnetset)
	}
	return rnetsets, nil
}

func (q *query) getNameAndNamespaceFromCombinedName(combined string) ([]string, bool) {
	parts := strings.Split(combined, "/")
	for _, part := range parts {
		if part == "" {
			return nil, false
		}
	}
	if len(parts) != 1 && len(parts) != 2 {
		return nil, false
	}
	return parts, true
}

func (q *query) getPolicyKeyFromCombinedName(combined string) (model.Key, bool) {
	logCxt := log.WithField("name", combined)
	logCxt.Info("Extracting policy key from combined name")
	parts, ok := q.getNameAndNamespaceFromCombinedName(combined)
	if !ok {
		logCxt.Info("Unable to extract name or namespace and name")
		return nil, false
	}
	switch len(parts) {
	case 1:
		logCxt.Info("Returning GNP")
		return model.ResourceKey{
			Kind: apiv3.KindGlobalNetworkPolicy,
			Name: parts[0],
		}, true
	case 2:
		logCxt.Info("Returning NP")
		return model.ResourceKey{
			Kind:      apiv3.KindNetworkPolicy,
			Namespace: parts[0],
			Name:      parts[1],
		}, true
	}
	return nil, false
}

func (q *query) getEndpointKeyFromCombinedName(combined string) (model.Key, bool) {
	parts, ok := q.getNameAndNamespaceFromCombinedName(combined)
	if !ok {
		return nil, false
	}
	switch len(parts) {
	case 1:
		return model.ResourceKey{
			Kind: apiv3.KindHostEndpoint,
			Name: parts[0],
		}, true
	case 2:
		return model.ResourceKey{
			Kind:      libapi.KindWorkloadEndpoint,
			Namespace: parts[0],
			Name:      parts[1],
		}, true
	}
	return nil, false
}

func (q *query) getNodeKeyFromCombinedName(combined string) (model.Key, bool) {
	parts, ok := q.getNameAndNamespaceFromCombinedName(combined)
	if !ok || len(parts) != 1 {
		return nil, false
	}
	return model.ResourceKey{
		Kind: libapi.KindNode,
		Name: parts[0],
	}, true
}

func (q *query) getNetworkSetKeyFromCombinedName(combined string) (model.Key, bool) {
	parts, ok := q.getNameAndNamespaceFromCombinedName(combined)
	if !ok {
		return nil, false
	} else if len(parts) == 2 {
		return model.ResourceKey{
			Kind:      apiv3.KindNetworkSet,
			Name:      parts[1],
			Namespace: parts[0],
		}, true
	} else if len(parts) == 1 {
		return model.ResourceKey{
			Kind: apiv3.KindGlobalNetworkSet,
			Name: parts[0],
		}, true
	}

	log.WithField("name", combined).Info("Extracting policy key from combined name failed with unknown name format")
	return nil, false
}

func (q *query) runQuery(w http.ResponseWriter, r *http.Request, req interface{}, exact bool) {
	resp, err := q.qi.RunQuery(context.Background(), req)
	if _, ok := err.(cerrors.ErrorResourceDoesNotExist); ok && exact {
		// This is an exact get and the resource does not exist. Return a 404 not found.
		q.writeError(w, err, http.StatusNotFound)
		return
	} else if err != nil {
		// All other errors return as a 400 Bad request.
		q.writeError(w, err, http.StatusBadRequest)
		return
	}

	js, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		q.writeError(w, err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(js)
	_, _ = w.Write([]byte{'\n'})
}

func (q *query) writeError(w http.ResponseWriter, err error, code int) {
	http.Error(w, "Error: "+err.Error(), code)
}

func (q *query) getPage(r *http.Request) (*client.Page, error) {
	if r.URL.Query().Get(QueryPageNum) == AllResults {
		return nil, nil
	}
	// We perform as much sanity checking as we can without performing an actual query.
	pageNum := q.getInt(r, QueryPageNum, 0)
	numPerPage := q.getInt(r, QueryNumPerPage, resultsPerPage)

	if pageNum < 0 {
		return nil, fmt.Errorf("page number should be an integer >=0, requested number: %d", pageNum)
	}
	if numPerPage <= 0 {
		return nil, fmt.Errorf("number of results must be >0, requested number: %d", numPerPage)
	}

	return &client.Page{
		PageNum:    pageNum,
		NumPerPage: numPerPage,
	}, nil
}

func (q *query) getSort(r *http.Request) *client.Sort {
	sortBy := r.URL.Query()[QuerySortBy]
	reverse := q.getBool(r, QueryReverseSort)
	if len(sortBy) == 0 && !reverse {
		return nil
	}
	return &client.Sort{
		SortBy:  sortBy,
		Reverse: reverse,
	}
}
