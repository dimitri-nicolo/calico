package testutils

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"

	"github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
)

// CreateElasticClient initializes and returns an Elasticsearch client.
func CreateElasticClient() (*elastic.Client, error) {
	// Load credentials from environment variables
	username := os.Getenv("ELASTIC_USERNAME")
	password := os.Getenv("ELASTIC_PASSWORD")
	if username == "" || password == "" {
		return nil, fmt.Errorf("missing Elasticsearch credentials")
	}

	// Create and return an Elasticsearch client
	esClient, err := elastic.NewSimpleClient(
		elastic.SetURL("https://localhost:9200"),
		elastic.SetBasicAuth(username, password),
		elastic.SetInfoLog(logrus.StandardLogger()),
		elastic.SetHttpClient(&http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Elasticsearch client: %v", err)
	}

	return esClient, nil
}
