// Copyright (c) 2023 Tigera, Inc. All rights reserved.

//go:build fvtests

package fv_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/projectcalico/calico/libcalico-go/lib/logutils"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/testutils"
	"github.com/projectcalico/calico/linseed/pkg/client"
	"github.com/projectcalico/calico/linseed/pkg/config"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

var (
	cli                     client.Client
	multiClusterQueryClient client.Client
	ctx                     context.Context
	lmaClient               lmaelastic.Client
	cluster1                string
	cluster2                string
	cluster3                string
	cluster1Info            bapi.ClusterInfo
	cluster2Info            bapi.ClusterInfo
	cluster3Info            bapi.ClusterInfo
	esClient                *elastic.Client
)

// setupAndTeardown provides common setup and teardown logic for all FV tests to use.
// It allows passing arugments for configuring the linseed instance, and the index to use for the test.
func setupAndTeardown(t *testing.T, args *RunLinseedArgs, confArgs *RunConfigureElasticArgs, idx bapi.Index) func() {
	// Hook logrus into testing.T
	config.ConfigureLogging("DEBUG")
	logCancel := logutils.RedirectLogrusToTestingT(t)

	// Configure elastic if needed
	if confArgs != nil {
		RunConfigureElasticLinseed(t, confArgs)
	}
	// Start a linseed instance.
	if args == nil {
		args = DefaultLinseedArgs()
	}
	linseed := RunLinseed(t, args)

	// Create an ES client.
	var err error

	// Load credentials from environment variables
	username := os.Getenv("ELASTIC_USERNAME")
	password := os.Getenv("ELASTIC_PASSWORD")

	// Get the current working directory, which we expect to by the fv dir.
	cwd, err := os.Getwd()
	require.NoError(t, err)

	// Turn it to an absolute path.
	cwd, err = filepath.Abs(cwd)
	require.NoError(t, err)

	caCertFile := fmt.Sprintf("%s/cert/http_ca.crt", cwd)
	caCert, err := ioutil.ReadFile(caCertFile)
	if err != nil {
		log.Fatalf("Error loading CA certificate: %s", err)
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	esClient, err = elastic.NewSimpleClient(elastic.SetURL("https://localhost:9200"),
		elastic.SetBasicAuth(username, password),
		elastic.SetInfoLog(logrus.StandardLogger()),
		elastic.SetSniff(false),
		elastic.SetHealthcheck(false),
		elastic.SetHttpClient(&http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs: caCertPool,
				},
			},
		}),
	)

	require.NoError(t, err)
	lmaClient = lmaelastic.NewWithClient(esClient)

	// Instantiate a Linseed client.
	cli, err = NewLinseedClient(args, TokenPath)
	require.NoError(t, err)

	// Instantiate a Linseed client with a token permitted to perform multi-cluster queries.
	multiClusterQueryClient, err = NewLinseedClient(args, TokenPathMultiCluster)
	require.NoError(t, err)

	// Create a random cluster name for each test to make sure we don't interfere between tests.
	cluster1 = testutils.RandomClusterName()
	cluster2 = testutils.RandomClusterName()
	cluster3 = testutils.RandomClusterName()
	cluster1Info = bapi.ClusterInfo{Cluster: cluster1, Tenant: args.TenantID}
	cluster2Info = bapi.ClusterInfo{Cluster: cluster2, Tenant: args.TenantID}
	cluster3Info = bapi.ClusterInfo{Cluster: cluster3, Tenant: args.TenantID}

	// Set up context with a timeout.
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)

	return func() {
		linseed.Stop()
		for _, clusterInfo := range []bapi.ClusterInfo{cluster1Info, cluster2Info, cluster3Info} {
			err := testutils.CleanupIndices(context.Background(), esClient, idx.IsSingleIndex(), idx, clusterInfo)
			require.NoError(t, err)
		}
		logCancel()
		cancel()
	}
}
