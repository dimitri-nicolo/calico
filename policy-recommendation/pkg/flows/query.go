// Copyright (c) 2022 Tigera, Inc. All rights reserved.
package flows

import (
	"encoding/json"
	"fmt"
	"strings"

	"golang.org/x/net/context"

	elastic "github.com/olivere/elastic/v7"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/lma/pkg/api"
)

// queryFlows exists for testing purposes.
type queryFlows struct {
	ctx context.Context
}

func NewQueryFlows(ctx context.Context) *queryFlows {
	return &queryFlows{
		ctx: ctx,
	}
}

func (qf *queryFlows) QueryElasticsearchFlows(ca CompositeAggregator, params *PolicyRecommendationParams) ([]*api.Flow, error) {
	if params == nil {
		err := fmt.Errorf("invalid flow query parameters")
		log.WithError(err).WithField("params", params)
		return nil, err
	}

	query := BuildElasticQuery(params)
	log.Tracef("elastic search query: %s", getFormattedQuery(query))

	flows, err := SearchFlows(qf.ctx, ca, query, params)
	if err != nil {
		return flows, err
	}

	return flows, err
}

// getFormattedQuery returns a formatted version of the query, to be used for logging.
func getFormattedQuery(query elastic.Query) string {
	qbytes, _ := query.Source()
	// convert the query to a JSON string
	queryBytes, err := json.Marshal(qbytes)
	if err != nil {
		log.WithError(err).Debugf("elastic search query: %s", getFormattedQuery(query))
		return ""
	}
	queryStr := string(queryBytes)

	// format the query string for printing
	formattedQuery := strings.ReplaceAll(queryStr, "\n", "")
	formattedQuery = strings.ReplaceAll(formattedQuery, " ", "")

	return formattedQuery
}
