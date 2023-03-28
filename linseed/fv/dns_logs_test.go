// Copyright (c) 2023 Tigera, Inc. All rights reserved.

//go:build fvtests

package fv_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/projectcalico/calico/linseed/pkg/backend/testutils"

	"github.com/google/gopacket/layers"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"

	elastic "github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/projectcalico/calico/libcalico-go/lib/logutils"
	"github.com/projectcalico/calico/linseed/pkg/client"
	"github.com/projectcalico/calico/linseed/pkg/client/rest"
	"github.com/projectcalico/calico/linseed/pkg/config"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

func dnslogSetupAndTeardown(t *testing.T) func() {
	// Hook logrus into testing.T
	config.ConfigureLogging("DEBUG")
	logCancel := logutils.RedirectLogrusToTestingT(t)

	// Create an ES client.
	esClient, err := elastic.NewSimpleClient(elastic.SetURL("http://localhost:9200"), elastic.SetInfoLog(logrus.StandardLogger()))
	require.NoError(t, err)
	lmaClient = lmaelastic.NewWithClient(esClient)

	// Instantiate a client.
	cfg := rest.Config{
		CACertPath:     "cert/RootCA.crt",
		URL:            "https://localhost:8444/",
		ClientCertPath: "cert/localhost.crt",
		ClientKeyPath:  "cert/localhost.key",
	}
	cli, err = client.NewClient("", cfg)
	require.NoError(t, err)

	// Create a random cluster name for each test to make sure we don't
	// interfere between tests.
	cluster = testutils.RandomClusterName()

	// Set up context with a timeout.
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)

	return func() {
		// Cleanup indices created by the test.
		testutils.CleanupIndices(context.Background(), esClient, fmt.Sprintf("tigera_secure_ee_dns.%s", cluster))
		logCancel()
		cancel()
	}
}

func TestDNS_FlowLogs(t *testing.T) {
	t.Run("should return an empty list if there are no dns logs", func(t *testing.T) {
		defer dnslogSetupAndTeardown(t)()

		params := v1.DNSLogParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: time.Now().Add(-5 * time.Second),
					To:   time.Now(),
				},
			},
		}

		// Perform a query.
		logs, err := cli.DNSLogs(cluster).List(ctx, &params)
		require.NoError(t, err)
		require.Equal(t, []v1.DNSLog{}, logs.Items)
	})

	t.Run("should create and list dns logs", func(t *testing.T) {
		defer dnslogSetupAndTeardown(t)()

		// Create a basic flow log.
		logs := []v1.DNSLog{
			{
				EndTime: time.Now().UTC(), // TODO: Add more fields
				QName:   "service.namespace.svc.cluster.local",
				QClass:  v1.DNSClass(layers.DNSClassIN),
				QType:   v1.DNSType(layers.DNSTypeAAAA),
				RCode:   v1.DNSResponseCode(layers.DNSResponseCodeNXDomain),
				RRSets:  v1.DNSRRSets{},
			},
		}
		bulk, err := cli.DNSLogs(cluster).Create(ctx, logs)
		require.NoError(t, err)
		require.Equal(t, bulk.Succeeded, 1, "create dns log did not succeed")

		// Refresh elasticsearch so that results appear.
		testutils.RefreshIndex(ctx, lmaClient, "tigera_secure_ee_dns*")

		// Read it back.
		params := v1.DNSLogParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: time.Now().Add(-5 * time.Second),
					To:   time.Now().Add(5 * time.Second),
				},
			},
		}
		resp, err := cli.DNSLogs(cluster).List(ctx, &params)
		require.NoError(t, err)
		actualLogs := testutils.AssertLogIDAndCopyDNSLogsWithoutID(t, resp)
		require.Equal(t, logs, actualLogs)
	})
}
