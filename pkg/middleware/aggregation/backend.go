// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package aggregation

import (
	"context"

	log "github.com/sirupsen/logrus"

	"github.com/olivere/elastic/v7"

	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"

	"github.com/tigera/lma/pkg/auth"
	lmaelastic "github.com/tigera/lma/pkg/elastic"
	"github.com/tigera/lma/pkg/k8s"
)

// Sanity check the realAggregationBackend satisfies the ServiceGraphBackend interface.
var _ AggregationBackend = &realAggregationBackend{}

// AggregationBackend provides the backend function for the aggregation queries.
type AggregationBackend interface {
	PerformUserAuthorizationReview(ctx context.Context, rd *RequestData) ([]v3.AuthorizedResourceVerbs, error)
	RunQuery(cxt context.Context, rd *RequestData, query elastic.Query) (elastic.Aggregations, error)
}

// realAggregationBackend implements the real backend for the aggregation queries.
type realAggregationBackend struct {
	elastic          lmaelastic.Client
	clientSetFactory k8s.ClientSetFactory
}

// PerformUserAuthorizationReview performs a user authorization check.
func (r *realAggregationBackend) PerformUserAuthorizationReview(ctx context.Context, rd *RequestData) ([]v3.AuthorizedResourceVerbs, error) {
	// Get the RBAC portion of the query to limit the documents the user can request.
	return auth.PerformUserAuthorizationReviewForElasticLogs(
		ctx, r.clientSetFactory, rd.HTTPRequest, rd.AggregationRequest.Cluster,
	)
}

// RunQuery performs an elasticsearch aggregation query.
func (r *realAggregationBackend) RunQuery(
	ctx context.Context, rd *RequestData, query elastic.Query,
) (elastic.Aggregations, error) {
	log.Debugf("Running aggregation; index=%s, query=%#v, agg=%#v", rd.Index, query, rd.Aggregations)

	search := r.elastic.Backend().
		Search(rd.Index).
		Query(query).
		Size(0)

	for an, av := range rd.Aggregations {
		search = search.Aggregation(an, av)
	}

	res, err := search.Do(ctx)
	if err != nil {
		return nil, err
	}

	return res.Aggregations, nil
}
