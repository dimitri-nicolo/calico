// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package elastic

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/olivere/elastic/v7"
	log "github.com/sirupsen/logrus"

	api "github.com/tigera/lma/pkg/api"
)

const (
	createIndexMaxRetries    = 3
	createIndexRetryInterval = 1 * time.Second
)

type IndexSettings struct {
	Replicas int `json:"number_of_replicas"`
	Shards   int `json:"number_of_shards"`
}

type Client interface {
	api.BenchmarksQuery
	api.BenchmarksStore
	api.BenchmarksGetter
	api.AuditLogReportHandler
	api.FlowLogReportHandler
	api.ReportRetriever
	api.ReportStorer
	api.ListDestination
	api.EventFetcher
	ClusterIndex(string, string) string
	Backend() *elastic.Client

	SearchCompositeAggregations(
		context.Context, *CompositeAggregationQuery, CompositeAggregationKey,
	) (<-chan *CompositeAggregationBucket, <-chan error)

	Do(ctx context.Context, s *elastic.SearchService) (*elastic.SearchResult, error)
}

// client implements the Client interface.
type client struct {
	*elastic.Client
	indexSuffix   string
	indexSettings IndexSettings
}

// doFunc invokes the Do on the search service. This is added to allow us to mock out the client in test code.
func (c *client) Do(ctx context.Context, s *elastic.SearchService) (*elastic.SearchResult, error) {
	return s.Do(ctx)
}

// MustGetElasticClient returns the elastic Client, or panics if it's not possible.
func MustGetElasticClient() Client {
	cfg := MustLoadConfig()
	c, err := NewFromConfig(cfg)
	if err != nil {
		log.Panicf("Unable to connect to Elasticsearch: %v", err)
	}
	return c
}

// NewFromConfig returns a new elastic Client using the supplied configuration.
func NewFromConfig(cfg *Config) (Client, error) {
	ca, err := x509.SystemCertPool()
	if err != nil {
		return nil, err
	}

	h := &http.Client{}
	if cfg.ParsedElasticURL.Scheme == "https" {
		if cfg.ElasticCA != "" {
			cert, err := ioutil.ReadFile(cfg.ElasticCA)
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

	return New(
		h, cfg.ParsedElasticURL, cfg.ElasticUser, cfg.ElasticPassword, cfg.ElasticIndexSuffix,
		cfg.ElasticConnRetries, cfg.ElasticConnRetryInterval, cfg.ParsedLogLevel == log.DebugLevel, cfg.ElasticReplicas,
		cfg.ElasticShards)
}

// New returns a new elastic client using the supplied parameters. This method performs retries if creation of the
// client fails.
func New(
	h *http.Client, url *url.URL, username, password, indexSuffix string,
	retries int, retryInterval time.Duration, trace bool, replicas int, shards int,
) (Client, error) {
	options := []elastic.ClientOptionFunc{
		elastic.SetURL(url.String()),
		elastic.SetHttpClient(h),
		elastic.SetErrorLog(log.StandardLogger()),
		elastic.SetSniff(false),
	}
	if trace {
		options = append(options, elastic.SetTraceLog(log.StandardLogger()))
	}
	if username != "" {
		options = append(options, elastic.SetBasicAuth(username, password))
	}

	var err error
	var c *elastic.Client
	for i := 0; i < retries; i++ {
		log.Info("Connecting to elastic")
		if c, err = elastic.NewClient(options...); err == nil {
			return &client{c, indexSuffix, IndexSettings{replicas, shards}}, nil
		}
		log.WithError(err).WithField("attempts", retries-i).Warning("Elastic connect failed, retrying")
		time.Sleep(retryInterval)
	}
	log.Errorf("Unable to connect to Elastic after %d retries", retries)
	return nil, err
}

func (c *client) ensureIndexExistsWithRetry(index, mapping string) error {
	// If multiple threads attempt to create the index at the same time we can end up with errors during the creation
	// which don't seem to match sensible error codes. Let's just add a retry mechanism and retry the creation a few
	// times.
	var err error
	for i := 0; i < createIndexMaxRetries; i++ {
		if err = c.ensureIndexExists(index, mapping); err == nil {
			break
		}
		time.Sleep(createIndexRetryInterval)
	}

	if err != nil {
		return fmt.Errorf("unable to create index: %v", err)
	}

	return err
}

func (c *client) ensureIndexExists(index, mapping string) error {
	clog := log.WithField("index", index)

	// Check if index exists.
	exists, err := c.IndexExists(index).Do(context.Background())
	if err != nil {
		clog.WithError(err).Info("failed to check if index exists")
		return err
	}

	// Return if index exists
	if exists {
		clog.Info("index already exists")
		return nil
	}

	// Create index.
	clog.Info("index doesn't exist, creating...")
	createIndex, err := c.
		CreateIndex(index).
		BodyJson(map[string]interface{}{
			"mappings": json.RawMessage(mapping),
			"settings": c.indexSettings,
		}).
		Do(context.Background())
	if err != nil {
		if elastic.IsConflict(err) {
			clog.Info("index already exists")
			return nil
		}
		clog.WithError(err).Info("failed to create index")
		return err
	}

	// Check if acknowledged
	if !createIndex.Acknowledged {
		clog.Warn("index creation has not yet been acknowledged")
	}
	clog.Info("index successfully created!")
	return nil
}

func (c *client) ClusterIndex(index, postfix string) string {
	return fmt.Sprintf("%s.%s.%s", index, c.indexSuffix, postfix)
}

func (c *client) Backend() *elastic.Client {
	return c.Client
}

func (c *client) Reset() {
	_, _ = c.Client.DeleteIndex(
		c.ClusterIndex(ReportsIndex, "*"),
		c.ClusterIndex(SnapshotsIndex, "*"),
		c.ClusterIndex(AuditLogIndex, "*"),
		c.ClusterIndex(BenchmarksIndex, "*"),
	).Do(context.Background())
}

// NewMockClient creates a mock client used for testing.
func NewMockClient(doFunc func(ctx context.Context, s *elastic.SearchService) (*elastic.SearchResult, error)) Client {
	mc := mockComplianceClient{}
	mc.DoFunc = doFunc
	return &mc
}

type mockComplianceClient struct {
	Client
	DoFunc func(ctx context.Context, s *elastic.SearchService) (*elastic.SearchResult, error)
}

func (m mockComplianceClient) Backend() *elastic.Client {
	return nil
}

func (m mockComplianceClient) ClusterIndex(string, string) string {
	return "fake-index"
}

func (m mockComplianceClient) Do(ctx context.Context, s *elastic.SearchService) (*elastic.SearchResult, error) {
	return m.DoFunc(ctx, s)
}

// NewMockSearchClient creates a mock client used for testing search results.
func NewMockSearchClient(results []interface{}) Client {
	idx := 0

	doFunc := func(_ context.Context, _ *elastic.SearchService) (*elastic.SearchResult, error) {
		if idx >= len(results) {
			return nil, errors.New("Enumerated past end of results")
		}
		result := results[idx]
		idx++

		switch rt := result.(type) {
		case *elastic.SearchResult:
			return rt, nil
		case elastic.SearchResult:
			return &rt, nil
		case error:
			return nil, rt
		case string:
			result := new(elastic.SearchResult)
			decoder := &elastic.DefaultDecoder{}
			err := decoder.Decode([]byte(rt), result)
			return result, err
		}

		return nil, errors.New("Unexpected result type")
	}

	return NewMockClient(doFunc)
}
