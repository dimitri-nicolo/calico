// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package elastic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/olivere/elastic/v7"
	log "github.com/sirupsen/logrus"

	cerrors "github.com/projectcalico/calico/libcalico-go/lib/errors"

	"github.com/projectcalico/calico/lma/pkg/api"
)

var (
	reportBuckets       = "report_buckets"
	reportSummaryFields = []string{
		"reportName", "reportTypeName", "reportSpec", "reportTypeSpec", "startTime", "endTime",
		"generationTime", "endpointsSummary", "namespacesSummary", "servicesSummary", "auditSummary",
		"uiSummary",
	}
)

// RetrieveArchivedReport implements the api.ReportRetriever interface
func (c *client) RetrieveArchivedReport(id string) (*api.ArchivedReportData, error) {
	clog := log.WithField("id", id)

	searchIndex := c.ClusterIndex(ReportsIndex, "*")

	// Execute query.
	res, err := c.Search().
		Index(searchIndex).
		Query(elastic.NewTermQuery("_id", id)).
		Size(1). // Only retrieve the first document found.
		Do(context.Background())
	if err != nil {
		clog.WithError(err).Error("failed to execute query")
		return nil, err
	}
	clog.WithField("latency (ms)", res.TookInMillis).Debug("query success")

	// Should only return one document.
	switch len(res.Hits.Hits) {
	case 0:
		clog.Error("no hits found")
		return nil, cerrors.ErrorResourceDoesNotExist{
			Identifier: id,
			Err:        errors.New("no report exists with the requested ID"),
		}
	case 1:
		break
	default:
		clog.WithField("hits", len(res.Hits.Hits)).
			Warn("expected to receive only one hit")
	}

	// Extract list from result.
	hit := res.Hits.Hits[0]
	r := new(api.ArchivedReportData)
	if err = json.Unmarshal(hit.Source, r); err != nil {
		clog.WithError(err).Error("failed to extract report from result")
		return nil, err
	}

	return r, nil

}

// RetrieveArchivedReport implements the api.ReportStorer interface
func (c *client) StoreArchivedReport(r *api.ArchivedReportData) error {
	index := c.ClusterAlias(ReportsIndex)
	reportsTemplate, err := c.IndexTemplate(index, ReportsIndex, reportsMapping, true)
	if err != nil {
		log.WithError(err).Error("failed to build index template")
		return err
	}

	if err := c.ensureIndexExistsWithRetry(ReportsIndex, reportsTemplate, true); err != nil {
		return err
	}
	res, err := c.Index().
		Index(index).
		Id(r.UID()).
		BodyJson(r).
		Do(context.Background())
	if err != nil {
		log.WithError(err).Error("failed to store report")
		return err
	}
	log.WithFields(log.Fields{"id": res.Id, "index": res.Index, "type": res.Type}).
		Info("successfully stored report")
	return nil
}

// RetrieveArchivedReport implements the api.ReportRetriever interface
//TODO(rlb): This is another example of an elastic query that is similar to the PIP composite query and could
//           therefore be put somewhere common. For now though leave as is because we intend to port this to
//           v2.5.x and it's better to minimize churn in this case.
func (c *client) RetrieveArchivedReportTypeAndNames(cxt context.Context, q api.ReportQueryParams) ([]api.ReportTypeAndName, error) {
	var reps []api.ReportTypeAndName

	searchIndex := c.ClusterIndex(ReportsIndex, "*")

	// Construct the base search query.
	baseQuery := c.Search().
		Index(searchIndex).
		Size(0)
	if queries := getReportFilterQueries(q); len(queries) != 0 {
		baseQuery = baseQuery.Query(elastic.NewBoolQuery().Must(queries...))
	}

	// Create a composite aggregation to get all unique combinations of reportTypeName and reportName.
	agg := elastic.NewCompositeAggregation().Sources(
		elastic.NewCompositeAggregationTermsValuesSource("reportTypeName").Field("reportTypeName"),
		elastic.NewCompositeAggregationTermsValuesSource("reportName").Field("reportName"),
	).Size(DefaultPageSize)

	// Iterate until we have all results.
	var resultsAfter map[string]interface{}
	for {
		if resultsAfter != nil {
			// We have a start after key to use - start enumerating from this key.
			log.Debugf("Enumerating after key %+v", resultsAfter)
			agg = agg.AggregateAfter(resultsAfter)
		}

		log.Debug("Issuing search query")
		searchResult, err := baseQuery.Aggregation(reportBuckets, agg).Do(cxt)
		if err != nil {
			// We hit an error, exit. This may be a context done error, but that's fine, pass the error on.
			log.WithError(err).Debugf("Error searching %s", searchIndex)
			return nil, err
		}
		if searchResult.TimedOut {
			// Results indicate a timeout.
			log.Errorf("Elastic query timed out: %s", searchIndex)
			return nil, fmt.Errorf("timed out querying %s", searchIndex)
		}

		// Extract the composite buckets from the response - this contains the set of unique keys.
		rawResults, ok := searchResult.Aggregations.Composite(reportBuckets)
		if !ok {
			// The report buckets is not present - this happens when there are no results.
			log.Debugf("No report buckets in %s query - returning no results", searchIndex)
			return nil, nil
		}

		// Extract the unique pairs of report type and name from the response.
		for _, item := range rawResults.Buckets {
			reportTypeName, ok := item.Key["reportTypeName"].(string)
			if !ok {
				log.Warningf("Error fetching composite results: reportTypeName missing from response - ignore")
				continue
			}
			reportName, ok := item.Key["reportName"].(string)
			if !ok {
				log.Warningf("Error fetching composite results: reportName missing from response - ignore")
				continue
			}

			// Append the unique combination of report name and type to the list.
			reps = append(reps, api.ReportTypeAndName{
				ReportTypeName: reportTypeName,
				ReportName:     reportName,
			})
		}

		// No results left - exit.
		if len(rawResults.Buckets) == 0 || rawResults.AfterKey == nil || len(rawResults.AfterKey) == 0 {
			log.Debugf("Completed processing %s", searchIndex)
			break
		}

		// Carry on enumerating from the after key.
		resultsAfter = rawResults.AfterKey
	}

	return reps, nil
}

// RetrieveArchivedReport implements the api.ReportRetriever interface
func (c *client) RetrieveArchivedReportSummaries(cxt context.Context, q api.ReportQueryParams) (*api.ArchivedReportSummaries, error) {
	reps := make([]*api.ArchivedReportData, 0)

	searchIndex := c.ClusterIndex(ReportsIndex, "*")

	// If MaxPerPage is specified, just use that value when querying elastic, otherwise use the default.
	size := DefaultPageSize
	startIdx := 0
	if q.MaxItems != nil {
		// MaxItems has been set. A value of 0 uses our default page size, but the server code that uses this should
		// default the value to something non-zero.
		if *q.MaxItems != 0 {
			size = *q.MaxItems
		}

		// Calculate the start index.
		startIdx = q.Page * size
	}

	// Construct the base search query.
	base := c.Search().
		Index(searchIndex).
		FetchSourceContext(elastic.NewFetchSourceContext(true).
			Include(reportSummaryFields...)).
		Size(size)
	for _, sb := range q.SortBy {
		base = base.Sort(sb.Field, sb.Ascending)
	}
	if queries := getReportFilterQueries(q); len(queries) != 0 {
		base = base.Query(elastic.NewBoolQuery().Must(queries...))
	}

	// Query for raw report data in a paginated fashion.
	var count int64
	for i := startIdx; ; i += DefaultPageSize {
		// Make search query
		res, err := base.From(i).Do(cxt)
		if err != nil {
			// We hit an error, exit. This may be a context done error, but that's fine, pass the error on.
			log.WithError(err).Error("failed to query for raw report data")
			return nil, err
		}
		log.WithField("latency (ms)", res.TookInMillis).Debug("query success")

		for _, hit := range res.Hits.Hits {
			rep := new(api.ArchivedReportData)
			if err := json.Unmarshal(hit.Source, rep); err != nil {
				log.WithFields(log.Fields{"index": hit.Index, "id": hit.Id}).WithError(err).Warn("failed to unmarshal report summary json")
				continue
			}
			reps = append(reps, rep)
		}
		count = res.Hits.TotalHits.Value

		// Exit if either of the following are true:
		// - MaxPerPage was specified, in this case we queried the exact number we needed, so we'll either have that
		//   or as many as there were remaining.
		// - We have exhausted all entries.
		if q.MaxItems != nil {
			log.Debug("Queried specific number per page")
			break
		}
		if i+DefaultPageSize >= int(res.Hits.TotalHits.Value) {
			log.Debug("Exhausted results")
			break
		}
	}
	return &api.ArchivedReportSummaries{
		Reports: reps,
		Count:   int(count),
	}, nil
}

// RetrieveArchivedReport implements the api.ReportRetriever interface
func (c *client) RetrieveArchivedReportSummary(id string) (*api.ArchivedReportData, error) {
	queries := []elastic.Query{
		elastic.NewTermQuery("_id", id),
	}
	res, err := c.retrieveArchivedReportSummary(queries, false)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, cerrors.ErrorResourceDoesNotExist{
			Identifier: id,
			Err:        errors.New("no report exists with the requested ID"),
		}
	}

	return res, nil
}

// RetrieveArchivedReport implements the api.ReportRetriever interface
func (c *client) RetrieveLastArchivedReportSummary(reportName string) (*api.ArchivedReportData, error) {
	// We are not searching for an exact report. On the off chance we have some rogue data, always put an upper cap
	// on the endTime of now. This is probably paranoid behavior, but best to be safe as we'd stop generating
	// reports until after that point if there was rogue data.
	log.Debugf("Retrieving last archived report summary for: %s", reportName)
	queries := []elastic.Query{
		elastic.NewTermQuery("reportName", reportName),
		//elastic.NewRangeQuery("endTime").From(nil).To(time.Now()),
	}
	res, err := c.retrieveArchivedReportSummary(queries, true)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, cerrors.ErrorResourceDoesNotExist{
			Identifier: reportName,
			Err:        errors.New("there is no archived report with the specified reportName"),
		}
	}
	return res, nil
}

func (c *client) retrieveArchivedReportSummary(queries []elastic.Query, includeReverseSort bool) (*api.ArchivedReportData, error) {
	rep := new(api.ArchivedReportData)

	searchIndex := c.ClusterIndex(ReportsIndex, "*")

	q := c.Search().
		Index(searchIndex).
		Query(elastic.NewBoolQuery().Must(queries...)).
		FetchSourceContext(elastic.NewFetchSourceContext(true).
			Include(reportSummaryFields...)).Size(1)
	if includeReverseSort {
		q = q.Sort("endTime", false)
	}
	res, err := q.Do(context.Background())
	if err != nil {
		log.WithError(err).Error("failed to query for raw report summary data")
		return nil, err
	}
	log.WithField("latency (ms)", res.TookInMillis).Debug("query success")

	// Should only return one document.
	switch len(res.Hits.Hits) {
	case 0:
		log.Debug("no hits found")
		return nil, nil
	case 1:
		log.Debug("result found")
	default:
		log.WithField("hits", len(res.Hits.Hits)).
			Warn("expected to receive only one hit")
	}

	// Extract list from result.
	hit := res.Hits.Hits[0]
	if err := json.Unmarshal(hit.Source, rep); err != nil {
		log.WithFields(log.Fields{"index": hit.Index, "id": hit.Id}).WithError(err).Warn("failed to unmarshal report summary json")
		return nil, err
	}

	return rep, nil
}

// getReportFilterQueries calculates the query for the report summaries. This is also used in the querying of the
// unique sets of report name and type.
func getReportFilterQueries(q api.ReportQueryParams) []elastic.Query {
	queries := []elastic.Query{}

	// If reports were requested, add the filters. The filter depends on which of the report type and name were
	// specified.
	var rqueries []elastic.Query
	for _, r := range q.Reports {
		if r.ReportName != "" && r.ReportTypeName != "" {
			rqueries = append(rqueries, elastic.NewBoolQuery().Must(
				elastic.NewMatchQuery("reportTypeName", r.ReportTypeName),
				elastic.NewMatchQuery("reportName", r.ReportName),
			))
		} else if r.ReportName == "" && r.ReportTypeName != "" {
			rqueries = append(rqueries, elastic.NewMatchQuery("reportTypeName", r.ReportTypeName))
		} else if r.ReportName != "" && r.ReportTypeName == "" {
			rqueries = append(rqueries, elastic.NewMatchQuery("reportName", r.ReportName))
		}
	}
	if len(rqueries) > 0 {
		queries = append(queries, elastic.NewBoolQuery().Should(rqueries...))
	}

	// If a start or end time were specified include a range query. The query depends on which time parameters were
	// actually specified.
	if q.FromTime != "" && q.ToTime != "" {
		queries = append(queries, elastic.NewBoolQuery().Should(
			elastic.NewRangeQuery("startTime").From(q.FromTime).To(q.ToTime),
			elastic.NewRangeQuery("endTime").From(q.FromTime).To(q.ToTime),
		))
	} else if q.FromTime != "" && q.ToTime == "" {
		queries = append(queries, elastic.NewRangeQuery("endTime").From(q.FromTime))
	} else if q.FromTime == "" && q.ToTime != "" {
		queries = append(queries, elastic.NewRangeQuery("startTime").To(q.ToTime))
	}

	return queries
}

// getAnyStringValueQuery calculates the query for a specific string field to match one of the supplied values.
func getAnyStringValueQuery(field string, vals []string) elastic.Query {
	queries := []elastic.Query{}
	for _, val := range vals {
		queries = append(queries, elastic.NewMatchQuery(field, val))
	}
	return elastic.NewBoolQuery().Should(queries...)
}
