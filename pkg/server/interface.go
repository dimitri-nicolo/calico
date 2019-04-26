package server

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	clientv3 "github.com/tigera/calico-k8sapiserver/pkg/client/clientset_generated/clientset/typed/projectcalico/v3"
)

const (
	QueryFormat = "format"
	QueryReport = ":report"
	UrlList     = "/compliance/reports"
	UrlGet      = "/compliance/reports/:report"
	UrlDownload = "/compliance/reports/:report/download"
	UrlVersion  = "/compliance/version"
)

type Server interface {
	Start()
	Stop()
	Wait()
}

type ReportConfigurationGetter interface {
	clientv3.GlobalReportsGetter
	clientv3.GlobalReportTypesGetter
}

type ReportList struct {
	Reports []Report `json:"reports"`
}

type Report struct {
	ReportId        string      `json:"reportId"`
	ReportType      string      `json:"reportType"`
	StartTime       metav1.Time `json:"startTime"`
	EndTime         metav1.Time `json:"endTime"`
	UISummary       interface{} `json:"uiSummary"`
	DownloadURL     string      `json:"downloadUrl"`
	DownloadFormats []Format    `json:"downloadFormats"`
	GenerationTime  metav1.Time `json:"generationTime"`
}

type Format struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type VersionData struct {
	Version   string `json:"version"`
	BuildDate string `json:"buildDate"`
	GitTagRef string `json:"gitTagRef"`
	GitCommit string `json:"gitCommit"`
}
