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
			name: "external-to-external",
			primary: api.ClusterInfo{
				Cluster: backendutils.RandomClusterName(),
				Tenant:  backendutils.RandomTenantName(),
			},
			secondary: api.ClusterInfo{
				Cluster: backendutils.RandomClusterName(),
				Tenant:  backendutils.RandomTenantName(),
			},
			backend: config.BackendTypeMultiIndex,
			idx:     index.FlowLogMultiIndex,
		},
		{
			name: "internal-to-external",
			primary: api.ClusterInfo{
				Cluster: backendutils.RandomClusterName(),
			},
			secondary: api.ClusterInfo{
				Cluster: backendutils.RandomClusterName(),
				Tenant:  backendutils.RandomTenantName(),
			},
			backend: config.BackendTypeMultiIndex,
			idx:     index.FlowLogMultiIndex,
		},
	}

	Run(t, "Migrate historical data", specs, func(t *testing.T, spec TestSpec) {
		catalogue := migrator.MustGetCatalogue(esConfig, spec.backend, "DEBUG", "utility")

		logs := generateData(t, catalogue, spec.primary)
		defer cleanUpData(t, spec.primary, spec.secondary, spec.idx)

		err := backendutils.RefreshIndex(ctx, esClient, spec.idx.Index(spec.primary))
		require.NoError(t, err)

		jobName := backendutils.RandStringRunes(4)
		oiler := RunOiler(t, OilerArgs{
			PrimaryClusterID: spec.primary.Cluster,
			PrimaryTenantID:  spec.primary.Tenant,
			PrimaryBackend:   spec.backend,
			SecondClusterID:  spec.secondary.Cluster,
			SecondTenantID:   spec.secondary.Tenant,
			SecondaryBackend: spec.backend,
			DataType:         api.FlowLogs,
			JobName:          jobName,
		})

		defer func() {
			oiler.StopLogs()
			oiler.Stop()
			cleanUpCheckPoints(t, api.FlowLogs, spec.primary)
		}()

		require.True(t, oiler.ListedInDockerPS())

		err = backendutils.RefreshIndex(ctx, esClient, spec.idx.Index(spec.secondary))
		require.NoError(t, err)

		validateMigratedData(t, spec, catalogue, logs)

		last := lastGeneratedTimeFromPrimary(t, catalogue, spec.primary)
		validateMetrics(t, jobName, spec.primary, spec.secondary, int64(len(logs)), last.UnixMilli())
		validateCheckpoints(t, api.FlowLogs, spec.primary, last)
	})

	Run(t, "Migrate new data", specs, func(t *testing.T, spec TestSpec) {
		catalogue := migrator.MustGetCatalogue(esConfig, config.BackendTypeMultiIndex, "DEBUG", "utility")

		jobName := backendutils.RandStringRunes(4)
		oiler := RunOiler(t, OilerArgs{
			PrimaryClusterID: spec.primary.Cluster,
			PrimaryTenantID:  spec.primary.Tenant,
			PrimaryBackend:   spec.backend,
			SecondClusterID:  spec.secondary.Cluster,
			SecondTenantID:   spec.secondary.Tenant,
			SecondaryBackend: spec.backend,
			DataType:         api.FlowLogs,
			JobName:          jobName,
		})

		defer func() {
			oiler.StopLogs()
			oiler.Stop()
			cleanUpCheckPoints(t, api.FlowLogs, spec.primary)
		}()

		require.True(t, oiler.ListedInDockerPS())

		logs := generateData(t, catalogue, spec.primary)
		defer cleanUpData(t, spec.primary, spec.secondary, spec.idx)

		err := backendutils.RefreshIndex(ctx, esClient, spec.idx.Index(spec.primary))
		require.NoError(t, err)
		err = backendutils.RefreshIndex(ctx, esClient, spec.idx.Index(spec.secondary))
		require.NoError(t, err)

		validateMigratedData(t, spec, catalogue, logs)

		last := lastGeneratedTimeFromPrimary(t, catalogue, spec.primary)
		validateMetrics(t, jobName, spec.primary, spec.secondary, int64(len(logs)), last.UnixMilli())
		validateCheckpoints(t, api.FlowLogs, spec.primary, last)
	})
}

func validateMigratedData(t *testing.T, spec TestSpec, catalogue migrator.BackendCatalogue, logs []v1.FlowLog) {
	require.Eventually(t, func() bool {
		t.Helper()

		migratedData, err := catalogue.FlowLogBackend.List(ctx, spec.secondary, &v1.FlowLogParams{})
		if err != nil {
			logrus.WithError(err).Error("failed to list logs")
			return false
		}

		err = resetUniqueFields(migratedData, spec.secondary.Cluster)
		if err != nil {
			logrus.WithError(err).Error("failed to reset logs")
			return false
		}

		logrus.Infof("migratedData: %d", len(migratedData.Items))
		return cmp.Equal(migratedData.Items, logs)
	}, 30*time.Second, 5*time.Millisecond)
}

func resetUniqueFields(migratedData *v1.List[v1.FlowLog], cluster string) error {
	for id := range migratedData.Items {
		// TODO: - Once the ID PR gets in this should match the ID of the previous log
		migratedData.Items[id].ID = ""
		migratedData.Items[id].GeneratedTime = nil
		if migratedData.Items[id].Cluster != cluster {
			logrus.Warnf("Items were not inserted correctly. Cluster value is set to %s", migratedData.Items[id].Cluster)
			return fmt.Errorf("cluster value is set to %s", migratedData.Items[id].Cluster)
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

func generateData(t *testing.T, catalogue migrator.BackendCatalogue, primary api.ClusterInfo) []v1.FlowLog {
	items := 100
	var logs []v1.FlowLog
	var err error
	startTime := time.Now().UTC()
	endTime := startTime.Add(5 * time.Second)
	for i := 0; i < items; i++ {
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
			Policies:                    &v1.FlowLogPolicy{AllPolicies: []string{"0|allow-tigera|kube-system/allow-tigera.cluster-dns|allow|0"}},
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
	response, err := catalogue.FlowLogBackend.Create(ctx, primary, logs)
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Zero(t, response.Failed)
	return logs
}
