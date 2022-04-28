// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package aggregation

import (
	"context"
	"net/http"

	"github.com/olivere/elastic/v7"
	log "github.com/sirupsen/logrus"

	"k8s.io/apiserver/pkg/endpoints/request"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/calico/lma/pkg/auth"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
	"github.com/projectcalico/calico/lma/pkg/httputils"
	"github.com/projectcalico/calico/lma/pkg/k8s"
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
	user, ok := request.UserFrom(ctx)
	if !ok {
		// There should be user info on the request context. If not this is is server error since an earlier handler
		// should have authenticated.
		log.Debug("No user information on request")
		return nil, &httputils.HttpStatusError{
			Status: http.StatusInternalServerError,
			Msg:    "No user information on request",
		}
	}
	return auth.PerformUserAuthorizationReviewForElasticLogs(
		ctx, r.clientSetFactory, user, rd.AggregationRequest.Cluster,
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
