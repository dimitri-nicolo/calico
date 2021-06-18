package kibana

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

const kibanaRequestTimeout = time.Second * 10

// client is a wrapper for a simple HTTP client. We'll use this since there is no
// official Golang Kibana client library and we only need to call Kibana API for
// the health check.
type client struct {
	httpClient *http.Client
	baseURL    string
	username   string
	password   string
}

// Client is an interface that exposes the required Kibana API operations for ES Gateway.
type Client interface {
	GetKibanaStatus() error
}

// NewClient returns a newly configured ES client.
func NewClient(url, username, password, certPath string) (Client, error) {
	// Load CA cert
	caCert, err := ioutil.ReadFile(certPath)
	if err != nil {
		log.Fatal(err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// Setup HTTPS client
	tlsConfig := &tls.Config{
		RootCAs: caCertPool,
	}
	tlsConfig.BuildNameToCertificate()
	transport := &http.Transport{TLSClientConfig: tlsConfig}
	httpClient := &http.Client{
		Transport: transport,
		Timeout:   kibanaRequestTimeout,
	}

	return &client{
		httpClient: httpClient,
		baseURL:    url,
		username:   username,
		password:   password,
	}, nil
}

// GetKibanaStatus checks the status of the Kibana API that the client is connected to.
// If the response is anything other than HTTP 200, an error is returned.
// Otherwise, we return nil.
// https://www.elastic.co/guide/en/kibana/master/access.html#status
func (c *client) GetKibanaStatus() error {
	url := fmt.Sprintf("%s%s", c.baseURL, "/tigera-kibana/api/status")
	req, _ := http.NewRequest("GET", url, nil)
	req.SetBasicAuth(c.username, c.password)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	// Dump response
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf(string(data))
	}

	return nil
}
