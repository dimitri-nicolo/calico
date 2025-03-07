// Copyright (c) 2025 Tigera, Inc. All rights reserved.

//go:build fvtests

package fv

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/projectcalico/calico/linseed/pkg/backend/api"
	backendutils "github.com/projectcalico/calico/linseed/pkg/backend/testutils"
	"github.com/projectcalico/calico/oiler/pkg/checkpoint"
)

const (
	lastGeneratedReadMetric    = `tigera_oiler_last_read_generated_timestamp{job_name="%s"}`
	lastGeneratedWrittenMetric = `tigera_oiler_last_written_generated_timestamp{job_name="%s"}`
	docsReadMetric             = `tigera_oiler_docs_read{cluster_id="%s",job_name="%s",source="primary",tenant_id="%s"}`
	docsWrittenMetric          = `tigera_oiler_docs_writes_successful{cluster_id="%s",job_name="%s",source="secondary",tenant_id="%s"}`
	eNotationFloatingPoint     = "[-+]?[0-9]*\\.?[0-9]+([eE][-+]?[0-9]+)?"
)

func readMetrics(t *testing.T) []byte {
	var err error
	resp, err := http.Get("http://localhost:8080/metrics")
	require.NoError(t, err)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return body
}

func getValue(t *testing.T, allMetrics []byte, name string) float64 {
	// Metrics have the following format `name{label="abc} 1.2e+1`
	metric := regexp.MustCompile(fmt.Sprintf("%s [0-9+-eE.]+", name)).Find(allMetrics)
	valStr := string(regexp.MustCompile(eNotationFloatingPoint).Find(metric))
	require.NotEmpty(t, valStr)
	value, err := strconv.ParseFloat(valStr, 64)
	require.NoError(t, err)
	return value
}

func validateMetrics(t *testing.T, jobName string, primary api.ClusterInfo, secondary api.ClusterInfo, numberOfLogs, last int64) {
	metrics := readMetrics(t)

	lastGeneratedRead := getValue(t, metrics, fmt.Sprintf(lastGeneratedReadMetric, jobName))
	lastGeneratedWritten := getValue(t, metrics, fmt.Sprintf(lastGeneratedWrittenMetric, jobName))
	docsRead := getValue(t, metrics, fmt.Sprintf(docsReadMetric, primary.Cluster, jobName, primary.Tenant))
	docsWritten := getValue(t, metrics, fmt.Sprintf(docsWrittenMetric, secondary.Cluster, jobName, secondary.Tenant))

	require.Equal(t, float64(numberOfLogs), docsRead)
	require.Equal(t, float64(numberOfLogs), docsWritten)
	require.Equal(t, float64(last), lastGeneratedRead)
	require.InDelta(t, float64(last), lastGeneratedWritten, 5000)
}

func cleanUpData(t *testing.T, primary api.ClusterInfo, secondary api.ClusterInfo, idx api.Index) {
	for _, clusterInfo := range []api.ClusterInfo{primary, secondary} {
		err := backendutils.CleanupIndices(context.Background(), esClient.Backend(), idx.IsSingleIndex(), idx, clusterInfo)
		require.NoError(t, err)
	}
}

func validateCheckpoints(t *testing.T, dataType api.DataType, primary api.ClusterInfo, last time.Time) {
	configMapName := checkpoint.ConfigMapName(dataType, primary.Tenant, primary.Cluster)
	configMap, err := k8sClient.CoreV1().ConfigMaps("default").Get(ctx, configMapName, metav1.GetOptions{})
	require.NoError(t, err)
	require.NotNil(t, configMap)
	require.NotNil(t, configMap.Data)
	lastCheckpoint, ok := configMap.Data["checkpoint"]
	require.True(t, ok)
	val, err := time.Parse(time.RFC3339, lastCheckpoint)
	require.NoError(t, err)
	require.InDelta(t, last.UnixMilli(), val.UnixMilli(), 1000)
}

func cleanUpCheckPoints(t *testing.T, dataType api.DataType, primary api.ClusterInfo) {
	configMapName := checkpoint.ConfigMapName(dataType, primary.Tenant, primary.Cluster)
	err := k8sClient.CoreV1().ConfigMaps("default").Delete(ctx, configMapName, metav1.DeleteOptions{})
	require.NoError(t, err)

}
