// Copyright (c) 2023 Tigera, Inc. All rights reserved.

//go:build fvtests

package fv_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/projectcalico/calico/linseed/pkg/backend/testutils"

	elastic "github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/calico/libcalico-go/lib/logutils"
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/client"
	"github.com/projectcalico/calico/linseed/pkg/client/rest"
	"github.com/projectcalico/calico/linseed/pkg/config"
	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

var (
	cli       client.Client
	ctx       context.Context
	lmaClient lmaelastic.Client
	cluster   string
)

func flowlogSetupAndTeardown(t *testing.T) func() {
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
		testutils.CleanupIndices(context.Background(), esClient, fmt.Sprintf("tigera_secure_ee_flows.%s", cluster))
		logCancel()
		cancel()
	}
}

func TestFV_FlowLogs(t *testing.T) {
	t.Run("should return an empty list if there are no flow logs", func(t *testing.T) {
		defer flowlogSetupAndTeardown(t)()

		params := v1.FlowLogParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: time.Now().Add(-5 * time.Second),
					To:   time.Now(),
				},
			},
		}

		// Perform a query.
		logs, err := cli.FlowLogs(cluster).List(ctx, &params)
		require.NoError(t, err)
		require.Equal(t, []v1.FlowLog{}, logs.Items)
	})

	t.Run("should create and list flow logs", func(t *testing.T) {
		defer flowlogSetupAndTeardown(t)()

		// Create a basic flow log.
		logs := []v1.FlowLog{
			{
				EndTime: time.Now().Unix(), // TODO- more fields.
			},
		}
		bulk, err := cli.FlowLogs(cluster).Create(ctx, logs)
		require.NoError(t, err)
		require.Equal(t, bulk.Succeeded, 1, "create flow log did not succeed")

		// Refresh elasticsearch so that results appear.
		testutils.RefreshIndex(ctx, lmaClient, "tigera_secure_ee_flows*")

		// Read it back.
		params := v1.FlowLogParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: time.Now().Add(-5 * time.Second),
					To:   time.Now().Add(5 * time.Second),
				},
			},
		}
		resp, err := cli.FlowLogs(cluster).List(ctx, &params)
		require.NoError(t, err)
		require.Equal(t, logs, testutils.AssertLogIDAndCopyFlowLogsWithoutID(t, resp))
	})

	t.Run("should support pagination", func(t *testing.T) {
		defer flowlogSetupAndTeardown(t)()

		// Create 5 flow logs.
		logTime := time.Now().Unix()
		for i := 0; i < 5; i++ {
			logs := []v1.FlowLog{
				{
					StartTime: logTime,
					EndTime:   logTime + int64(i), // Make sure logs are ordered.
					Host:      fmt.Sprintf("%d", i),
				},
			}
			bulk, err := cli.FlowLogs(cluster).Create(ctx, logs)
			require.NoError(t, err)
			require.Equal(t, bulk.Succeeded, 1, "create flow log did not succeed")
		}

		// Refresh elasticsearch so that results appear.
		testutils.RefreshIndex(ctx, lmaClient, "tigera_secure_ee_flows*")

		// Read them back one at a time.
		var afterKey map[string]interface{}
		for i := 0; i < 5; i++ {
			params := v1.FlowLogParams{
				QueryParams: v1.QueryParams{
					TimeRange: &lmav1.TimeRange{
						From: time.Now().Add(-5 * time.Second),
						To:   time.Now().Add(5 * time.Second),
					},
					MaxPageSize: 1,
					AfterKey:    afterKey,
				},
			}
			resp, err := cli.FlowLogs(cluster).List(ctx, &params)
			require.NoError(t, err)
			require.Equal(t, 1, len(resp.Items))
			require.Equal(t, []v1.FlowLog{
				{
					StartTime: logTime,
					EndTime:   logTime + int64(i),
					Host:      fmt.Sprintf("%d", i),
				},
			}, testutils.AssertLogIDAndCopyFlowLogsWithoutID(t, resp), fmt.Sprintf("Flow #%d did not match", i))
			require.NotNil(t, resp.AfterKey)

			// Use the afterKey for the next query.
			afterKey = resp.AfterKey
		}

		// If we query once more, we should get no results, and no afterkey, since
		// we have paged through all of the items.
		params := v1.FlowLogParams{
			QueryParams: v1.QueryParams{
				TimeRange: &lmav1.TimeRange{
					From: time.Now().Add(-5 * time.Second),
					To:   time.Now().Add(5 * time.Second),
				},
				MaxPageSize: 1,
				AfterKey:    afterKey,
			},
		}
		resp, err := cli.FlowLogs(cluster).List(ctx, &params)
		require.NoError(t, err)
		require.Equal(t, 0, len(resp.Items))
		require.Nil(t, resp.AfterKey)
	})
}

func TestFV_FlowLogsRBAC(t *testing.T) {
	type filterTestCase struct {
		name        string
		permissions []v3.AuthorizedResourceVerbs

		sourceType      string
		sourceNamespace string
		destType        string
		destNamespace   string

		expectError bool
		expectMatch bool
	}

	testcases := []filterTestCase{
		// Create a request with no List permissions. It should return an error.
		{
			name: "should reject requests with no list permissions",
			permissions: []v3.AuthorizedResourceVerbs{
				{
					APIGroup: "projectcalico.org/v3",
					Resource: "workloadendpoints",
					Verbs: []v3.AuthorizedResourceVerb{
						{
							Verb: "create",
						},
					},
				},
			},
			expectError: true,
		},

		// Create a flow log with source type WEP, but only
		// provide permissions for HEP. We shouldn't get any results.
		{
			name:            "should filter out on source type",
			sourceType:      "wep",
			sourceNamespace: "default",
			permissions: []v3.AuthorizedResourceVerbs{
				{
					APIGroup: "projectcalico.org/v3",
					Resource: "hostendpoints",
					Verbs: []v3.AuthorizedResourceVerb{
						{
							Verb: "list",
							ResourceGroups: []v3.AuthorizedResourceGroup{
								{Namespace: ""},
							},
						},
					},
				},
			},
			expectError: false,
			expectMatch: false,
		},

		// Create a flow log with source type WEP, provide permissions for pods.
		// We should be able to query the log.
		{
			name:            "should select on source type",
			sourceType:      "wep",
			sourceNamespace: "default",
			permissions: []v3.AuthorizedResourceVerbs{
				{
					APIGroup: "projectcalico.org/v3",
					Resource: "pods",
					Verbs: []v3.AuthorizedResourceVerb{
						{
							Verb: "list",
							ResourceGroups: []v3.AuthorizedResourceGroup{
								{Namespace: ""},
							},
						},
					},
				},
			},
			expectError: false,
			expectMatch: true,
		},

		// Create a flow log with source type WEP, provide permissions for pods in
		// a different namespace, but not the flow log's namespace.
		// We should not see the log in the response.
		{
			name:            "should filter out based on source namespace",
			sourceType:      "wep",
			sourceNamespace: "default",
			permissions: []v3.AuthorizedResourceVerbs{
				{
					APIGroup: "projectcalico.org/v3",
					Resource: "pods",
					Verbs: []v3.AuthorizedResourceVerb{
						{
							Verb: "list",
							ResourceGroups: []v3.AuthorizedResourceGroup{
								{Namespace: "another-namespace"},
							},
						},
					},
				},
			},
			expectError: false,
			expectMatch: false,
		},

		// Create a flow log with destination of a global network set.
		// Allow permissions for network sets in all namespaces.
		// We should not see the log in the response.
		{
			name:            "should filter out based on source namespace",
			sourceType:      "wep",
			sourceNamespace: "default",
			destType:        "ns",
			destNamespace:   "-",
			permissions: []v3.AuthorizedResourceVerbs{
				{
					APIGroup: "projectcalico.org/v3",
					Resource: "networksets",
					Verbs: []v3.AuthorizedResourceVerb{
						{
							Verb: "list",
							ResourceGroups: []v3.AuthorizedResourceGroup{
								{Namespace: ""},
							},
						},
					},
				},
			},
			expectError: false,
			expectMatch: false,
		},

		// Create a flow log with destination of a global network set.
		// Allow permissions for global network sets.
		// We should see the log in the response.
		{
			name:            "should filter out based on source namespace",
			sourceType:      "wep",
			sourceNamespace: "default",
			destType:        "ns",
			destNamespace:   "-",
			permissions: []v3.AuthorizedResourceVerbs{
				{
					APIGroup: "projectcalico.org/v3",
					Resource: "globalnetworksets",
					Verbs: []v3.AuthorizedResourceVerb{
						{
							Verb: "list",
							ResourceGroups: []v3.AuthorizedResourceGroup{
								{Namespace: ""},
							},
						},
					},
				},
			},
			expectError: false,
			expectMatch: true,
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			defer flowlogSetupAndTeardown(t)()

			// Create a flow log with the given parameters.
			logs := []v1.FlowLog{
				{
					SourceNamespace: testcase.sourceNamespace,
					SourceType:      testcase.sourceType,
					DestNamespace:   testcase.destNamespace,
					DestType:        testcase.destType,
					EndTime:         time.Now().Unix(),
				},
			}
			bulk, err := cli.FlowLogs(cluster).Create(ctx, logs)
			require.NoError(t, err)
			require.Equal(t, bulk.Succeeded, 1, "create flow log did not succeed")

			// Refresh elasticsearch so that results appear.
			testutils.RefreshIndex(ctx, lmaClient, "tigera_secure_ee_flows*")

			// Perform a query using the testcase permissions.
			params := v1.FlowLogParams{
				QueryParams: v1.QueryParams{
					TimeRange: &lmav1.TimeRange{
						From: time.Now().Add(-5 * time.Second),
						To:   time.Now().Add(5 * time.Second),
					},
					MaxPageSize: 1,
				},
				LogSelectionParams: v1.LogSelectionParams{Permissions: testcase.permissions},
			}
			resp, err := cli.FlowLogs(cluster).List(ctx, &params)

			if testcase.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if testcase.expectMatch {
				require.Equal(t, logs, testutils.AssertLogIDAndCopyFlowLogsWithoutID(t, resp))
			}
		})
	}
}
