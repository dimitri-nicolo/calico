package client

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	log "github.com/sirupsen/logrus"

	querycacheclient "github.com/projectcalico/calico/ts-queryserver/pkg/querycache/client"
)

type QueryServerConfig struct {
	QueryServerTunnelURL    string
	QueryServerURL          string
	QueryServerCA           string
	QueryServerToken        string
	AddImpersonationHeaders bool
}

var errInvalidToken = errors.New("queryServer Token is not valid")

type QueryServerClient interface {
	Client() *http.Client
	SearchEndpoints(*QueryServerConfig, *querycacheclient.QueryEndpointsReq) (*http.Response, error)
}

type queryServerClient struct {
	client                  *http.Client
	addImpersonationHeaders bool
}

type QueryServerResults struct {
	Err        error
	Body       []byte
	StatusCode int
}

func (q *queryServerClient) Client() *http.Client {
	return q.client
}

func NewQueryServerClient(cfg *QueryServerConfig) (*queryServerClient, error) {
	// create client
	cert, err := os.ReadFile(cfg.QueryServerCA)
	if err != nil {
		log.WithError(err).Info("failed to read queryserver CA from file: ", err)
		return nil, err
	}
	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(cert)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: certPool,
			},
		},
	}

	return &queryServerClient{
		client:                  client,
		addImpersonationHeaders: cfg.AddImpersonationHeaders,
	}, nil
}

func (q *queryServerClient) SearchEndpoints(cfg *QueryServerConfig, reqBody *querycacheclient.QueryEndpointsReqBody,
	clusterId string) (*querycacheclient.QueryEndpointsResp, error) {
	reqBodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		log.WithError(err).Info("failed to json.marshal QueryEndpointsReqBody: ", err)
		return nil, err
	}
	path := "/endpoints"

	// service url is set differently for mcm vs standalone environments
	url := cfg.QueryServerURL
	if clusterId != "cluster" {
		url = cfg.QueryServerTunnelURL
	}

	req, err := http.NewRequest("POST", url+path, bytes.NewBuffer(reqBodyBytes))
	if err != nil {
		log.WithError(err).Info("failed to create http request: ", err)
		return nil, err
	}
	if cfg.QueryServerToken == "" {
		log.WithError(errInvalidToken).Info("queryserver token path is empty: ", errInvalidToken)
		return nil, errInvalidToken
	}

	token, err := os.ReadFile(cfg.QueryServerToken)
	if err != nil {
		log.WithError(err).Info("failed to read token from file: ", err)
		return nil, err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Add("Accept", "application/json")
	req.Header.Add("x-cluster-id", clusterId)

	if q.addImpersonationHeaders {
		// This is a multi-tenant management cluster. In this setup, in order to fetch information
		// from the managed cluster, we must impersonate the canonical service account
		// The Bearer token stores a service account created with a tenant namespace, but we need
		// to use the canonical service account tigera-manager from tigera-manager namespace
		req.Header.Add("Impersonate-User", "system:serviceaccount:tigera-manager:tigera-manager")
		req.Header.Add("Impersonate-Group", "system:authenticated")
		req.Header.Add("Impersonate-Group", "system:serviceaccounts")
		req.Header.Add("Impersonate-Group", "system:serviceaccounts:tigera-manager")
	}

	resp, err := q.client.Do(req)
	if err != nil {
		log.WithError(err).Info("failed to execute queryserver request: ", err)
		return nil, err
	}

	// read response from queryserver endpoints call
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.WithError(err).Error("call to read response body from queryserver failed.")
		return nil, errors.New("failed to read response from queryserver")
	}

	qsResp := querycacheclient.QueryEndpointsResp{}
	err = json.Unmarshal(respBytes, &qsResp)
	if err != nil {
		log.Errorf("Response: %s", string(respBytes))
		log.WithError(err).Error("unmarshaling endpointsRespBody failed.")
		return nil, err
	}

	return &qsResp, err
}
