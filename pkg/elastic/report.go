package elastic

import (
	"context"
	"encoding/json"

	log "github.com/sirupsen/logrus"

	"github.com/olivere/elastic"

	"github.com/projectcalico/libcalico-go/lib/errors"

	"github.com/tigera/compliance/pkg/report"
)

func (c *client) RetrieveArchivedReport(id string) (*report.ArchivedReportData, error) {
	clog := log.WithField("id", id)

	// Execute query.
	res, err := c.Search().
		Index(reportsIndex).
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
		return nil, errors.ErrorResourceDoesNotExist{}
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

func (c *client) RetrieveArchivedReportSummaries() ([]*report.ArchivedReportData, error) {
	reps := []*report.ArchivedReportData{}

	// Query for raw report data in a paginated fashion.
	//TODO(rlb): We will need to add pagination options for the UI.... and also sort options.
	exit := false
	for i := 0; !exit; i += pageSize {
		// Make search query
		//TODO(rlb): Can we use exclusion rather than inclusion for this list if that makes sense?
		res, err := c.Search().
			Index(reportsIndex).
			Sort("startTime", false).
			FetchSourceContext(elastic.NewFetchSourceContext(true).Include(
				"reportName", "reportTypeName", "reportSpec", "reportTypeSpec", "startTime", "endTime",
				"endpointsSummary", "namespacesSummary", "servicesSummary", "uiSummary",
			)).From(i).Size(pageSize).Do(context.Background())
		if err != nil {
			log.WithError(err).Error("failed to query for raw report data")
			return nil, err
		}
		log.WithField("latency (ms)", res.TookInMillis).Debug("query success")

		for _, hit := range res.Hits.Hits {
			rep := new(report.ArchivedReportData)
			if err := json.Unmarshal(*hit.Source, rep); err != nil {
				log.WithFields(log.Fields{"index": hit.Index, "id": hit.Id}).WithError(err).Warn("failed to unmarshal event json")
				continue
			}
			reps = append(reps, rep)
		}

		exit = i+pageSize > int(res.Hits.TotalHits)
	}
	return reps, nil
}

func (c *client) StoreArchivedReport(r *report.ArchivedReportData) error {
	res, err := c.Index().
		Index(reportsIndex).
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
