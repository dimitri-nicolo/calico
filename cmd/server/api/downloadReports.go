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

	q := request.URL.Query()

	log.Info(request.URL, q)

	f := q.Get("downloadFormats")

	//ensure at least one format type was specified
	if len(f) < 1 {
		response.WriteHeader(400)
		response.Write([]byte("Missing downloadFormats parameter"))
		return
	}

	fmts := strings.Split(f, ",")

	//ensure all formats requested are valid
	if !areValidFormats(fmts) {
		response.WriteHeader(400)
		response.Write([]byte("Invalid format"))
		return
	}

	if len(fmts) == 1 {
		//single file
		log.Info("single", fmts[0])
		serveSingleFile(response, request, string(fmts[0]))
	} else {
		//multiple files via zip file
		log.Info("multi", len(fmts), fmts)
		serveZipFile(response, request, fmts)
	}
}

const dateformat = "2006-01-02"

func serveZipFile(response http.ResponseWriter, request *http.Request, formats []string) {
	log.Info("SERVE MULTI FILE")

	//TODO: passing the url but what should this really be... something to find this report in ES
	data := getReportData(request.URL.String())

	//TODO: now just generating a fake file
	fakeFormat := ".zip"
	contentType := "application/zip"
	startDate := time.Now()
	endDate := time.Now()
	reportType := "reporttype"
	reportName := "reportName"

	fileName := generateFileName(reportType, reportName, startDate, endDate, fakeFormat)

	log.Info("serve report zip", fileName)

	//set the response header and send the file
	response.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
	response.Header().Set("Content-Type", contentType)
	http.ServeContent(response, request, fileName, time.Now(), bytes.NewReader(data))
}

func serveSingleFile(response http.ResponseWriter, request *http.Request, format string) {
	log.Info("SERVE SINGLE FILE")

	//TODO: passing the url but what should this really be... something to find this report in ES
	data := getReportData(request.URL.String())

	//TODO: now just generating a fake file
	fakeFormat := format + ".txt"
	contentType := "text/plain"
	startDate := time.Now()
	endDate := time.Now()
	reportType := "reporttype"
	reportName := "reportName"

	fileName := generateFileName(reportType, reportName, startDate, endDate, fakeFormat)

	log.Info("serve report", fileName)

	//set the response header and send the file
	response.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
	response.Header().Set("Content-Type", contentType)
	http.ServeContent(response, request, fileName, time.Now(), bytes.NewReader(data))

}

//generates our standard report download file name
func generateFileName(reportType, reportName string, startDate, endDate time.Time, formatWithExtension string) string {
	startDateStr := startDate.Format(dateformat)
	endDateStr := endDate.Format(dateformat)
	var fileName string
	if strings.HasPrefix(formatWithExtension, ".") {
		fileName = fmt.Sprintf("%s-%s-%s-%s%s", reportType, reportName, startDateStr, endDateStr, formatWithExtension)
	} else {
		fileName = fmt.Sprintf("%s-%s-%s-%s-%s", reportType, reportName, startDateStr, endDateStr, formatWithExtension)
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
	//TODO: got get the actual data from elastic for now random file data
	d := make([]byte, 4096)
	rand.Read(d)
	return d
}
