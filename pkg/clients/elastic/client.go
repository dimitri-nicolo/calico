package elastic

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	es7 "github.com/elastic/go-elasticsearch/v7"

	log "github.com/sirupsen/logrus"

	httpCommon "github.com/tigera/es-gateway/pkg/clients/internal/http"
)

// client is a wrapper for the ES client.
type client struct {
	*es7.Client
}

// Client is an interface that exposes the required ES API operations for ES Gateway.
type Client interface {
	AuthenticateUser(string) (*User, error)
	GetClusterHealth() error
}

// NewClient returns a newly configured ES client.
func NewClient(url, username, password, certPath string) (Client, error) {
	// Attempt to load
	cert, err := ioutil.ReadFile(certPath)
	if err != nil {
		return nil, err
	}

	certPool := x509.NewCertPool()
	ok := certPool.AppendCertsFromPEM(cert)
	if !ok {
		return nil, fmt.Errorf("failed to parse root certificate")
	}

	// Configure the ES client
	config := es7.Config{
		Addresses: []string{
			url,
		},
		Username: username,
		Password: password,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: certPool,
			},
		},
	}

	esClient, err := es7.NewClient(config)
	if err != nil {
		return nil, err
	}

	return &client{esClient}, nil
}

// User contains the revelant fields for an ES user that we want to examine in the ES Gateway.
type User struct {
	Username string
	Roles    []string
}

// AuthenticateUser takes the given credentials and attempts to validate them against the configured
// Elasticsearch backend. If the provided credentials are authenticated successfully a User is returned.
// Otherwise, an error is returned.
func (es *client) AuthenticateUser(authToken string) (*User, error) {
	auth := es.API.Security.Authenticate

	res, err := auth(auth.WithHeader(map[string]string{"Authorization": authToken}))
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	log.Debugf("Response for authentication attempt: %s", res.String())

	if res.IsError() {
		return nil, fmt.Errorf("failed to authenticate user: %s", res.String())
	}

	// Attempt to unmarshall the response payload and load into User type.
	user := &User{}
	err = json.NewDecoder(res.Body).Decode(user)
	if err != nil {
		return nil, err
	}

	log.Debugf("Authenticated user: %+v", user)
	return user, nil
}

// GetClusterHealth checks the health of the ES cluster that the client is connected to.
// If the response is anything other than HTTP 200, then an error is returned.
// Otherwise, we return nil.
// http://www.elastic.co/guide/en/elasticsearch/reference/master/cluster-health.html
func (es *client) GetClusterHealth() error {
	health := es.API.Cluster.Health

	res, err := health(health.WithTimeout(httpCommon.HealthCheckTimeout))
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf(res.String())
	}

	return nil
}
