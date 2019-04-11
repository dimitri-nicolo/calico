// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package elastic

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/olivere/elastic"
	log "github.com/sirupsen/logrus"

	"github.com/tigera/compliance/pkg/event"
	"github.com/tigera/compliance/pkg/list"
	"github.com/tigera/compliance/pkg/report"
)

const (
	DefaultElasticScheme = "http"
	DefaultElasticHost   = "elasticsearch-tigera-elasticsearch.calico-monitoring.svc.cluster.local"
	DefaultElasticPort   = 9200
	DefaultElasticUser   = "elastic"
)

type Client interface {
	report.AuditLogReportHandler
	report.ReportRetriever
	report.ReportStorer
	list.Destination
	event.Fetcher
	EnsureIndices() error
	Backend() *elastic.Client
}

// client implements the Client interface.
type client struct {
	*elastic.Client
}

// MustGetElasticClient returns the elastic Client, or panics if it's not possible.
func MustGetElasticClient() Client {
	c, err := NewFromEnv()
	if err != nil {
		panic(err)
	}
	return c
}

// NewFromEnv returns a new elastic Client using configuration in the environments.
func NewFromEnv() (Client, error) {
	var u *url.URL
	uri := os.Getenv("ELASTIC_URI")
	if uri != "" {
		var err error
		u, err = url.Parse(uri)
		if err != nil {
			return nil, err
		}
	} else {
		scheme := os.Getenv("ELASTIC_SCHEME")
		if scheme == "" {
			scheme = DefaultElasticScheme
		}

		host := os.Getenv("ELASTIC_HOST")
		if host == "" {
			host = DefaultElasticHost
		}

		portStr := os.Getenv("ELASTIC_PORT")
		var port int64
		if portStr == "" {
			port = DefaultElasticPort
		} else {
			var err error
			port, err = strconv.ParseInt(portStr, 10, 16)
			if err != nil {
				return nil, err
			}
		}

		u = &url.URL{
			Scheme: scheme,
			Host:   fmt.Sprintf("%s:%d", host, port),
		}
	}
	log.WithField("url", u).Debug("using elastic url")

	//log.SetLevel(log.TraceLevel)
	user := os.Getenv("ELASTIC_USER")
	if user == "" {
		user = DefaultElasticUser
	}
	pass := os.Getenv("ELASTIC_PASSWORD")
	pathToCA := os.Getenv("ELASTIC_CA")

	ca, err := x509.SystemCertPool()
	if err != nil {
		return nil, err
	}
	if pathToCA != "" {
		cert, err := ioutil.ReadFile(pathToCA)
		if err != nil {
			return nil, err
		}
		ok := ca.AppendCertsFromPEM(cert)
		if !ok {
			return nil, fmt.Errorf("failed to add CA")
		}
	}
	h := &http.Client{}
	if u.Scheme == "https" {
		h.Transport = &http.Transport{TLSClientConfig: &tls.Config{RootCAs: ca}}
	}
	return New(h, u, user, pass)
}

func New(h *http.Client, url *url.URL, username, password string) (Client, error) {
	options := []elastic.ClientOptionFunc{
		elastic.SetURL(url.String()),
		elastic.SetHttpClient(h),
		elastic.SetErrorLog(log.StandardLogger()),
		elastic.SetSniff(false),
		//elastic.SetTraceLog(log.StandardLogger()),
	}
	if username != "" {
		options = append(options, elastic.SetBasicAuth(username, password))
	}
	c, err := elastic.NewClient(options...)
	if err != nil {
		return nil, err
	}
	return &client{c}, nil
}

func (c *client) EnsureIndices() error {
	if err := c.ensureIndexExists(snapshotsIndex, snapshotsMapping); err != nil {
		return err
	}
	return c.ensureIndexExists(reportsIndex, reportsMapping)
}

func (c *client) ensureIndexExists(index, mapping string) error {
	clog := log.WithField("index", index)

	// Check if index exists.
	exists, err := c.IndexExists(index).Do(context.Background())
	if err != nil {
		clog.WithError(err).Error("failed to check if index exists")
		return err
	}

	// Return if index exists
	if exists {
		clog.Info("index already exists, bailing out...")
		return nil
	}

	// Create index.
	clog.Info("index doesn't exist, creating...")
	createIndex, err := c.
		CreateIndex(index).
		Body(mapping).
		Do(context.Background())
	if err != nil {
		clog.WithError(err).Error("failed to create index")
		return err
	}

	// Check if acknowledged
	if !createIndex.Acknowledged {
		clog.Warn("index creation has not yet been acknowledged...")
	}
	clog.Info("index successfully created!")
	return nil
}

func (c *client) Backend() *elastic.Client {
	return c.Client
}
