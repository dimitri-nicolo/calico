package api

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"net/http"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

func HandleDownloadReports(response http.ResponseWriter, request *http.Request) {

	rd := reportDownloader{
		response: response,
		request:  request,
	}

	if err := rd.setFormats(); err != nil {
		return
	}

	if len(rd.formats) == 1 {
		rd.prepSingleFile()
	} else {
		rd.prepZipFile()
	}

	rd.serveContent()
}

const dateformat = "20060102"

type reportDownloader struct {
	response http.ResponseWriter
	request  *http.Request
	formats  []string
	content  reportContent
}

type reportContent struct {
	outputFormat string
	contentType  string
	startDate    time.Time
	endDate      time.Time
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
		rd.response.WriteHeader(400)
		rd.response.Write([]byte("Missing downloadFormats parameter"))
		return fmt.Errorf("Missing download formats")
	}

	rd.formats = strings.Split(f, ",")

	//ensure all formats requested are valid
	if !areValidFormats(rd.formats) {
		rd.response.WriteHeader(400)
		rd.response.Write([]byte("Invalid format"))
		return fmt.Errorf("Invalid format")
	}

	return nil
}

//fetch and prepare the content for single file
func (rd *reportDownloader) prepSingleFile() {
	log.Info("serve single report")

	//TODO: now just generating a fake file
	content := reportContent{
		outputFormat: rd.formats[0] + ".txt",
		contentType:  "text/plain",
		startDate:    time.Now(),
		endDate:      time.Now(),
		reportType:   "reporttype",
		reportName:   "reportName",
	}

	//TODO: passing the url but what should this really be... something to find this report in ES
	content.data = getReportData(rd.request.URL.String())

	rd.content = content
}

//fetch and prepare the content for multifile zip
func (rd *reportDownloader) prepZipFile() {
	log.Info("serve multi report zip")

	//TODO: now just generating a fake file
	content := reportContent{
		outputFormat: ".zip",
		contentType:  "application/zip",
		startDate:    time.Now(),
		endDate:      time.Now(),
		reportType:   "reporttype",
		reportName:   "reportName",
	}

	//TODO: get all the format files and zip them together
	//TODO: passing the url but what should this really be... something to find this report in ES
	content.data = getReportData(rd.request.URL.String())

	rd.content = content
}

//serve the report or zip content
func (rd *reportDownloader) serveContent() {
	fileName := rd.content.generateFileName()

	//set the response header and send the file
	rd.response.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
	rd.response.Header().Set("Content-Type", rd.content.contentType)
	http.ServeContent(rd.response, rd.request, fileName, time.Now(), bytes.NewReader(rd.content.data))
}

//generates our report download file name
func (rc *reportContent) generateFileName() string {
	startDateStr := rc.startDate.Format(dateformat)
	endDateStr := rc.endDate.Format(dateformat)
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

//get the data for a report from elastic
func getReportData(someKeyForElastic string) []byte {
	//TODO: go get the actual data from elastic... for now random data
	d := make([]byte, 4096)
	rand.Read(d)
	return d
}
