package pip

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/tigera/compliance/pkg/list"

	pipcfg "github.com/tigera/es-proxy/pkg/pip/config"
	pelastic "github.com/tigera/lma/pkg/elastic"
)

// New returns a new PIP instance.
func New(cfg *pipcfg.Config, listSrc list.Source, es pelastic.Client) PIP {
	p := &pip{
		listSrc:  listSrc,
		esClient: es,
		cfg:      cfg,
	}
	return p
}

// pip implements the PIP interface.
type pip struct {
	listSrc  list.Source
	esClient pelastic.Client
	cfg      *pipcfg.Config
}

type FlowLogResults struct {
	pelastic.CompositeAggregationResults `json:",inline"`
	AggregationsPreview                  map[string]interface{} `json:"aggregations_preview"`
}

// GetFlows returns the set of PIP-processed flows based on the request parameters in `params`. The map is
// JSON serializable
func (p *pip) GetFlows(ctxIn context.Context, params *PolicyImpactParams) (*FlowLogResults, error) {
	// Create a context with timeout to ensure we don't block for too long with this calculation.
	ctxWithTimeout, cancel := context.WithTimeout(ctxIn, p.cfg.MaxCalculationTime)
	defer cancel() // Releases timer resources if the operation completes before the timeout.

	// Get a primed policy calculator.
	calc, err := p.GetPolicyCalculator(ctxWithTimeout, params)
	if err != nil {
		return nil, err
	}

	// Construct the query.
	// TODO(rlb): This should be fully parsed from the HTTP request.
	q := &pelastic.CompositeAggregationQuery{
		Name:          FlowlogBuckets,
		DocumentIndex: params.DocumentIndex,
		Query:         params.Query,
		AggCompositeSourceInfos: UICompositeSources,
		AggNestedTermInfos:      AggregatedTerms,
		AggSumInfos:             UIAggregationSums,
	}

	// Enumerate the aggregation buckets until we have all we need. The channel will be automatically closed.
	var before []*pelastic.CompositeAggregationBucket
	var after []*pelastic.CompositeAggregationBucket
	startTime := time.Now()
	buckets, errs := p.SearchAndProcessFlowLogs(ctxWithTimeout, q, nil, calc, params.Limit)
	for bucket := range buckets {
		before = append(before, bucket.Before...)
		after = append(after, bucket.After...)
	}
	took := int64(time.Now().Sub(startTime) / time.Millisecond)

	// Check for errors.
	// We can use the blocking version of the channel operator since the error channel will have been closed (it
	// is closed alongside the results channel).
	err = <-errs

	// If there was an error, check for a time out. If it timed out just flag this in the response, but return whatever
	// data we already have. Otherwise return the error.
	// For timeouts we have a couple of mechanisms for hitting this:
	// -  The elastic search query returns a timeout.
	// -  We exceed the context deadline.
	var timedOut bool
	if err != nil {
		if _, ok := err.(pelastic.TimedOutError); ok {
			// Response from ES indicates a handled timeout.
			log.Info("Response from ES indicates time out - flag results as timedout")
			timedOut = true
		} else if ctxIn.Err() == nil && ctxWithTimeout.Err() == context.DeadlineExceeded {
			// The context passed to us has no error, but our context with timeout is indicating it has timed out.
			// We need to check the context error rather than checking the returned error since elastic wraps the
			// original context error.
			log.Info("Context deadline exceeded - flag results as timedout")
			timedOut = true
		} else {
			// Just pass the received error up the stack.
			log.WithError(err).Warning("Error response from elasticsearch query")
			return nil, err
		}
	}

	return &FlowLogResults{
		CompositeAggregationResults: pelastic.CompositeAggregationResults{
			TimedOut:     timedOut,
			Took:         took,
			Aggregations: pelastic.CompositeAggregationBucketsToMap(before, q),
		},
		AggregationsPreview: pelastic.CompositeAggregationBucketsToMap(after, q),
	}, nil
}
