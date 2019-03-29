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
)

const (
	DefaultElasticScheme = "http"
	DefaultElasticHost   = "elasticsearch-tigera-elasticsearch.calico-monitoring.svc.cluster.local"
	DefaultElasticPort   = 9200
	DefaultElasticUser   = "elastic"

	snapshotsIndex   = "tigera_secure_ee_snapshots"
	snapshotsMapping = `{
  "mappings": {
    "_doc": {
      "properties": {
        "apiVersion": { "type": "text" },
        "kind": { "type": "text" },
        "items": {
          "properties": {
            "apiVersion": { "type": "text" },
            "kind": { "type": "text" },
            "metadata": { "type": "object" },
            "spec": { "type": "object", "enabled": false }
          }
        },
        "metadata": { "type": "object" },
        "requestStartedTimestamp": { "type": "date" },
        "requestCompletedTimestamp": { "type": "date" }
      }
    }
  }
}`
)

// TODO(rlb): This should be an interface not a public struct.
type Client struct {
	*elastic.Client
}

func NewFromEnv() (*Client, error) {
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

func New(h *http.Client, url *url.URL, username, password string) (*Client, error) {
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
	return &Client{c}, nil
}

func (c *Client) EnsureIndices() error {
	return c.ensureIndexExists(snapshotsIndex, snapshotsMapping)
}

func (c *Client) ensureIndexExists(index, mapping string) error {
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
