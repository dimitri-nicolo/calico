// Copyright (c) 2018 Tigera, Inc. All rights reserved.
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/tigera/calicoq/web/pkg/querycache/client"
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
	QueryTier                = "tier"

	AllResults     = "all"
	resultsPerPage = 100

	numURLSegmentsWithName = 3
)

var (
	errorPolicyMultiParm = errors.New("specify only one of " + QueryEndpoint +
		" or " + QueryUnmatched)
	errorEndpointMultiParm = errors.New("specify only one of " + QuerySelector +
		" or " + QueryPolicy)
	errorInvalidEndpointName = errors.New("the endpoint name is not valid; it should be of the format " +
		"<HostEndpoint name> or <namespace>/<WorkloadEndpoint name>")
	errorInvalidPolicyName = errors.New("the policy name is not valid; it should be of the format " +
		"<GlobalNetworkPolicy name> or <namespace>/<NetworkPolicy name>")
	errorInvalidEndpointURL = errors.New("the URL does not contain a valid endpoint name; the final URL segments should " +
		"be of the format <HostEndpoint name> or <namespace>/<WorkloadEndpoint name>")
	errorInvalidPolicyURL = errors.New("the URL does not contain a valid endpoint name; the final URL segments should " +
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
	q.runQuery(w, r, client.QueryClusterReq{})
}

func (q *query) Endpoints(w http.ResponseWriter, r *http.Request) {
	selector := r.URL.Query().Get(QuerySelector)
	policies, err := q.getPolicies(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	unprotected := q.getBool(r, QueryUnprotected)
	if (selector != "" && len(policies) > 0) || (unprotected && len(policies) > 0) || len(policies) > 1 {
		http.Error(w, errorEndpointMultiParm.Error(), http.StatusBadRequest)
		return
	}
	var policy model.Key
	if len(policies) > 0 {
		policy = policies[0]
	}
	q.runQuery(w, r, client.QueryEndpointsReq{
		Selector:            selector,
		Policy:              policy,
		Unprotected:         unprotected,
		RuleDirection:       r.URL.Query().Get(QueryRuleDirection),
		RuleIndex:           q.getInt(r, QueryRuleIndex, 0),
		RuleEntity:          r.URL.Query().Get(QueryRuleEntity),
		RuleNegatedSelector: q.getBool(r, QueryRuleNegatedSelector),
		Page:                q.getPage(r),
		Sort:                q.getSort(r),
		Node:                r.URL.Query().Get(QueryNode),
	})
}

func (q *query) Endpoint(w http.ResponseWriter, r *http.Request) {
	urlParts := strings.SplitN(r.URL.Path, "/", numURLSegmentsWithName)
	if len(urlParts) != numURLSegmentsWithName {
		http.Error(w, errorInvalidEndpointURL.Error(), http.StatusBadRequest)
		return
	}
	endpoint, ok := q.getEndpointKeyFromCombinedName(urlParts[numURLSegmentsWithName-1])
	if !ok {
		http.Error(w, errorInvalidEndpointURL.Error(), http.StatusBadRequest)
		return
	}
	q.runQuery(w, r, client.QueryEndpointsReq{
		Endpoint: endpoint,
	})
}

func (q *query) Policies(w http.ResponseWriter, r *http.Request) {
	endpoints, err := q.getEndpoints(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	unmatched := q.getBool(r, QueryUnmatched)
	if (unmatched && len(endpoints) > 0) || len(endpoints) > 1 {
		http.Error(w, errorPolicyMultiParm.Error(), http.StatusBadRequest)
		return
	}
	var endpoint model.Key
	if len(endpoints) > 0 {
		endpoint = endpoints[0]
	}
	q.runQuery(w, r, client.QueryPoliciesReq{
		Tier:      r.URL.Query().Get(QueryTier),
		Labels:    q.getLabels(r),
		Endpoint:  endpoint,
		Unmatched: unmatched,
		Page:      q.getPage(r),
		Sort:      q.getSort(r),
	})
}

func (q *query) Policy(w http.ResponseWriter, r *http.Request) {
	urlParts := strings.SplitN(r.URL.Path, "/", numURLSegmentsWithName)
	if len(urlParts) != numURLSegmentsWithName {
		http.Error(w, errorInvalidPolicyURL.Error(), http.StatusBadRequest)
		return
	}
	policy, ok := q.getPolicyKeyFromCombinedName(urlParts[numURLSegmentsWithName-1])
	if !ok {
		http.Error(w, errorInvalidPolicyURL.Error(), http.StatusBadRequest)
		return
	}
	q.runQuery(w, r, client.QueryPoliciesReq{
		Policy: policy,
	})
}

func (q *query) Nodes(w http.ResponseWriter, r *http.Request) {
	q.runQuery(w, r, client.QueryNodesReq{
		Page: q.getPage(r),
		Sort: q.getSort(r),
	})
}

func (q *query) Node(w http.ResponseWriter, r *http.Request) {
	urlParts := strings.SplitN(r.URL.Path, "/", numURLSegmentsWithName)
	if len(urlParts) != numURLSegmentsWithName {
		http.Error(w, errorInvalidNodeURL.Error(), http.StatusBadRequest)
		return
	}
	node, ok := q.getNodeKeyFromCombinedName(urlParts[numURLSegmentsWithName-1])
	if !ok {
		http.Error(w, errorInvalidNodeURL.Error(), http.StatusBadRequest)
		return
	}
	q.runQuery(w, r, client.QueryNodesReq{
		Node: node,
	})
}

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
	labels := make(map[string]string, 0)
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
			return nil, errorInvalidEndpointName
		}
		rpols = append(rpols, rpol)
	}
	return rpols, nil
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
			Kind: v3.KindGlobalNetworkPolicy,
			Name: parts[0],
		}, true
	case 2:
		logCxt.Info("Returning NP")
		return model.ResourceKey{
			Kind:      v3.KindNetworkPolicy,
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
			Kind: v3.KindHostEndpoint,
			Name: parts[0],
		}, true
	case 2:
		return model.ResourceKey{
			Kind:      v3.KindWorkloadEndpoint,
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
		Kind: v3.KindNode,
		Name: parts[0],
	}, true
}

func (q *query) runQuery(w http.ResponseWriter, r *http.Request, req interface{}) {
	resp, err := q.qi.RunQuery(context.Background(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	js, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
	w.Write([]byte{'\n'})
}

func (q *query) getPage(r *http.Request) *client.Page {
	if r.URL.Query().Get(QueryPageNum) == AllResults {
		return nil
	}
	return &client.Page{
		PageNum:    q.getInt(r, QueryPageNum, 0),
		NumPerPage: q.getInt(r, QueryNumPerPage, resultsPerPage),
	}
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
