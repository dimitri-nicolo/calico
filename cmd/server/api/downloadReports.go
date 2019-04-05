package api

import (
	"net/http"

	log "github.com/sirupsen/logrus"
)

func HandleDownloadReports(http.ResponseWriter, *http.Request) {
	log.Info("DownloadReports")

}
