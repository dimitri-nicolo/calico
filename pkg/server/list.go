package server

import (
	"fmt"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/libcalico-go/lib/compliance"
)

// handleListReports returns a json list of the reports available to the client
func (s *server) handleListReports(response http.ResponseWriter, request *http.Request) {
	log.Info(request.URL)

	// Pull the report summaries from elastic
	reportSummaries, err := s.rr.RetrieveArchivedReportSummaries()
	if err != nil {
		errString := fmt.Sprintf("Unable to list reports: %v", err)
		http.Error(response, errString, http.StatusServiceUnavailable)
		log.WithError(err).Error(errString)
		return
	}

	// Our report list to return
	rl := &ReportList{}

	// Create an RBAC helper for determining which reports we should include in the returned list.
	rbac := newReportRbacHelper(s, request)
	if canList, err := rbac.canListReports(); err != nil {
		log.WithError(err).Error("Unable to determine access permissions for request")
		http.Error(response, err.Error(), http.StatusServiceUnavailable)
		return
	} else if !canList {
		log.Warning("Requester has insufficient permissions to list reports")
		http.Error(response, "Insufficient permissions to list reports", http.StatusUnauthorized)
		return
	}

	// Obtain the current set of configured ReportTypes.
	rts, err := s.getReportTypes()
	if err != nil {
		log.WithError(err).Error("Unable to query report types")
		http.Error(response, err.Error(), http.StatusServiceUnavailable)
		return
	}

	// Turn each of the reportSummaries into Report objects that will marshal into a format for the documented API.
	for _, v := range reportSummaries {
		log.Debugf("Processing report. ReportType: %s, Report: %s", v.ReportSpec.ReportType, v.ReportName)

		// If we can view the report then include it in the list.
		if include, err := rbac.canViewReport(v.ReportName, v.ReportTypeName); err != nil {
			log.WithError(err).Error("Unable to determine access permissions for request")
			http.Error(response, err.Error(), http.StatusServiceUnavailable)
			return
		} else if !include {
			log.Debug("Requester has insufficient permissions to view report")
			continue
		}

		// Look up the specific report type if it still exists.
		rt, ok := rts[v.ReportTypeName]
		// ReportType is deleted, use ReportTypeSpec in the ReportData.
		if !ok {
			rt = &v.ReportTypeSpec
			log.Infof("ReportType (%s) deleted from the configuration, using from ReportData", v.ReportTypeName)
		}

		var uiSummary string
		var formats []Format

		// If the ReportType does not exist, we'll still include the report but there will be no download options and
		// no summary.
		// TODO(rlb) See todo above.
		if rt != nil {
			log.Debug("ReportType exists, render report data summary")

			// Create the UI summary from the template in the global report type and the report data. If we are unable
			// to render the data just don't include the summary.
			uiSummary, err = compliance.RenderTemplate(rt.UISummaryTemplate.Template, v.ReportData)
			if err != nil {
				log.WithError(err).Debug("Unable to render report data summary")
			}

			//load report formats from download templates in the global report report type
			for _, dlt := range rt.DownloadTemplates {
				f := Format{
					dlt.Name,
					dlt.Description,
				}
				formats = append(formats, f)
			}
		}

		// Package it up in a report and append to slice.
		r := Report{
			ReportId:        v.UID(),
			ReportType:      v.ReportTypeName,
			StartTime:       v.StartTime,
			EndTime:         v.EndTime,
			UISummary:       uiSummary,
			DownloadURL:     strings.Replace(UrlDownload, QueryReport, v.UID(), 1),
			DownloadFormats: formats,
		}
		rl.Reports = append(rl.Reports, r)
	}

	// Write the response as a JSON encoded blob
	writeJSON(response, rl, false)
}
