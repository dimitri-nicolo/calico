package api

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	comp "github.com/projectcalico/libcalico-go/lib/compliance"
	"github.com/projectcalico/libcalico-go/lib/options"
	log "github.com/sirupsen/logrus"
	"github.com/tigera/compliance/pkg/datastore"
	"github.com/tigera/compliance/pkg/report"
)

// HandleDownloadReports sends one or multiple (via zip) reports to the client
func HandleDownloadReports(response http.ResponseWriter, request *http.Request) {

	rd := reportDownloader{
		response: response,
		request:  request,
	}

	//make sure at least one valid format was requested
	if err := rd.setFormats(); err != nil {
		log.WithError(err)
		http.Error(rd.response, err.Error(), http.StatusBadRequest)
		return
	}

	//single file or zip based on number of formats requested
	if len(rd.formats) == 1 {
		rd.prepSingleFile()
	} else {
		//TODO: implement multi-file zip
		log.Info("multi report zip, not currently available")
		http.Error(rd.response, "Multi format download not available yet", http.StatusServiceUnavailable)
		return
	}

	rd.serveContent()
}

type reportDownloader struct {
	response http.ResponseWriter
	request  *http.Request
	reportId string //something like:  "Report0::2019-04-11T16:39:11-07:00::2019-04-11T16:39:11-07:00"
	formats  []string
	content  downloadContent
}

type downloadContent struct {
	outputFormat string
	contentType  string
	startDate    metav1.Time
	endDate      metav1.Time
	reportType   string
	reportName   string
	data         []byte
}

//parse and validate the formats passed in the request
func (rd *reportDownloader) setFormats() error {
	q := rd.request.URL.Query()

	log.Info(rd.request.URL, q)

	f := q.Get("downloadFormats")

	//ensure at least one format type was specified
	if len(f) < 1 {
		return fmt.Errorf("Missing download formats")
	}

	rd.formats = strings.Split(f, ",")

	//ensure all formats requested are valid
	if !areValidFormats(rd.formats) {
		return fmt.Errorf("Invalid format")
	}

	return nil
}

//fetch and prepare the content for single file
func (rd *reportDownloader) prepSingleFile() {
	log.Info("serve single report")

	reportId := "Report0::2019-04-11T16:39:11-07:00::2019-04-11T16:39:11-07:00" //TODO:get from param

	report, e := rep.RetrieveArchivedReport(reportId)
	if e != nil {
		errString := "RetrieveArchivedReport failed"
		http.Error(rd.response, errString, http.StatusInternalServerError)
		log.WithError(e).Error(errString)
		return
	}

	content := downloadContent{
		contentType: "text/plain",
		startDate:   report.StartTime,
		endDate:     report.EndTime,
		reportName:  report.ReportName,
		reportType:  report.ReportSpec.ReportType,
	}
	report.UID()

	//TODO:
	//		outputFormat: rd.formats[0] + ".txt",

	content.data, e = rd.renderReportData(report)
	if e != nil {
		errString := "report rendering failed failed"
		http.Error(rd.response, errString, http.StatusInternalServerError)
		log.WithError(e).Error(errString)
		return
	}

	rd.content = content
}

//download eeport or zip content
func (rd *reportDownloader) serveContent() {
	fileName := rd.content.generateFileName()

	//set the response header and send the file
	rd.response.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
	rd.response.Header().Set("Content-Type", rd.content.contentType)
	http.ServeContent(rd.response, rd.request, fileName, time.Now(), bytes.NewReader(rd.content.data))
}

//generates our report download file name
func (rc *downloadContent) generateFileName() string {
	startDateStr := rc.startDate.Format(time.RFC3339)
	endDateStr := rc.endDate.Format(time.RFC3339)
	var fileName string
	if strings.HasPrefix(rc.outputFormat, ".") {
		fileName = fmt.Sprintf("%s-%s-%s-%s%s", rc.reportType, rc.reportName, startDateStr, endDateStr, rc.outputFormat)
	} else {
		fileName = fmt.Sprintf("%s-%s-%s-%s-%s", rc.reportType, rc.reportName, startDateStr, endDateStr, rc.outputFormat)
	}
	return fileName
}

//returns true if all formats passed are valid
func areValidFormats(fmts []string) bool {
	//TODO: check for valid formats
	return true
}

//renders the report data
func (rd *reportDownloader) renderReportData(report *report.ArchivedReportData) ([]byte, error) {

	//get a handle to the calico client
	calico := datastore.MustGetCalicoClient()

	//get the global report type from calico
	grt, e := calico.GlobalReportTypes().Get(context.Background(), report.ReportSpec.ReportType, options.GetOptions{})
	if e != nil {
		errString := "global report type look up failed"
		http.Error(rd.response, errString, http.StatusInternalServerError)
		log.WithError(e).Error(errString)
		return nil, e
	}

	//create the formated report  from the global report type and the report data
	log.Info("ReportTemplate", grt.Spec.UICompleteTemplate.Template)
	log.Info("ReportData", *report.ReportData)

	renderedReport, e := comp.RenderTemplate(grt.Spec.UICompleteTemplate.Template, report.ReportData)
	if e != nil {
		errString := "report generation failed"
		http.Error(rd.response, errString, http.StatusInternalServerError)
		log.WithError(e).Error(errString)
		return nil, e
	}

	return []byte(renderedReport), nil
}
