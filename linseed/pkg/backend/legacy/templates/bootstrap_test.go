// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package templates_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/projectcalico/calico/libcalico-go/lib/logutils"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/templates"
	"github.com/projectcalico/calico/linseed/pkg/backend/testutils"
	"github.com/projectcalico/calico/linseed/pkg/config"
)

var (
	client *elastic.Client
	ctx    context.Context
)

func setupTest(t *testing.T) func() {
	// Hook logrus into testing.T
	config.ConfigureLogging("DEBUG")
	logCancel := logutils.RedirectLogrusToTestingT(t)

	// Create an elasticsearch client to use for the test. For this suite, we use a real
	// elasticsearch instance created via "make run-elastic".
	var err error
	client, err = elastic.NewSimpleClient(elastic.SetURL("http://localhost:9200"), elastic.SetInfoLog(logrus.StandardLogger()))
	require.NoError(t, err)

	// Each test should take less than 5 seconds.
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)

	return func() {
		// Cleanup after ourselves.
		err = testutils.CleanupIndices(context.Background(), client, "tigera_secure_ee_flows.test")
		require.NoError(t, err)

		// Cancel the context
		cancel()
		logCancel()
	}
}

func TestBootstrapTemplate(t *testing.T) {
	defer setupTest(t)()

	// Check that the template returned has the correct
	// index_patterns, ILM policy, mappings and shards and replicas
	application := "app-test"
	templateConfig := templates.NewTemplateConfig(bapi.FlowLogs, bapi.ClusterInfo{Cluster: "test"},
		templates.WithApplication(application))

	_, err := templates.IndexBootstrapper(ctx, client, templateConfig)
	require.NoError(t, err)

	// Check that the template was created
	templateExists, err := client.IndexTemplateExists("tigera_secure_ee_flows.test.").Do(ctx)
	require.NoError(t, err)
	require.True(t, templateExists)

	// Check that the bootstrap index exists
	index := fmt.Sprintf("tigera_secure_ee_flows.test.%s-%s-000001", application, time.Now().Format("20060102"))
	indexExists, err := client.IndexExists(index).Do(ctx)
	require.NoError(t, err)
	require.True(t, indexExists, "index doesn't exist: %s", index)

	// Check that write alias exists
	responseAlias, err := client.CatAliases().Do(ctx)
	require.NoError(t, err)
	require.Greater(t, len(responseAlias), 0)
	var hasAlias bool
	for _, row := range responseAlias {
		if row.Alias == "tigera_secure_ee_flows.test." {
			hasAlias = true
			require.Equal(t, row.Index, index)
			require.Equal(t, row.IsWriteIndex, "true")
			break
		}
	}
	require.True(t, hasAlias)
}

func TestBootstrapTemplateMultipleTimes(t *testing.T) {
	defer setupTest(t)()

	templateConfig := templates.NewTemplateConfig(bapi.FlowLogs, bapi.ClusterInfo{Cluster: "test"})

	for i := 0; i < 10; i++ {
		_, err := templates.IndexBootstrapper(ctx, client, templateConfig)
		require.NoError(t, err)
	}
}
