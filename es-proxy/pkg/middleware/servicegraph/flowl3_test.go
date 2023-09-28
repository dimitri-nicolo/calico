package servicegraph

import (
	"context"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"

	svapi "github.com/projectcalico/calico/es-proxy/pkg/apis/v1"
	lapi "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/client"
	"github.com/projectcalico/calico/linseed/pkg/client/rest"
	v1 "github.com/projectcalico/calico/lma/pkg/apis/v1"
)

func TestGetL3FlowData(t *testing.T) {
	tests := []struct {
		name        string
		inputFlows  []lapi.L3Flow
		wantL3Flows []L3Flow
		wantErr     bool
	}{
		{
			name: "Display process info for flows reported only at source",
			inputFlows: []lapi.L3Flow{
				{
					Key: lapi.L3FlowKey{
						Action:   "allow",
						Reporter: "src",
						Protocol: "tcp",
						Source: lapi.Endpoint{
							Type:           "wep",
							Name:           "",
							AggregatedName: "frontend-8475b5657d-*",
							Namespace:      "online-boutique",
							Port:           0,
						},
						Destination: lapi.Endpoint{
							Type:           "wep",
							Name:           "",
							AggregatedName: "cartservice-5d844fc8b7-*",
							Namespace:      "online-boutique",
							Port:           7070,
						},
					},
					Process: &lapi.Process{
						Name: "/src/server",
					},
					// Process stats are only reported at source
					ProcessStats: &lapi.ProcessStats{
						MaxNumNamesPerFlow: 1,
						MinNumNamesPerFlow: 1,
						MaxNumIDsPerFlow:   1,
						MinNumIDsPerFlow:   1,
					},
				},
				{
					Key: lapi.L3FlowKey{
						Action:   "allow",
						Reporter: "dst",
						Protocol: "tcp",
						Source: lapi.Endpoint{
							Type:           "wep",
							Name:           "",
							AggregatedName: "frontend-8475b5657d-*",
							Namespace:      "online-boutique",
							Port:           0,
						},
						Destination: lapi.Endpoint{
							Type:           "wep",
							Name:           "",
							AggregatedName: "cartservice-5d844fc8b7-*",
							Namespace:      "online-boutique",
							Port:           7070,
						},
					},
					Process: &lapi.Process{
						Name: "/app/cartservice",
					},
					// Process stats are missing at destination
					ProcessStats: nil,
				},
			},
			wantL3Flows: []L3Flow{
				{
					Edge: FlowEdge{
						Source: FlowEndpoint{
							Type:      "rep",
							Namespace: "online-boutique",
							Name:      "",
							NameAggr:  "frontend-8475b5657d-*",
							PortNum:   0,
						},
						Dest: FlowEndpoint{
							Type:      "rep",
							Namespace: "online-boutique",
							Name:      "",
							NameAggr:  "cartservice-5d844fc8b7-*",
							PortNum:   0,
						},
					},
					AggregatedProtoPorts: &svapi.AggregatedProtoPorts{
						ProtoPorts: []svapi.AggregatedPorts{
							{PortRanges: []svapi.PortRange{{MinPort: 7070, MaxPort: 7070}}},
						},
						NumOtherProtocols: 0,
					},
					Stats: svapi.GraphL3Stats{
						Allowed:        &svapi.GraphPacketStats{},
						DeniedAtSource: nil,
						DeniedAtDest:   nil,
						Connections: svapi.GraphConnectionStats{
							TotalPerSampleInterval: -9223372036854775808,
						},
						TCP: nil,
					},
					Processes: &svapi.GraphProcesses{
						// We expect that process stat is extract from the flow reported at source
						Source: svapi.GraphEndpointProcesses(map[string]svapi.GraphEndpointProcess{"/src/server": {
							Name:               "/src/server",
							MinNumNamesPerFlow: 1,
							MaxNumNamesPerFlow: 1,
							MinNumIDsPerFlow:   1,
							MaxNumIDsPerFlow:   1,
						}}),
						Dest: nil,
					},
				},
			},
		},
		{
			name: "Display process info for flows reported only at destination",
			inputFlows: []lapi.L3Flow{
				{
					Key: lapi.L3FlowKey{
						Action:   "allow",
						Reporter: "src",
						Protocol: "tcp",
						Source: lapi.Endpoint{
							Type:           "wep",
							Name:           "",
							AggregatedName: "frontend-8475b5657d-*",
							Namespace:      "online-boutique",
							Port:           0,
						},
						Destination: lapi.Endpoint{
							Type:           "wep",
							Name:           "",
							AggregatedName: "cartservice-5d844fc8b7-*",
							Namespace:      "online-boutique",
							Port:           7070,
						},
					},
					Process: &lapi.Process{
						Name: "/src/server",
					},
					// Process stats are missing at source
					ProcessStats: nil,
				},
				{
					Key: lapi.L3FlowKey{
						Action:   "allow",
						Reporter: "dst",
						Protocol: "tcp",
						Source: lapi.Endpoint{
							Type:           "wep",
							Name:           "",
							AggregatedName: "frontend-8475b5657d-*",
							Namespace:      "online-boutique",
							Port:           0,
						},
						Destination: lapi.Endpoint{
							Type:           "wep",
							Name:           "",
							AggregatedName: "cartservice-5d844fc8b7-*",
							Namespace:      "online-boutique",
							Port:           7070,
						},
					},
					Process: &lapi.Process{
						Name: "/app/cartservice",
					},
					// Process stats are reported only at destination
					ProcessStats: &lapi.ProcessStats{
						MaxNumNamesPerFlow: 1,
						MinNumNamesPerFlow: 1,
						MaxNumIDsPerFlow:   1,
						MinNumIDsPerFlow:   1,
					},
				},
			},
			wantL3Flows: []L3Flow{
				{
					Edge: FlowEdge{
						Source: FlowEndpoint{
							Type:      "rep",
							Namespace: "online-boutique",
							Name:      "",
							NameAggr:  "frontend-8475b5657d-*",
							PortNum:   0,
						},
						Dest: FlowEndpoint{
							Type:      "rep",
							Namespace: "online-boutique",
							Name:      "",
							NameAggr:  "cartservice-5d844fc8b7-*",
							PortNum:   0,
						},
					},
					AggregatedProtoPorts: &svapi.AggregatedProtoPorts{
						ProtoPorts: []svapi.AggregatedPorts{
							{PortRanges: []svapi.PortRange{{MinPort: 7070, MaxPort: 7070}}},
						},
						NumOtherProtocols: 0,
					},
					Stats: svapi.GraphL3Stats{
						Allowed:        &svapi.GraphPacketStats{},
						DeniedAtSource: nil,
						DeniedAtDest:   nil,
						Connections: svapi.GraphConnectionStats{
							TotalPerSampleInterval: -9223372036854775808,
						},
						TCP: nil,
					},
					Processes: &svapi.GraphProcesses{
						// We expect that process stat is extract from the flow reported at destination
						Dest: svapi.GraphEndpointProcesses(map[string]svapi.GraphEndpointProcess{"/app/cartservice": {
							Name:               "/app/cartservice",
							MinNumNamesPerFlow: 1,
							MaxNumNamesPerFlow: 1,
							MinNumIDsPerFlow:   1,
							MaxNumIDsPerFlow:   1,
						}}),
						Source: nil,
					},
				},
			},
		},
		{
			name: "Display process info for flows reported at source and destination",
			inputFlows: []lapi.L3Flow{
				{
					Key: lapi.L3FlowKey{
						Action:   "allow",
						Reporter: "src",
						Protocol: "tcp",
						Source: lapi.Endpoint{
							Type:           "wep",
							Name:           "",
							AggregatedName: "recommendationservice-6ffb84bb94-*",
							Namespace:      "online-boutique",
							Port:           0,
						},
						Destination: lapi.Endpoint{
							Type:           "wep",
							Name:           "",
							AggregatedName: "productcatalogservice-5b9df8d49b-*",
							Namespace:      "online-boutique",
							Port:           3550,
						},
					},
					Process: &lapi.Process{
						Name: "/usr/local/bin/python",
					},
					ProcessStats: &lapi.ProcessStats{
						MaxNumNamesPerFlow: 1,
						MinNumNamesPerFlow: 1,
						MaxNumIDsPerFlow:   1,
						MinNumIDsPerFlow:   1,
					},
				},
				{
					Key: lapi.L3FlowKey{
						Action:   "allow",
						Reporter: "dst",
						Protocol: "tcp",
						Source: lapi.Endpoint{
							Type:           "wep",
							Name:           "",
							AggregatedName: "recommendationservice-6ffb84bb94-*",
							Namespace:      "online-boutique",
							Port:           0,
						},
						Destination: lapi.Endpoint{
							Type:           "wep",
							Name:           "",
							AggregatedName: "productcatalogservice-5b9df8d49b-*",
							Namespace:      "online-boutique",
							Port:           3550,
						},
					},
					Process: &lapi.Process{
						Name: "/src/server",
					},
					ProcessStats: &lapi.ProcessStats{
						MaxNumNamesPerFlow: 1,
						MinNumNamesPerFlow: 1,
						MaxNumIDsPerFlow:   1,
						MinNumIDsPerFlow:   1,
					},
				},
			},
			wantL3Flows: []L3Flow{
				{
					Edge: FlowEdge{
						Source: FlowEndpoint{
							Type:      "rep",
							Namespace: "online-boutique",
							Name:      "",
							NameAggr:  "recommendationservice-6ffb84bb94-*",
							PortNum:   0,
						},
						Dest: FlowEndpoint{
							Type:      "rep",
							Namespace: "online-boutique",
							Name:      "",
							NameAggr:  "productcatalogservice-5b9df8d49b-*",
							PortNum:   0,
						},
					},
					AggregatedProtoPorts: &svapi.AggregatedProtoPorts{
						ProtoPorts: []svapi.AggregatedPorts{
							{PortRanges: []svapi.PortRange{{MinPort: 3550, MaxPort: 3550}}},
						},
						NumOtherProtocols: 0,
					},
					Stats: svapi.GraphL3Stats{
						Allowed:        &svapi.GraphPacketStats{},
						DeniedAtSource: nil,
						DeniedAtDest:   nil,
						Connections: svapi.GraphConnectionStats{
							TotalPerSampleInterval: -9223372036854775808,
						},
						TCP: nil,
					},
					Processes: &svapi.GraphProcesses{
						// We expect that process stat is extract from the flow reported at source
						Source: svapi.GraphEndpointProcesses(map[string]svapi.GraphEndpointProcess{"/usr/local/bin/python": {
							Name:               "/usr/local/bin/python",
							MinNumNamesPerFlow: 1,
							MaxNumNamesPerFlow: 1,
							MinNumIDsPerFlow:   1,
							MaxNumIDsPerFlow:   1,
						}}),
						Dest: svapi.GraphEndpointProcesses(map[string]svapi.GraphEndpointProcess{"/src/server": {
							Name:               "/src/server",
							MinNumNamesPerFlow: 1,
							MaxNumNamesPerFlow: 1,
							MinNumIDsPerFlow:   1,
							MaxNumIDsPerFlow:   1,
						}}),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := []rest.MockResult{}
			results = append(results, rest.MockResult{
				Body: lapi.List[lapi.L3Flow]{
					Items: tt.inputFlows,
				},
			})

			// mock linseed client
			lsc := client.NewMockClient("", results...)

			gotFs, err := GetL3FlowData(context.TODO(), lsc, "any", v1.TimeRange{}, &FlowConfig{}, &Config{})
			if (err != nil) != tt.wantErr {
				t.Errorf("GetL3FlowData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotFs, tt.wantL3Flows) {
				t.Logf(cmp.Diff(gotFs, tt.wantL3Flows))
				t.Errorf("GetL3FlowData() gotFs = %v, want %v", gotFs, tt.wantL3Flows)
			}
		})
	}
}
