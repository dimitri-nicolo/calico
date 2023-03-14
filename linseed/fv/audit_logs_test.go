// Copyright (c) 2023 Tigera, Inc. All rights reserved.

//go:build fvtests

package fv_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"

	"k8s.io/apiserver/pkg/apis/audit"

	"github.com/projectcalico/calico/linseed/pkg/backend/testutils"

	elastic "github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/projectcalico/calico/libcalico-go/lib/logutils"
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/client"
	"github.com/projectcalico/calico/linseed/pkg/client/rest"
	"github.com/projectcalico/calico/linseed/pkg/config"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

func auditSetupAndTeardown(t *testing.T) func() {
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
		testutils.CleanupIndices(context.Background(), esClient, fmt.Sprintf("tigera_secure_ee_audit_ee.%s", cluster))
		testutils.CleanupIndices(context.Background(), esClient, fmt.Sprintf("tigera_secure_ee_audit_kube.%s", cluster))
		logCancel()
		cancel()
	}
}

func TestFV_AuditEE(t *testing.T) {
	t.Run("should return an empty list if there are no EE audits", func(t *testing.T) {
		defer auditSetupAndTeardown(t)()

		params := v1.AuditLogParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: time.Now().Add(-5 * time.Second),
					To:   time.Now(),
				},
			},
			Type: v1.AuditLogTypeEE,
		}

		// Perform a query.
		audits, err := cli.AuditLogs(cluster).List(ctx, &params)
		require.NoError(t, err)
		require.Equal(t, []v1.AuditLog{}, audits.Items)
	})

	t.Run("should create and list EE audits", func(t *testing.T) {
		defer auditSetupAndTeardown(t)()

		reqTime := time.Now()
		// Create a basic audit log
		audits := []v1.AuditLog{{Event: audit.Event{
			AuditID:                  "any-ee-id",
			RequestReceivedTimestamp: metav1.NewMicroTime(reqTime),
		}}}
		bulk, err := cli.AuditLogs(cluster).Create(ctx, v1.AuditLogTypeEE, audits)
		require.NoError(t, err)
		require.Equal(t, bulk.Succeeded, 1, "create audit did not succeed")

		// Refresh elasticsearch so that results appear.
		testutils.RefreshIndex(ctx, lmaClient, "tigera_secure_ee_audit_ee*")

		// Read it back.
		params := v1.AuditLogParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: reqTime.Add(-5 * time.Second),
					To:   reqTime.Add(5 * time.Second),
				},
			},
			Type: v1.AuditLogTypeEE,
		}
		resp, err := cli.AuditLogs(cluster).List(ctx, &params)
		require.NoError(t, err)
		require.Len(t, resp.Items, 1)

		// Reset the time as it microseconds to not match perfectly
		require.NotEqual(t, "", resp.Items[0].RequestReceivedTimestamp)
		resp.Items[0].RequestReceivedTimestamp = metav1.NewMicroTime(reqTime)

		require.Equal(t, audits, resp.Items)
	})

	t.Run("should return an empty list if there are no Kube audits", func(t *testing.T) {
		defer auditSetupAndTeardown(t)()

		params := v1.AuditLogParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: time.Now().Add(-5 * time.Second),
					To:   time.Now(),
				},
			},
			Type: v1.AuditLogTypeKube,
		}

		// Perform a query.
		audits, err := cli.AuditLogs(cluster).List(ctx, &params)
		require.NoError(t, err)
		require.Equal(t, []v1.AuditLog{}, audits.Items)
	})

	t.Run("should create and list Kube audits", func(t *testing.T) {
		defer auditSetupAndTeardown(t)()

		// Create a basic audit log.
		reqTime := time.Now()
		audits := []v1.AuditLog{{Event: audit.Event{
			AuditID:                  "any-kube-id",
			RequestReceivedTimestamp: metav1.NewMicroTime(reqTime),
		}}}
		bulk, err := cli.AuditLogs(cluster).Create(ctx, v1.AuditLogTypeKube, audits)
		require.NoError(t, err)
		require.Equal(t, bulk.Succeeded, 1, "create audit did not succeed")

		// Refresh elasticsearch so that results appear.
		testutils.RefreshIndex(ctx, lmaClient, "tigera_secure_ee_audit_kube*")

		// Read it back.
		params := v1.AuditLogParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: reqTime.Add(-5 * time.Second),
					To:   reqTime.Add(5 * time.Second),
				},
			},
			Type: v1.AuditLogTypeKube,
		}
		resp, err := cli.AuditLogs(cluster).List(ctx, &params)
		require.NoError(t, err)

		require.Len(t, resp.Items, 1)
		// Reset the time as it microseconds to not match perfectly
		require.NotEqual(t, "", resp.Items[0].RequestReceivedTimestamp)
		resp.Items[0].RequestReceivedTimestamp = metav1.NewMicroTime(reqTime)

		require.Equal(t, audits, resp.Items)
	})
}
