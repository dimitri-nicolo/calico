// Copyright (c) 2018 Tigera, Inc. All rights reserved.
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/tigera/calicoq/web/pkg/querycache/client"
)

const (
	queryLabelPrefix      = "label_"
	querySelector         = "selector"
	queryPolicy    = "policy"
	queryRuleDirection    = "ruleDirection"
	queryRuleIndex        = "ruleIndex"
	queryRuleEntity       = "ruleEntity"
	queryRuleNegatedSelector = "ruleNegatedSelector"
	queryPageNum          = "page"
	queryNumPerPage       = "maxItems"
	queryEndpoint     = "endpoint"
	queryUnmatched        = "unmatched"
	queryTier             = "tier"

	allResults     = "all"
	resultsPerPage = 100
)

var (
	errorPolicyMultiParm = errors.New("specify only one of " + queryEndpoint +
		" or " + queryUnmatched)
	errorEndpointMultiParm = errors.New("specify only one of " + querySelector +
		" or " + queryPolicy)
	errorInvalidWEPName = errors.New("the workload endpoint name is not valid; it should be of the format " +
		"<namespace>/<workload endpoint name>")
	errorInvalidPolicyName = errors.New("the policy name is not valid; it should be of the format " +
		"<namespace>/<workload endpoint name>")
)

type Query interface {
	Policies(w http.ResponseWriter, r *http.Request)
	Nodes(w http.ResponseWriter, r *http.Request)
	Endpoints(w http.ResponseWriter, r *http.Request)
	Summary(w http.ResponseWriter, r *http.Request)
}

func NewQuery(qi client.QueryInterface) Query {
	return &query{qi: qi}
}

type query struct {
	qi client.QueryInterface
}

func (q *query) Endpoints(w http.ResponseWriter, r *http.Request) {
	selector := r.URL.Query().Get(querySelector)
	policies, err := q.getPolicies(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if (selector != "" && len(policies) > 0) || len(policies) > 1 {
		http.Error(w, errorEndpointMultiParm.Error(), http.StatusBadRequest)
		return
	}
	var policy model.Key
	if len(policies) > 0 {
		policy = policies[0]
	}
	endpoints, err := q.getEndpoints(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	q.runQuery(w, r, client.QueryEndpointsReq{
		Selector: selector,
		Endpoints: endpoints,
		Policy: policy,
		RuleDirection: r.URL.Query().Get(queryRuleDirection),
		RuleIndex: q.getInt(r, queryRuleIndex, 0),
		RuleEntity: r.URL.Query().Get(queryRuleEntity),
		RuleNegatedSelector: q.getBool(r, queryRuleNegatedSelector),
		Page: q.getPage(r),
	})
}

func (q *query) Policies(w http.ResponseWriter, r *http.Request) {
	endpoints, err := q.getEndpoints(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	unmatched := q.getBool(r, queryUnmatched)
	if (unmatched && len(endpoints) > 0) || len(endpoints) > 1 {
		http.Error(w, errorPolicyMultiParm.Error(), http.StatusBadRequest)
		return
	}
	var endpoint model.Key
	if len(endpoints) > 0 {
		endpoint = endpoints[0]
	}
	policies, err := q.getPolicies(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	q.runQuery(w, r, client.QueryPoliciesReq{
		Tier:      r.URL.Query().Get(queryTier),
		Labels:    q.getLabels(r),
		Endpoint:  endpoint,
		Policies:  policies,
		Unmatched: unmatched,
		Page: q.getPage(r),
	})
}

func (q *query) Summary(w http.ResponseWriter, r *http.Request) {
	q.runQuery(w, r, client.QueryClusterReq{})
}

func (q *query) Nodes(w http.ResponseWriter, r *http.Request) {
	q.runQuery(w, r, client.QueryNodesReq{
		Page: q.getPage(r),
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
		if strings.HasPrefix(k, queryLabelPrefix) {
			labels[strings.TrimPrefix(k, queryLabelPrefix)] = pvs[0]
		}
	}
	return labels
}

func (q *query) getEndpoints(r *http.Request) ([]model.Key, error) {
	eps := r.URL.Query()[queryEndpoint]
	reps := make([]model.Key, 0, len(eps))
	for _, ep := range eps {
		parts := strings.Split(ep, "/")
		if len(parts) > 2 {
			return nil, errorInvalidWEPName
		} else if len(parts) == 1 {
			reps = append(reps, model.ResourceKey{
				Kind: v3.KindHostEndpoint,
				Name: parts[0],
			})
		}
		reps = append(reps, model.ResourceKey{
			Kind:      v3.KindWorkloadEndpoint,
			Namespace: parts[0],
			Name:      parts[1],
		})
	}
	return reps, nil
}

func (q *query) getPolicies(r *http.Request) ([]model.Key, error) {
	pols := r.URL.Query()[queryPolicy]
	rpols := make([]model.Key, 0, len(pols))
	for _, pol := range pols {
		parts := strings.Split(pol, "/")
		if len(parts) > 2 {
			return nil, errorInvalidPolicyName
		} else if len(parts) == 1 {
			rpols = append(rpols, model.ResourceKey{
				Kind: v3.KindGlobalNetworkPolicy,
				Name: parts[0],
			})
		}
		rpols = append(rpols,  model.ResourceKey{
			Kind:      v3.KindNetworkPolicy,
			Namespace: parts[0],
			Name:      parts[1],
		})
	}
	return rpols, nil
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
	if r.URL.Query().Get(queryPageNum) == allResults {
		return nil
	}
	return &client.Page{
		PageNum:    q.getInt(r, queryPageNum, 0),
		NumPerPage: q.getInt(r, queryNumPerPage, resultsPerPage),
	}
}
