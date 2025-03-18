// Copyright (c) 2025 Tigera, Inc. All rights reserved.

//go:build fvtests

package fv

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/index"
	backendutils "github.com/projectcalico/calico/linseed/pkg/backend/testutils"
	"github.com/projectcalico/calico/linseed/pkg/config"
	"github.com/projectcalico/calico/linseed/pkg/testutils"
	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"
	"github.com/projectcalico/calico/oiler/pkg/migrator"
)

func TestRunOiler(t *testing.T) {
	specs := []TestSpec{
		{
			name:            "external-to-external",
			primaryTenant:   backendutils.RandomTenantName(),
			secondaryTenant: backendutils.RandomTenantName(),
			clusters:        []string{backendutils.RandomClusterName(), backendutils.RandomClusterName()},
			backend:         config.BackendTypeMultiIndex,
			idx:             index.FlowLogMultiIndex,
		},
		{
			name:            "internal-to-external",
			primaryTenant:   "",
			secondaryTenant: backendutils.RandomTenantName(),
			clusters:        []string{backendutils.RandomClusterName(), backendutils.RandomClusterName()},
			backend:         config.BackendTypeMultiIndex,
			idx:             index.FlowLogMultiIndex,
		},
	}

	Run(t, "Migrate historical data", specs, func(t *testing.T, spec TestSpec) {
		catalogue := migrator.MustGetCatalogue(esConfig, spec.backend, "DEBUG", "utility")

		var primaries []api.ClusterInfo
		var secondaries []api.ClusterInfo
		for _, cluster := range spec.clusters {
			primaries = append(primaries, api.ClusterInfo{
				Cluster: cluster,
				Tenant:  spec.primaryTenant,
			})
			secondaries = append(secondaries, api.ClusterInfo{
				Cluster: cluster,
				Tenant:  spec.secondaryTenant,
			})
		}

		numLogs := 100
		for idx := range spec.clusters {
			generateData(t, catalogue, numLogs, primaries[idx])
			err := backendutils.RefreshIndex(ctx, esClient, spec.idx.Index(primaries[idx]))
			require.NoError(t, err)
		}
		defer cleanUpData(t, spec.idx, append(primaries, secondaries...)...)

		jobName := backendutils.RandStringRunes(4)
		oiler := RunOiler(t, OilerArgs{
			Clusters:         spec.clusters,
			PrimaryTenantID:  spec.primaryTenant,
			PrimaryBackend:   spec.backend,
			SecondTenantID:   spec.secondaryTenant,
			SecondaryBackend: spec.backend,
			DataType:         api.FlowLogs,
			JobName:          jobName,
		})

		defer func() {
			oiler.StopLogs()
			oiler.Stop()
			cleanUpCheckPoints(api.FlowLogs, primaries...)
		}()

		require.True(t, oiler.ListedInDockerPS())

		for idx := range spec.clusters {
			err := backendutils.RefreshIndex(ctx, esClient, spec.idx.Index(secondaries[idx]))
			require.NoError(t, err)
			validateMigratedData(t, primaries[idx], secondaries[idx], catalogue, numLogs)

			last := lastGeneratedTimeFromPrimary(t, catalogue, primaries[idx])
			validateMetrics(t, jobName, primaries[idx], secondaries[idx], int64(numLogs), last.UnixMilli())
			validateCheckpoints(t, api.FlowLogs, primaries[idx], last)
		}
	})

	Run(t, "Migrate new data", specs, func(t *testing.T, spec TestSpec) {
		catalogue := migrator.MustGetCatalogue(esConfig, config.BackendTypeMultiIndex, "DEBUG", "utility")

		var primaries []api.ClusterInfo
		var secondaries []api.ClusterInfo
		for _, cluster := range spec.clusters {
			primaries = append(primaries, api.ClusterInfo{
				Cluster: cluster,
				Tenant:  spec.primaryTenant,
			})
			secondaries = append(secondaries, api.ClusterInfo{
				Cluster: cluster,
				Tenant:  spec.secondaryTenant,
			})
		}

		jobName := backendutils.RandStringRunes(4)
		oiler := RunOiler(t, OilerArgs{
			Clusters:         spec.clusters,
			PrimaryTenantID:  spec.primaryTenant,
			PrimaryBackend:   spec.backend,
			SecondTenantID:   spec.secondaryTenant,
			SecondaryBackend: spec.backend,
			DataType:         api.FlowLogs,
			JobName:          jobName,
		})

		defer func() {
			oiler.StopLogs()
			oiler.Stop()
			cleanUpCheckPoints(api.FlowLogs, primaries...)
		}()

		require.True(t, oiler.ListedInDockerPS())

		numLogs := 100
		for idx := range spec.clusters {
			generateData(t, catalogue, numLogs, primaries[idx])
			err := backendutils.RefreshIndex(ctx, esClient, spec.idx.Index(primaries[idx]))
			require.NoError(t, err)
		}
		defer cleanUpData(t, spec.idx, append(primaries, secondaries...)...)

		for idx := range spec.clusters {
			err := backendutils.RefreshIndex(ctx, esClient, spec.idx.Index(secondaries[idx]))
			require.NoError(t, err)
			validateMigratedData(t, primaries[idx], secondaries[idx], catalogue, numLogs)

			last := lastGeneratedTimeFromPrimary(t, catalogue, primaries[idx])
			validateMetrics(t, jobName, primaries[idx], secondaries[idx], int64(numLogs), last.UnixMilli())
			validateCheckpoints(t, api.FlowLogs, primaries[idx], last)
		}
	})
}

func validateMigratedData(t *testing.T, primary api.ClusterInfo, secondary api.ClusterInfo, catalogue migrator.BackendCatalogue, numLogs int) {
	require.Eventually(t, func() bool {
		t.Helper()

		originalData, err := catalogue.FlowLogBackend.List(ctx, primary, &v1.FlowLogParams{})
		if err != nil {
			logrus.WithError(err).Error("failed to list logs")
			return false
		}

		migratedData, err := catalogue.FlowLogBackend.List(ctx, secondary, &v1.FlowLogParams{})
		if err != nil {
			logrus.WithError(err).Error("failed to list logs")
			return false
		}

		err = resetUniqueFields(migratedData, secondary.Cluster)
		if err != nil {
			logrus.WithError(err).Error("failed to reset logs")
			return false
		}
		err = resetUniqueFields(originalData, primary.Cluster)
		if err != nil {
			logrus.WithError(err).Error("failed to reset logs")
			return false
		}

		if len(migratedData.Items) != numLogs {
			return false
		}

		logrus.Infof("migratedData: %d", len(migratedData.Items))
		return cmp.Equal(migratedData.Items, originalData.Items)
	}, 30*time.Second, 5*time.Millisecond)
}

func resetUniqueFields(migratedData *v1.List[v1.FlowLog], cluster string) error {
	for id := range migratedData.Items {
		migratedData.Items[id].GeneratedTime = nil
		if migratedData.Items[id].Cluster != cluster {
			logrus.Warnf("Items were not inserted correctly. Cluster value is set to %s", migratedData.Items[id].Cluster)
			return fmt.Errorf("clusters value is set to %s", migratedData.Items[id].Cluster)
		}
		migratedData.Items[id].Cluster = ""
	}
	return nil
}

func lastGeneratedTimeFromPrimary(t *testing.T, catalogue migrator.BackendCatalogue, primary api.ClusterInfo) time.Time {
	primaryData, err := catalogue.FlowLogBackend.List(ctx, primary, &v1.FlowLogParams{
		QueryParams: v1.QueryParams{
			TimeRange: &lmav1.TimeRange{
				Field: "generated_time",
			},
			MaxPageSize: 1,
		},
		QuerySortParams: v1.QuerySortParams{
			Sort: []v1.SearchRequestSortBy{{Field: "generated_time", Descending: true}},
		},
	})
	require.NoError(t, err)
	require.Len(t, primaryData.Items, 1)
	expected := *primaryData.Items[0].GeneratedTime
	return expected
}

func generateData(t *testing.T, catalogue migrator.BackendCatalogue, numLogs int, clusterInfo api.ClusterInfo) {
	var logs []v1.FlowLog
	var err error
	startTime := time.Now().UTC()
	endTime := startTime.Add(5 * time.Second)
	for i := 0; i < numLogs; i++ {
		logs = append(logs, v1.FlowLog{
			StartTime:                   startTime.Unix(),
			EndTime:                     endTime.Unix(),
			SourceIP:                    nil,
			SourceName:                  "-",
			SourceNameAggr:              "http-server-6f9cb89f44-*",
			SourceNamespace:             "source-ns",
			NatOutgoingPorts:            nil,
			SourcePort:                  nil,
			SourceType:                  "wep",
			SourceLabels:                &v1.FlowLogLabels{Labels: []string{"app.kubernetes.io/name=http-server", "k8s-app=http-server", "pod-template-hash=6f9cb89f44"}},
			DestIP:                      nil,
			DestName:                    "-",
			DestNameAggr:                "coredns-565d847f94-*",
			DestNamespace:               "kube-system",
			DestPort:                    testutils.Int64Ptr(53),
			DestType:                    "wep",
			DestLabels:                  &v1.FlowLogLabels{Labels: []string{"pod-template-hash=565d847f94", "k8s-app=kube-dns"}},
			DestServiceNamespace:        "-",
			DestServiceName:             "-",
			DestServicePortName:         "-",
			DestServicePortNum:          nil,
			DestDomains:                 nil,
			Protocol:                    "udp",
			Action:                      "allow",
			Reporter:                    "dst",
			Policies:                    &v1.FlowLogPolicy{AllPolicies: []string{"0|allow-tigera|kube-system/allow-tigera.clusters-dns|allow|0"}},
			BytesIn:                     200176,
			BytesOut:                    48235,
			NumFlows:                    1829,
			NumFlowsStarted:             1594,
			NumFlowsCompleted:           1820,
			PacketsIn:                   1635,
			PacketsOut:                  1626,
			HTTPRequestsAllowedIn:       0,
			HTTPRequestsDeniedIn:        0,
			ProcessName:                 "-",
			NumProcessNames:             0,
			ProcessID:                   "-",
			NumProcessIDs:               0,
			ProcessArgs:                 []string{""},
			NumProcessArgs:              0,
			OrigSourceIPs:               nil,
			NumOrigSourceIPs:            0,
			TCPMeanSendCongestionWindow: 0,
			TCPMinSendCongestionWindow:  0,
			TCPMeanSmoothRTT:            0,
			TCPMaxSmoothRTT:             0,
			TCPMeanMinRTT:               0,
			TCPMaxMinRTT:                0,
			TCPMeanMSS:                  0,
			TCPMinMSS:                   0,
			TCPTotalRetransmissions:     0,
			TCPLostPackets:              0,
			TCPUnrecoveredTo:            0,
			Host:                        fmt.Sprintf("flows-%d", i),
			Timestamp:                   startTime.Unix(),
		})
	}
	response, err := catalogue.FlowLogBackend.Create(ctx, clusterInfo, logs)
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Zero(t, response.Failed)
}
