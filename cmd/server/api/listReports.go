package api

import (
	"context"

	"encoding/json"
	"fmt"
	"net/http"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	comp "github.com/projectcalico/libcalico-go/lib/compliance"
	"github.com/projectcalico/libcalico-go/lib/options"
	log "github.com/sirupsen/logrus"
	"github.com/tigera/compliance/pkg/datastore"
)

// HandleListReports returns a json list of the reports available to the client
func HandleListReports(response http.ResponseWriter, request *http.Request) {
	log.Info(request.URL)

	//pull the report summaries from elastic
	reportSummaries, e := rep.RetrieveArchivedReportSummaries()

	if e != nil {
		errString := "RetrieveArchivedReportSummaries failed"
		http.Error(response, errString, http.StatusInternalServerError)
		log.WithError(e).Error(errString)
		return
	}

	//get a handle to the calico client
	calico := datastore.MustGetCalicoClient()

	//our report list to return
	rl := &ReportList{}

	//turn each of the reportSummaries into Report objects
	for _, v := range reportSummaries {
		log.Info("ReportType", v.ReportSpec.ReportType, v.ReportName)

		//get the global report type from calico
		grt, e := calico.GlobalReportTypes().Get(context.Background(), v.ReportSpec.ReportType, options.GetOptions{})
		if e != nil {
			errString := "global report type look up failed"
			http.Error(response, errString, http.StatusInternalServerError)
			log.WithError(e).Error(errString)
			return
		}

		//create the UI summary from the template in the global report type and the report data
		UISummary, e := comp.RenderTemplate(grt.Spec.UISummaryTemplate.Template, v.ReportData)
		if e != nil {
			errString := "ui summary generation failed"
			http.Error(response, errString, http.StatusInternalServerError)
			log.WithError(e).Error(errString)
			return
		}

		//load report formats from download templates in the global report report type
		formats := []Format{}
		for _, dlt := range grt.Spec.DownloadTemplates {
			f := Format{dlt.Name, dlt.Description}
			formats = append(formats, f)
		}

		//build the report download url
		url := fmt.Sprintf("/downloadreport/%s", v.UID())

		//package it up in a report and append to slice
		r := Report{
			ReportId:        v.UID(),
			ReportType:      v.ReportSpec.ReportType,
			StartTime:       v.StartTime,
			EndTime:         v.EndTime,
			UISummary:       UISummary,
			DownloadURL:     url,
			DownloadFormats: formats,
		}
		rl.Reports = append(rl.Reports, r)
	}

	//marshal and return
	b, _ := json.Marshal(rl)
	_, e = response.Write(b)
	if e != nil {
		log.WithError(e).Error("http response write failure")
	}
}

type ReportList struct {
	Reports []Report `json:"reports"`
}

type Report struct {
	ReportId        string      `json:"reportId"`
	ReportType      string      `json:"reportType"`
	StartTime       metav1.Time `json:"startTime"`
	EndTime         metav1.Time `json:"endTime"`
	UISummary       string      `json:"uiSummary"`
	DownloadURL     string      `json:"downloadUrl"`
	DownloadFormats []Format    `json:"downloadFormats"`
}

type Format struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}
