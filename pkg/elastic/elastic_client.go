// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package elastic

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/olivere/elastic/v7"
	log "github.com/sirupsen/logrus"
	"github.com/tigera/deep-packet-inspection/pkg/config"
)

func NewElasticClient(cfg *config.Config) (*elastic.Client, error) {
	u := &url.URL{
		Scheme: cfg.ElasticScheme,
		Host:   fmt.Sprintf("%s:%s", cfg.ElasticHost, cfg.ElasticPort),
	}

	ca, err := x509.SystemCertPool()
	if err != nil {
		log.Fatal(err)
	}
	if cfg.ElasticCA != "" {
		cert, err := ioutil.ReadFile(cfg.ElasticCA)
		if err != nil {
			log.Fatal(err)
		}
		ok := ca.AppendCertsFromPEM(cert)
		if !ok {
			log.Fatal("failed to add CA")
		}
	}

	h := &http.Client{}
	if cfg.ElasticScheme == "https" {
		h.Transport = &http.Transport{TLSClientConfig: &tls.Config{
			RootCAs:            ca,
			InsecureSkipVerify: cfg.ElasticInsecureSkipVerify,
		}}
	}

	options := []elastic.ClientOptionFunc{
		elastic.SetURL(u.String()),
		elastic.SetHttpClient(h),
		elastic.SetErrorLog(log.StandardLogger()),
		elastic.SetSniff(false),
		elastic.SetHealthcheck(false),
		elastic.SetBasicAuth(cfg.ElasticUsername, cfg.ElasticPassword),
	}

	// Enable debug (trace-level) logging for the Elastic client library we use
	if cfg.ElasticDebug {
		options = append(options, elastic.SetTraceLog(log.StandardLogger()))
	}

	log.Debugf("Elastic options %#v", options)
	var esCli *elastic.Client
	ConnectionRetries := cfg.ElasticConnectionRetries
	for i := 0; i < ConnectionRetries; i++ {
		log.Info("Connecting to elastic")
		esCli, err = elastic.NewClient(options...)
		if err == nil {
			log.Info("Connected to elastic")
			break
		}
		log.WithError(err).WithField("attempts", ConnectionRetries-i).Warning("Elastic connect failed, retrying")
		time.Sleep(cfg.ElasticConnectionRetryInterval)
	}
	return esCli, err
}
