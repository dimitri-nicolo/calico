package api

import (
	"encoding/json"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

// HandleListReports returns a json list of the reports available to the client
func HandleListReports(response http.ResponseWriter, request *http.Request) {
	log.Info(request.URL)

	//generate some bogus data
	f := &Format{"some format", "some format description"}
	rl := &ReportList{
		Reports: []Report{
			{"type one", time.Now(), time.Now(), "UI Summary 1 text", []Format{*f, *f}},
			{"type two", time.Now(), time.Now(), "UI Summary 2 text", []Format{*f, *f}},
		},
	}

	//marshal and return
	b, _ := json.Marshal(rl)
	_, e := response.Write(b)
	if e != nil {
		log.WithError(e).Error("http response write failure")
	}
}

type ReportList struct {
	Reports []Report `json:"reports"`
}

type Report struct {
	ReportType      string    `json:"reportType"`
	StartTime       time.Time `json:"startTime"`
	EndTime         time.Time `json:"endTime"`
	UISummary       string    `json:"uiSummary"`
	DownloadFormats []Format  `json:"downloadFormats"`
}

type Format struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}
