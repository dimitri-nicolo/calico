// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package dns_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/google/gopacket/layers"
	"github.com/stretchr/testify/require"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
)

// TestCreateDNSLog tests running a real elasticsearch query to create a DNS log.
func TestCreateDNSLog(t *testing.T) {
	defer setupTest(t)()

	clusterInfo := bapi.ClusterInfo{Cluster: cluster}

	ip := net.ParseIP("10.0.1.1")

	// Create a dummy log.
	f := v1.DNSLog{
		StartTime:       time.Now(),
		EndTime:         time.Now(),
		Type:            v1.DNSLogTypeLog,
		Count:           1,
		ClientName:      "client-name",
		ClientNameAggr:  "client-",
		ClientNamespace: "default",
		ClientIP:        &ip,
		ClientLabels:    map[string]string{"pickles": "good"},
		QName:           "qname",
		QType:           v1.DNSType(layers.DNSTypeA),
		QClass:          v1.DNSClass(layers.DNSClassIN),
		RCode:           v1.DNSResponseCode(layers.DNSResponseCodeNoErr),
		Servers: []v1.DNSServer{
			{
				Endpoint: v1.Endpoint{
					Name:           "kube-dns-one",
					AggregatedName: "kube-dns",
					Namespace:      "kube-system",
					Type:           v1.WEP,
				},
				IP:     net.ParseIP("10.0.0.10"),
				Labels: map[string]string{"app": "dns"},
			},
		},
		RRSets: v1.DNSRRSets{},
		Latency: v1.DNSLatency{
			Count: 15,
			Mean:  5 * time.Second,
			Max:   10 * time.Second,
		},
		LatencyCount: 100,
		LatencyMean:  100,
		LatencyMax:   100,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := lb.Create(ctx, clusterInfo, []v1.DNSLog{f})
	require.NoError(t, err)
	require.Empty(t, resp.Errors)
}
