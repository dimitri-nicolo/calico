package elastic

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/olivere/elastic"
	log "github.com/sirupsen/logrus"

	cerrors "github.com/projectcalico/libcalico-go/lib/errors"

	"github.com/tigera/compliance/pkg/report"
)

var (
	reportSummaryFields = []string{
		"reportName", "reportTypeName", "reportSpec", "reportTypeSpec", "startTime", "endTime",
		"generationTime", "endpointsSummary", "namespacesSummary", "servicesSummary", "auditSummary",
		"uiSummary",
	}
)

func (c *client) RetrieveArchivedReport(id string) (*report.ArchivedReportData, error) {
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
	r := new(report.ArchivedReportData)
	if err = json.Unmarshal(*hit.Source, r); err != nil {
		clog.WithError(err).Error("failed to extract report from result")
		return nil, err
	}

	return r, nil

}

func (c *client) StoreArchivedReport(r *report.ArchivedReportData, t time.Time) error {
	index := c.ClusterIndex(ReportsIndex, t.Format(IndexTimeFormat))
	if err := c.ensureIndexExistsWithRetry(index, reportsMapping); err != nil {
		return err
	}
	res, err := c.Index().
		Index(index).
		Type("_doc").
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

func (c *client) RetrieveArchivedReportSummaries() ([]*report.ArchivedReportData, error) {
	reps := []*report.ArchivedReportData{}

	searchIndex := c.ClusterIndex(ReportsIndex, "*")
	// Query for raw report data in a paginated fashion.
	//TODO(rlb): We will need to add pagination options for the UI.... and also sort options.
	exit := false
	for i := 0; !exit; i += pageSize {
		// Make search query
		res, err := c.Search().
			Index(searchIndex).
			Sort("startTime", false).
			FetchSourceContext(elastic.NewFetchSourceContext(true).Include(reportSummaryFields...)).From(i).Size(pageSize).Do(context.Background())
		if err != nil {
			log.WithError(err).Error("failed to query for raw report data")
			return nil, err
		}
		log.WithField("latency (ms)", res.TookInMillis).Debug("query success")

		for _, hit := range res.Hits.Hits {
			rep := new(report.ArchivedReportData)
			if err := json.Unmarshal(*hit.Source, rep); err != nil {
				log.WithFields(log.Fields{"index": hit.Index, "id": hit.Id}).WithError(err).Warn("failed to unmarshal report summary json")
				continue
			}
			reps = append(reps, rep)
		}

		exit = i+pageSize > int(res.Hits.TotalHits)
	}
	return reps, nil
}

func (c *client) RetrieveArchivedReportSummary(id string) (*report.ArchivedReportData, error) {
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

func (c *client) RetrieveLastArchivedReportSummary(reportName string) (*report.ArchivedReportData, error) {
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

func (c *client) retrieveArchivedReportSummary(queries []elastic.Query, includeReverseSort bool) (*report.ArchivedReportData, error) {
	rep := new(report.ArchivedReportData)

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
		break
	default:
		log.WithField("hits", len(res.Hits.Hits)).
			Warn("expected to receive only one hit")
	}

	// Extract list from result.
	hit := res.Hits.Hits[0]
	if err := json.Unmarshal(*hit.Source, rep); err != nil {
		log.WithFields(log.Fields{"index": hit.Index, "id": hit.Id}).WithError(err).Warn("failed to unmarshal report summary json")
		return nil, err
	}

	return rep, nil
}
