package elastic

import (
	"context"
	"encoding/json"

	log "github.com/sirupsen/logrus"

	"github.com/tigera/compliance/pkg/report"
)

func (c *client) GetRawComplianceReports() ([]*report.RawComplianceReport, error) {
	reports := []*report.RawComplianceReport{}

	// Query for raw report data in a paginated fashion
	exit := false
	for i := 0; !exit; i += pageSize {
		// Make search query
		res, err := c.Search().
			Index(reportsIndex).
			Sort("startTime", false).
			From(i).Size(pageSize).
			Do(context.Background())
		if err != nil {
			log.WithError(err).Error("failed to query for raw report data")
			return nil, err
		}
		log.WithField("latency (ms)", res.TookInMillis).Debug("query success")

		// define function that pushes the search results into the channel.
		for _, hit := range res.Hits.Hits {
			report := new(report.RawComplianceReport)
			if err := json.Unmarshal(*hit.Source, report); err != nil {
				log.WithFields(log.Fields{"index": hit.Index, "id": hit.Id}).WithError(err).Warn("failed to unmarshal event json")
				continue
			}
			reports = append(reports, report)
		}

		exit = i+pageSize > int(res.Hits.TotalHits)
	}
	return reports, nil
}
