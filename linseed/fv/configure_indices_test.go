// Copyright (c) 2023 Tigera, Inc. All rights reserved.

//go:build fvtests

package fv_test

import (
	"context"
	"testing"
	"time"

	"github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/projectcalico/calico/libcalico-go/lib/logutils"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/index"
	"github.com/projectcalico/calico/linseed/pkg/backend/testutils"
	"github.com/projectcalico/calico/linseed/pkg/config"
)

// configureIndicesSetupAndTeardown performs additional setup and teardown for ingestion tests.
func configureIndicesSetupAndTeardown(t *testing.T, idx bapi.Index) func() {
	// Hook logrus into testing.T
	config.ConfigureLogging("DEBUG")
	logCancel := logutils.RedirectLogrusToTestingT(t)

	// Create an ES client.
	var err error
	esClient, err = elastic.NewSimpleClient(elastic.SetURL("http://localhost:9200"), elastic.SetInfoLog(logrus.StandardLogger()))
	require.NoError(t, err)

	// Set up context with a timeout.
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)

	return func() {
		testutils.CleanupIndices(context.Background(), esClient, idx.IsSingleIndex(), idx, clusterInfo)
		logCancel()
		cancel()
	}
}

func TestFV_ConfigureFlowIndices(t *testing.T) {
	t.Run("Configure Elastic Indices [SingleIndex]", func(t *testing.T) {
		idx := index.FlowLogIndex(index.WithIndexName("calico_flowlogs_free"), index.WithPolicyName("calico_free"))
		defer configureIndicesSetupAndTeardown(t, idx)()

		// Start a linseed configuration instance.
		args := &RunConfigureElasticArgs{
			FlowIndexName:  idx.Name(bapi.ClusterInfo{}),
			FlowPolicyName: idx.ILMPolicyName(),
		}
		linseed := RunConfigureElasticLinseed(t, args)
		defer func() {
			if linseed.ListedInDockerPS() {
				linseed.Stop()
			}
		}()

		testutils.CheckSingleIndexTemplateBootstrapping(t, ctx, esClient, idx, bapi.ClusterInfo{})
	})
}
