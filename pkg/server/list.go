package server

import (
	"fmt"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
)

// handleListReports returns a json list of the reports available to the client
func (s *server) handleListReports(response http.ResponseWriter, request *http.Request) {
	log.Info(request.URL)

	// Create an RBAC helper for determining which reports we should include in the returned list.
	rbac := s.rhf.NewReportRbacHelper(request)

	// First check if the user is able to List reports.
	if canList, err := rbac.CanListReports(); err != nil {
		log.WithError(err).Error("Unable to determine access permissions for request")
		http.Error(response, err.Error(), http.StatusServiceUnavailable)
		return
	} else if !canList {
		log.Debug("Requester has insufficient permissions to list reports")
		http.Error(response, "Access denied", http.StatusUnauthorized)
		return
	}

	// Obtain the current set of configured ReportTypes.
	rts, err := s.getReportTypes()
	if err != nil {
		log.WithError(err).Error("Unable to query report types")
		http.Error(response, err.Error(), http.StatusServiceUnavailable)
		return
	}

	// Pull the report summaries from elastic
	reportSummaries, err := s.rr.RetrieveArchivedReportSummaries()
	if err != nil {
		errString := fmt.Sprintf("Unable to list reports: %v", err)
		http.Error(response, errString, http.StatusServiceUnavailable)
		log.WithError(err).Error(errString)
		return
	}

	// Initialize the report list to return.
	rl := &ReportList{
		Reports: []Report{},
	}

	// Turn each of the reportSummaries into Report objects that will marshal into a format for the documented API.
	for _, v := range reportSummaries {
		log.Debugf("Processing report. ReportType: %s, Report: %s", v.ReportTypeName, v.ReportName)

		// If user can list the report then include it in the list.
		if include, err := rbac.CanViewReportSummary(v.ReportName); err != nil {
			log.WithError(err).Error("Unable to determine access permissions for request")
			http.Error(response, err.Error(), http.StatusServiceUnavailable)
			return
		} else if !include {
			log.Debug("Requester has insufficient permissions to view report")
			continue
		}

		// Look up the specific report type if it still exists.
		rt := rts[v.ReportTypeName]
		// ReportType is deleted, use ReportTypeSpec in the ReportData.
		if rt == nil {
			// If the report type has been deleted just use the one stored in the ReportData.
			log.Infof("ReportType (%s) deleted from the configuration, using from ReportData", v.ReportTypeName)
			rt = &v.ReportTypeSpec
		}

		var formats []Format
		var downloadUrl = ""

		// If the user can view the report then include the download url and formats
		if include, err := rbac.CanViewReport(v.ReportTypeName, v.ReportName); err != nil {
			log.WithError(err).Error("Unable to determine access permissions for request")
			http.Error(response, err.Error(), http.StatusServiceUnavailable)
			return
		} else if include {
			//build the download url
			downloadUrl = strings.Replace(UrlDownload, QueryReport, v.UID(), 1)

			//load report formats from download templates in the global report report type
			for _, dlt := range rt.DownloadTemplates {
				log.Debugf("Including download format: %s", dlt.Name)
				f := Format{
					dlt.Name,
					dlt.Description,
				}
				formats = append(formats, f)
			}
		}

		// Package it up in a report and append to slice.
		r := Report{
			Id:              v.UID(),
			Name:            v.ReportName,
			Type:            v.ReportTypeName,
			StartTime:       v.StartTime,
			EndTime:         v.EndTime,
			UISummary:       v.UISummary,
			DownloadURL:     downloadUrl,
			DownloadFormats: formats,
			GenerationTime:  v.GenerationTime,
		}
		rl.Reports = append(rl.Reports, r)
	}

	// Write the response as a JSON encoded blob
	writeJSON(response, rl, false)
}
