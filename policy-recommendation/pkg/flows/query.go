// Copyright (c) 2022-2023 Tigera, Inc. All rights reserved.
package flows

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	lapi "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	linseed "github.com/projectcalico/calico/linseed/pkg/client"
	"github.com/projectcalico/calico/lma/pkg/api"
	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"
	"github.com/projectcalico/calico/lma/pkg/timeutils"
)

type PolicyRecommendationQuery interface {
	QueryFlows(params *RecommendationFlowLogQueryParams) ([]*api.Flow, error)
}

type recommendationFlowLogQuery struct {
	ctx       context.Context
	client    linseed.Client
	clusterID string
}

func NewRecommendationFlowLogQuery(ctx context.Context, client linseed.Client, id string) *recommendationFlowLogQuery {
	return &recommendationFlowLogQuery{
		ctx:       ctx,
		client:    client,
		clusterID: id,
	}
}

func (qf *recommendationFlowLogQuery) QueryFlows(params *RecommendationFlowLogQueryParams) ([]*api.Flow, error) {
	clog := log.WithField("cluster", qf.clusterID)

	if params == nil {
		err := fmt.Errorf("invalid flow query parameters")
		clog.WithError(err).WithField("params", params)
		return nil, err
	}

	// Query with the parameters provided
	pager := linseed.NewListPager[lapi.L3Flow](buildPendingPolicyMatchQuery(params))
	flows, err := searchFlows(qf.ctx, qf.client.L3Flows(qf.clusterID).List, pager)
	if err != nil || len(flows) == 0 {
		// If no pending_policies matches are found, query all_policies matches.
		// This is done to ensure that versions that don't support the pending_policies
		// sub-type return matching results.
		// Policy recommendation follows an additive approach to recommending new rules
		// getting a match from all_policies and not pending_policies is acceptable.
		clog.Debug("No pending_policies matches found, querying all_policies matches")
		pager = linseed.NewListPager[lapi.L3Flow](buildAllPolicyMatchQuery(params))
		flows, err = searchFlows(qf.ctx, qf.client.L3Flows(qf.clusterID).List, pager)
		if err != nil {
			return flows, err
		}
	}

	return flows, err
}

// buildAllPolicyMatchQuery returns a linseed L3 flow query that will return a match to
// all_policies flows.
func buildAllPolicyMatchQuery(params *RecommendationFlowLogQueryParams) *lapi.L3FlowParams {
	return buildQuery(params, false)
}

// buildPendingPolicyMatchQuery returns a linseed L3 flow query that will return a match to
// pending_policies flows.
func buildPendingPolicyMatchQuery(params *RecommendationFlowLogQueryParams) *lapi.L3FlowParams {
	return buildQuery(params, true)
}

// buildQuery returns linseed L3 flow query parameters using policy recommendation query parameters.
func buildQuery(params *RecommendationFlowLogQueryParams, matchPending bool) *lapi.L3FlowParams {
	// Parse the start and end times.
	now := time.Now()

	startTime := getDurationAsTimeRelToNow(params.StartTime)
	start, _, err := timeutils.ParseTime(now, &startTime)
	if err != nil {
		log.WithError(err).Warning("Failed to parse start time")
	}

	endTime := getDurationAsTimeRelToNow(params.EndTime)
	end, _, err := timeutils.ParseTime(now, &endTime)
	if err != nil {
		log.WithError(err).Warning("Failed to parse start time")
	}

	fp := lapi.L3FlowParams{}
	fp.TimeRange = &lmav1.TimeRange{}
	if start != nil {
		fp.TimeRange.From = *start
	}
	if end != nil {
		fp.TimeRange.To = *end
	}

	fp.SourceTypes = []lapi.EndpointType{lapi.Network, lapi.NetworkSet, lapi.WEP, lapi.HEP}
	fp.DestinationTypes = []lapi.EndpointType{lapi.Network, lapi.NetworkSet, lapi.WEP, lapi.HEP}
	if params.Namespace != "" {
		fp.NamespaceMatches = []lapi.NamespaceMatch{
			{Type: lapi.MatchTypeAny, Namespaces: []string{params.Namespace}},
		}
	}

	// If the request is only for unprotected flows then return a query that will
	// specifically only pick flows that are allowed by a profile.
	allow := lapi.FlowActionAllow
	var policyMatches []lapi.PolicyMatch
	if params.Unprotected {
		policyMatches = []lapi.PolicyMatch{
			{
				Tier:   "__PROFILE__",
				Action: &allow,
			},
		}
	} else {
		// Otherwise, return any flows that are seen by the default tier
		// or allowed by a profile.
		policyMatches = []lapi.PolicyMatch{
			{
				Tier: "default",
			},
			{
				Tier:   "__PROFILE__",
				Action: &allow,
			},
		}
	}

	if matchPending {
		fp.PendingPolicyMatches = policyMatches
	} else {
		fp.PolicyMatches = policyMatches
	}

	return &fp
}

// getDurationAsTimeRelToNow returns a duration as a string in seconds relative to now.
func getDurationAsTimeRelToNow(d time.Duration) string {
	return fmt.Sprintf("now-%ds", int(d.Seconds()))
}

func searchFlows(ctx context.Context, listFn linseed.ListFunc[lapi.L3Flow], pager linseed.ListPager[lapi.L3Flow]) ([]*api.Flow, error) {
	// Search for the raw data in ES.
	pages, errors := pager.Stream(ctx, listFn)

	flows := []*api.Flow{}
	for page := range pages {
		for _, f := range page.Items {
			flow := api.FromLinseedFlow(f)
			if flow != nil {
				flows = append(flows, flow)
			}
		}
	}

	if err, ok := <-errors; ok {
		log.WithError(err).Warning("Hit error processing flow logs")
		return flows, err
	}

	return flows, nil
}
