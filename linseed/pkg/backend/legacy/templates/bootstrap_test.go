// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package templates_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/projectcalico/calico/linseed/pkg/backend/testutils"

	"github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/projectcalico/calico/libcalico-go/lib/logutils"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/templates"
	"github.com/projectcalico/calico/linseed/pkg/config"
)

var (
	client  *elastic.Client
	ctx     context.Context
	cluster string
)

func setupTest(t *testing.T, indices []string) func() {
	// Hook logrus into testing.T
	config.ConfigureLogging("DEBUG")
	logCancel := logutils.RedirectLogrusToTestingT(t)

	// Create an elasticsearch client to use for the test. For this suite, we use a real
	// elasticsearch instance created via "make run-elastic".
	var err error
	client, err = elastic.NewSimpleClient(elastic.SetURL("http://localhost:9200"), elastic.SetInfoLog(logrus.StandardLogger()))
	require.NoError(t, err)

	// Create a random cluster name for each test to make sure we don't
	// interfere between tests.
	cluster = testutils.RandomClusterName()

	// Each test should take less than 5 seconds.
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)

	return func() {
		// Cleanup after ourselves.
		err = testutils.CleanupIndices(context.Background(), client, cluster)
		require.NoError(t, err)

		// Cancel the context
		cancel()
		logCancel()
	}
}

func TestBootstrapTemplate(t *testing.T) {
	defer setupTest(t, []string{"tigera_secure_ee_flows"})()

	// Check that the template returned has the correct
	// index_patterns, ILM policy, mappings and shards and replicas
	templateConfig := templates.NewTemplateConfig(bapi.FlowLogs, bapi.ClusterInfo{Cluster: cluster})

	templ, err := templates.IndexBootstrapper(ctx, client, templateConfig)
	require.NoError(t, err)
	require.NotNil(t, templ)
	require.Len(t, templ.IndexPatterns, 1)

	checkTemplateBootstrapping(t, "tigera_secure_ee_flows", "fluentd", cluster)
}

func checkTemplateBootstrapping(t *testing.T, indexPrefix, application, cluster string) {
	// Check that the template was created
	templateExists, err := client.IndexTemplateExists(fmt.Sprintf("%s.%s.", indexPrefix, cluster)).Do(ctx)
	require.NoError(t, err)
	require.True(t, templateExists)

	// Check that the bootstrap index exists
	index := fmt.Sprintf("%s.%s.%s-%s-000001", indexPrefix, cluster, application, time.Now().UTC().Format("20060102"))
	indexExists, err := client.IndexExists(index).Do(ctx)
	require.NoError(t, err)
	require.True(t, indexExists, "index doesn't exist: %s", index)

	// Check that write alias exists.
	responseAlias, err := client.CatAliases().Do(ctx)
	require.NoError(t, err)
	require.Greater(t, len(responseAlias), 0)
	hasAlias := false
	for _, row := range responseAlias {
		if row.Alias == fmt.Sprintf("%s.%s.", indexPrefix, cluster) {
			require.Equal(t, row.Index, index)
			require.Equal(t, row.IsWriteIndex, "true")
			hasAlias = true
			break
		}
	}
	require.True(t, hasAlias)

	responseSettings, err := client.IndexGetSettings(index).Do(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, responseSettings)
	require.Contains(t, responseSettings, index)
	require.NotEmpty(t, responseSettings[index].Settings)
	require.Contains(t, responseSettings[index].Settings, "index")
	settings, _ := responseSettings[index].Settings["index"].(map[string]interface{})
	// Check lifecycle section
	require.Contains(t, settings, "lifecycle")
	lifecycle, _ := settings["lifecycle"].(map[string]interface{})
	require.Contains(t, lifecycle, "name")
	require.EqualValues(t, lifecycle["name"], fmt.Sprintf("%s_policy", indexPrefix))
	require.EqualValues(t, lifecycle["rollover_alias"], fmt.Sprintf("%s.%s.", indexPrefix, cluster))
	// Check shards and replicas
	require.Contains(t, settings, "number_of_replicas")
	require.EqualValues(t, settings["number_of_replicas"], "0")
	require.Contains(t, settings, "number_of_shards")
	require.EqualValues(t, settings["number_of_shards"], "1")
	// Check template for index name
	require.Contains(t, settings, "provided_name")
	require.EqualValues(t, settings["provided_name"], fmt.Sprintf("<%s.%s.%s-{now/s{yyyyMMdd}}-000001>", indexPrefix, cluster, application))
}

func TestBootstrapAuditTemplates(t *testing.T) {
	defer setupTest(t, []string{"tigera_secure_ee_audit_ee", "tigera_secure_ee_audit_kube"})()

	auditKubeTemplateConfig := templates.NewTemplateConfig(bapi.AuditKubeLogs, bapi.ClusterInfo{Cluster: cluster})
	auditEETemplateConfig := templates.NewTemplateConfig(bapi.AuditEELogs, bapi.ClusterInfo{Cluster: cluster})

	templKubeAudit, err := templates.IndexBootstrapper(ctx, client, auditKubeTemplateConfig)
	require.NoError(t, err)
	require.NotNil(t, templKubeAudit)
	require.Len(t, templKubeAudit.IndexPatterns, 1)

	templEEAudit, err := templates.IndexBootstrapper(ctx, client, auditEETemplateConfig)
	require.NoError(t, err)
	require.NotNil(t, templEEAudit)
	require.Len(t, templEEAudit.IndexPatterns, 1)

	// Check that the template returned has the correct
	// index_patterns, ILM policy, mappings and shards and replicas
	checkTemplateBootstrapping(t, "tigera_secure_ee_audit_kube", "fluentd", cluster)
	checkTemplateBootstrapping(t, "tigera_secure_ee_audit_ee", "fluentd", cluster)
}

func TestBootstrapTemplateMultipleTimes(t *testing.T) {
	defer setupTest(t, []string{"tigera_secure_ee_flows"})()

	templateConfig := templates.NewTemplateConfig(bapi.FlowLogs, bapi.ClusterInfo{Cluster: cluster})

	for i := 0; i < 10; i++ {
		_, err := templates.IndexBootstrapper(ctx, client, templateConfig)
		require.NoError(t, err)
		checkTemplateBootstrapping(t, "tigera_secure_ee_flows", "fluentd", cluster)
	}
}
