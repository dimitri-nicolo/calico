package elastic

import (
	"net/http"
	"net/url"
	"time"

	"github.com/olivere/elastic/v7"
	log "github.com/sirupsen/logrus"
)

const (
	ConnectionRetries       = 10
	ConnectionRetryInterval = "500ms"
)

func NewClient(h *http.Client, url *url.URL, username, password string, debug bool) (*elastic.Client, error) {
	options := []elastic.ClientOptionFunc{
		elastic.SetURL(url.String()),
		elastic.SetHttpClient(h),
		elastic.SetErrorLog(log.StandardLogger()),
		elastic.SetSniff(false),
		elastic.SetHealthcheck(false),
	}
	// Enable debug (trace-level) logging for the Elastic client library we use
	if debug {
		options = append(options, elastic.SetTraceLog(log.StandardLogger()))
	}
	if username != "" {
		options = append(options, elastic.SetBasicAuth(username, password))
	}

	retryInterval, err := time.ParseDuration(ConnectionRetryInterval)
	if err != nil {
		return nil, err
	}

	var esCli *elastic.Client
	for i := 0; i < ConnectionRetries; i++ {
		log.Info("Connecting to elastic")
		esCli, err = elastic.NewClient(options...)
		if err == nil {
			break
		}
		log.WithError(err).WithField("attempts", ConnectionRetries-i).Warning("Elastic connect failed, retrying")
		time.Sleep(retryInterval)
	}
	return esCli, err
}
