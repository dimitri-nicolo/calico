// Copyright 2019 Tigera Inc. All rights reserved.
package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/olivere/elastic/v7"
	log "github.com/sirupsen/logrus"
	auditv1 "k8s.io/apiserver/pkg/apis/audit"
)

const (
	auditIndex       = "tigera_secure_ee_audit_kube"
	auditLimit       = 1
	epoch            = "1970-01-01T00:00:00Z" //RFC3339
	esTimestampField = "timestamp"
)

// Get elastic client handler
func ESSetup(cfg *Config) (*elastic.Client, error) {
	ca, err := x509.SystemCertPool()
	if err != nil {
		return nil, err
	}

	h := &http.Client{}
	if cfg.ESScheme == "https" {
		if cfg.ESCA != "" {
			cert, err := ioutil.ReadFile(cfg.ESCA)
			if err != nil {
				return nil, err
			}
			ok := ca.AppendCertsFromPEM(cert)
			if !ok {
				return nil, fmt.Errorf("invalid Elasticsearch CA in environment variable ELASTIC_CA")
			}
		}

		h.Transport = &http.Transport{TLSClientConfig: &tls.Config{RootCAs: ca}}
	}

	var esURL *url.URL
	if cfg.ESURI != "" {
		esURL, err = url.Parse(cfg.ESURI)
		if err != nil {
			return nil, err
		}
	} else {
		esURL = &url.URL{
			Scheme: cfg.ESScheme,
			Host:   fmt.Sprintf("%s:%d", cfg.ESHost, cfg.ESPort),
		}
	}
	options := []elastic.ClientOptionFunc{
		elastic.SetURL(esURL.String()),
		elastic.SetHttpClient(h),
		elastic.SetErrorLog(log.StandardLogger()),
		elastic.SetSniff(false),
	}

	if cfg.ESUser != "" {
		options = append(options, elastic.SetBasicAuth(cfg.ESUser, cfg.ESPassword))
	}

	var c *elastic.Client
	for i := 0; i < cfg.ESConnRetries; i++ {
		log.Info("Connecting to elastic")
		if c, err = elastic.NewClient(options...); err == nil {
			return c, nil
		}
		time.Sleep(cfg.ESConnRetryInterval)
	}
	log.Errorf("Unable to connect to Elastic after %d retries", cfg.ESConnRetries)
	return nil, err
}

// Returns the last available log timestamp for audit log.
// We use thus retrieved timestamp to get the logstream not retrieved so far rather than starting from scratch.
func ESGetStartTime(cfg *Config, client *elastic.Client) (int64, error) {
	var ss *elastic.SearchService

	// We are trying to find the latest timestamp available in ES data.
	// To do this, we get the Top 1 (i.e. first) result from the data
	// sorted in descending order of timestamp using audit index.
	idx := fmt.Sprintf("%s.%s.*", auditIndex, cfg.ESIndexSuffix)
	ss = elastic.NewSearchService(client).Index(idx)

	ss = ss.SortBy(elastic.NewFieldSort(esTimestampField).Desc()) // sort by timestamp in descending order.
	ss = ss.Size(auditLimit)                                      // return the first result

	queries := []elastic.Query{
		elastic.NewRangeQuery(esTimestampField).Lte(time.Now()),
	}
	query := elastic.NewBoolQuery().Filter(queries...)
	result, err := ss.Query(query).Do(context.Background())
	if err != nil {
		log.Error("error querying elastic for the latest audit index data timestamp.")
		return 0, err
	}

	if result.TotalHits() > 0 {
		audit := new(auditv1.Event)
		if err = json.Unmarshal(result.Hits.Hits[0].Source, audit); err != nil {
			log.Error("error reading search result values")
			return 0, err
		}

		// Convert timestamp to number of milliseconds since Epoch.
		epochTime, err := time.Parse(time.RFC3339, epoch)
		if err != nil {
			log.Errorf("error parsing epoch timestamp")
			return 0, err
		}
		dur := audit.StageTimestamp.Sub(epochTime)
		log.Infof("start-time: %v", dur.Milliseconds())

		return dur.Milliseconds(), nil
	}

	// Elasticsearch doesn't have any matching entry at the very first run.
	// don't error out, silently log with 0 value.
	log.Info("Elasticsearch didn't return any result, assuming no logs exist yet.")
	return 0, nil
}
