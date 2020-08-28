// Copyright (c) 2020 Tigera, Inc. All rights reserved.
package bgp

import (
	"bytes"
	"testing"

	log "github.com/sirupsen/logrus"
)

const (
	outputNoPeers = `0001 BIRD v0.3.3+birdv1.6.8 ready.
0016 Access restricted
2002-name     proto    table    state  since       info
1002-static1  Static   master   up     04:40:14
1006-  Preference:     200
       Input filter:   ACCEPT
       Output filter:  REJECT
       Routes:         1 imported, 0 exported, 1 preferred
       Route change stats:     received   rejected   filtered    ignored   accepted
         Import updates:              1          0          0          0          1
         Import withdraws:            0          0        ---          0          0
         Export updates:              0          0          0        ---          0
         Export withdraws:            0        ---        ---        ---          0

1002-kernel1  Kernel   master   up     04:40:14
1006-  Preference:     10
       Input filter:   ACCEPT
       Output filter:  calico_kernel_programming
       Routes:         9 imported, 5 exported, 9 preferred
       Route change stats:     received   rejected   filtered    ignored   accepted
         Import updates:             11          0          0          0         11
         Import withdraws:            2          0        ---          0          2
         Export updates:             19         14          0        ---          5
         Export withdraws:            2        ---        ---        ---          0

1002-device1  Device   master   up     04:40:14
1006-  Preference:     240
       Input filter:   ACCEPT
       Output filter:  REJECT
       Routes:         0 imported, 0 exported, 0 preferred
       Route change stats:     received   rejected   filtered    ignored   accepted
         Import updates:              0          0          0          0          0
         Import withdraws:            0          0        ---          0          0
         Export updates:              0          0          0        ---          0
         Export withdraws:            0        ---        ---        ---          0

1002-direct1  Direct   master   up     04:40:14
1006-  Preference:     240
       Input filter:   ACCEPT
       Output filter:  REJECT
       Routes:         3 imported, 0 exported, 3 preferred
       Route change stats:     received   rejected   filtered    ignored   accepted
         Import updates:              3          0          0          0          3
         Import withdraws:            0          0        ---          0          0
         Export updates:              0          0          0        ---          0
         Export withdraws:            0        ---        ---        ---          0

1002-bfd1     BFD      master   up     04:40:14
1006-  Preference:     0
       Input filter:   ACCEPT
       Output filter:  REJECT
       Routes:         0 imported, 0 exported, 0 preferred
       Route change stats:     received   rejected   filtered    ignored   accepted
         Import updates:              0          0          0          0          0
         Import withdraws:            0          0        ---          0          0
         Export updates:              0          0          0        ---          0
         Export withdraws:            0        ---        ---        ---          0
`

	outputMeshPeers = `0001 BIRD v0.3.3+birdv1.6.8 ready.
0016 Access restricted
2002-name     proto    table    state  since       info
1002-static1  Static   master   up     04:40:14
1006-  Preference:     200
       Input filter:   ACCEPT
       Output filter:  REJECT
       Routes:         1 imported, 0 exported, 1 preferred
       Route change stats:     received   rejected   filtered    ignored   accepted
         Import updates:              1          0          0          0          1
         Import withdraws:            0          0        ---          0          0
         Export updates:              0          0          0        ---          0
         Export withdraws:            0        ---        ---        ---          0

1002-kernel1  Kernel   master   up     04:40:14
1006-  Preference:     10
       Input filter:   ACCEPT
       Output filter:  calico_kernel_programming
       Routes:         9 imported, 5 exported, 9 preferred
       Route change stats:     received   rejected   filtered    ignored   accepted
         Import updates:             11          0          0          0         11
         Import withdraws:            2          0        ---          0          2
         Export updates:             19         14          0        ---          5
         Export withdraws:            2        ---        ---        ---          0

1002-device1  Device   master   up     04:40:14
1006-  Preference:     240
       Input filter:   ACCEPT
       Output filter:  REJECT
       Routes:         0 imported, 0 exported, 0 preferred
       Route change stats:     received   rejected   filtered    ignored   accepted
         Import updates:              0          0          0          0          0
         Import withdraws:            0          0        ---          0          0
         Export updates:              0          0          0        ---          0
         Export withdraws:            0        ---        ---        ---          0

1002-direct1  Direct   master   up     04:40:14
1006-  Preference:     240
       Input filter:   ACCEPT
       Output filter:  REJECT
       Routes:         3 imported, 0 exported, 3 preferred
       Route change stats:     received   rejected   filtered    ignored   accepted
         Import updates:              3          0          0          0          3
         Import withdraws:            0          0        ---          0          0
         Export updates:              0          0          0        ---          0
         Export withdraws:            0        ---        ---        ---          0

1002-bfd1     BFD      master   up     04:40:14
1006-  Preference:     0
       Input filter:   ACCEPT
       Output filter:  REJECT
       Routes:         0 imported, 0 exported, 0 preferred
       Route change stats:     received   rejected   filtered    ignored   accepted
         Import updates:              0          0          0          0          0
         Import withdraws:            0          0        ---          0          0
         Export updates:              0          0          0        ---          0
         Export withdraws:            0        ---        ---        ---          0

1002-Mesh_10_128_0_202 BGP      master   up     04:41:04    Established
1006-  Description:    Connection to BGP peer
       Preference:     100
       Input filter:   ACCEPT
       Output filter:  calico_export_to_bgp_peers
       Routes:         1 imported, 1 exported, 1 preferred
       Route change stats:     received   rejected   filtered    ignored   accepted
         Import updates:              1          0          0          0          1
         Import withdraws:            0          0        ---          0          0
         Export updates:             19          4         14        ---          1
         Export withdraws:            2        ---        ---        ---          0
       BGP state:          Established
         Neighbor address: 10.128.0.202
         Neighbor AS:      64512
         Neighbor ID:      10.128.0.202
         Neighbor caps:    refresh enhanced-refresh restart-able llgr-aware AS4 add-path-rx add-path-tx
         Session:          internal multihop AS4 add-path-rx add-path-tx
         Source address:   10.128.0.200
         Hold timer:       175/240
         Keepalive timer:  67/80

1002-Mesh_10_128_0_199 BGP      master   up     04:40:16    Established
1006-  Description:    Connection to BGP peer
       Preference:     100
       Input filter:   ACCEPT
       Output filter:  calico_export_to_bgp_peers
       Routes:         1 imported, 1 exported, 1 preferred
       Route change stats:     received   rejected   filtered    ignored   accepted
         Import updates:              1          0          0          0          1
         Import withdraws:            0          0        ---          0          0
         Export updates:             19          4         14        ---          1
         Export withdraws:            2        ---        ---        ---          0
       BGP state:          Established
         Neighbor address: 10.128.0.199
         Neighbor AS:      64512
         Neighbor ID:      10.128.0.199
         Neighbor caps:    refresh enhanced-refresh restart-able llgr-aware AS4 add-path-rx add-path-tx
         Session:          internal multihop AS4 add-path-rx add-path-tx
         Source address:   10.128.0.200
         Hold timer:       204/240
         Keepalive timer:  4/80

1002-Mesh_10_128_0_201 BGP      master   up     04:40:14    Established
1006-  Description:    Connection to BGP peer
       Preference:     100
       Input filter:   ACCEPT
       Output filter:  calico_export_to_bgp_peers
       Routes:         1 imported, 1 exported, 1 preferred
       Route change stats:     received   rejected   filtered    ignored   accepted
         Import updates:              1          0          0          0          1
         Import withdraws:            0          0        ---          0          0
         Export updates:             21          6         14        ---          1
         Export withdraws:            2        ---        ---        ---          0
       BGP state:          Established
         Neighbor address: 10.128.0.201
         Neighbor AS:      64512
         Neighbor ID:      10.128.0.201
         Neighbor caps:    refresh enhanced-refresh restart-able llgr-aware AS4 add-path-rx add-path-tx
         Session:          internal multihop AS4 add-path-rx add-path-tx
         Source address:   10.128.0.200
         Hold timer:       141/240
         Keepalive timer:  19/80

1002-Mesh_10_128_0_203 BGP      master   up     04:40:14    Established
1006-  Description:    Connection to BGP peer
       Preference:     100
       Input filter:   ACCEPT
       Output filter:  calico_export_to_bgp_peers
       Routes:         1 imported, 1 exported, 1 preferred
       Route change stats:     received   rejected   filtered    ignored   accepted
         Import updates:              1          0          0          0          1
         Import withdraws:            0          0        ---          0          0
         Export updates:             21          6         14        ---          1
         Export withdraws:            2        ---        ---        ---          0
       BGP state:          Established
         Neighbor address: 10.128.0.203
         Neighbor AS:      64512
         Neighbor ID:      10.128.0.203
         Neighbor caps:    refresh enhanced-refresh restart-able llgr-aware AS4 add-path-rx add-path-tx
         Session:          internal multihop AS4 add-path-rx add-path-tx
         Source address:   10.128.0.200
         Hold timer:       178/240
         Keepalive timer:  47/80
`

	expectedMeshPeers = `Mesh_10_128_0_202 BGP      master   up     04:41:04    Established
  Description:    Connection to BGP peer
  Preference:     100
  Input filter:   ACCEPT
  Output filter:  calico_export_to_bgp_peers
  Routes:         1 imported, 1 exported, 1 preferred
  Route change stats:     received   rejected   filtered    ignored   accepted
    Import updates:              1          0          0          0          1
    Import withdraws:            0          0        ---          0          0
    Export updates:             19          4         14        ---          1
    Export withdraws:            2        ---        ---        ---          0
  BGP state:          Established
    Neighbor address: 10.128.0.202
    Neighbor AS:      64512
    Neighbor ID:      10.128.0.202
    Neighbor caps:    refresh enhanced-refresh restart-able llgr-aware AS4 add-path-rx add-path-tx
    Session:          internal multihop AS4 add-path-rx add-path-tx
    Source address:   10.128.0.200
    Hold timer:       175/240
    Keepalive timer:  67/80

Mesh_10_128_0_199 BGP      master   up     04:40:16    Established
  Description:    Connection to BGP peer
  Preference:     100
  Input filter:   ACCEPT
  Output filter:  calico_export_to_bgp_peers
  Routes:         1 imported, 1 exported, 1 preferred
  Route change stats:     received   rejected   filtered    ignored   accepted
    Import updates:              1          0          0          0          1
    Import withdraws:            0          0        ---          0          0
    Export updates:             19          4         14        ---          1
    Export withdraws:            2        ---        ---        ---          0
  BGP state:          Established
    Neighbor address: 10.128.0.199
    Neighbor AS:      64512
    Neighbor ID:      10.128.0.199
    Neighbor caps:    refresh enhanced-refresh restart-able llgr-aware AS4 add-path-rx add-path-tx
    Session:          internal multihop AS4 add-path-rx add-path-tx
    Source address:   10.128.0.200
    Hold timer:       204/240
    Keepalive timer:  4/80

Mesh_10_128_0_201 BGP      master   up     04:40:14    Established
  Description:    Connection to BGP peer
  Preference:     100
  Input filter:   ACCEPT
  Output filter:  calico_export_to_bgp_peers
  Routes:         1 imported, 1 exported, 1 preferred
  Route change stats:     received   rejected   filtered    ignored   accepted
    Import updates:              1          0          0          0          1
    Import withdraws:            0          0        ---          0          0
    Export updates:             21          6         14        ---          1
    Export withdraws:            2        ---        ---        ---          0
  BGP state:          Established
    Neighbor address: 10.128.0.201
    Neighbor AS:      64512
    Neighbor ID:      10.128.0.201
    Neighbor caps:    refresh enhanced-refresh restart-able llgr-aware AS4 add-path-rx add-path-tx
    Session:          internal multihop AS4 add-path-rx add-path-tx
    Source address:   10.128.0.200
    Hold timer:       141/240
    Keepalive timer:  19/80

Mesh_10_128_0_203 BGP      master   up     04:40:14    Established
  Description:    Connection to BGP peer
  Preference:     100
  Input filter:   ACCEPT
  Output filter:  calico_export_to_bgp_peers
  Routes:         1 imported, 1 exported, 1 preferred
  Route change stats:     received   rejected   filtered    ignored   accepted
    Import updates:              1          0          0          0          1
    Import withdraws:            0          0        ---          0          0
    Export updates:             21          6         14        ---          1
    Export withdraws:            2        ---        ---        ---          0
  BGP state:          Established
    Neighbor address: 10.128.0.203
    Neighbor AS:      64512
    Neighbor ID:      10.128.0.203
    Neighbor caps:    refresh enhanced-refresh restart-able llgr-aware AS4 add-path-rx add-path-tx
    Session:          internal multihop AS4 add-path-rx add-path-tx
    Source address:   10.128.0.200
    Hold timer:       178/240
    Keepalive timer:  47/80
`

	outputMeshNodeGlobalPeers = `0001 BIRD v0.3.3+birdv1.6.8 ready.
0016 Access restricted
2002-name     proto    table    state  since       info
1002-static1  Static   master   up     04:40:14
1006-  Preference:     200
       Input filter:   ACCEPT
       Output filter:  REJECT
       Routes:         1 imported, 0 exported, 1 preferred
       Route change stats:     received   rejected   filtered    ignored   accepted
         Import updates:              1          0          0          0          1
         Import withdraws:            0          0        ---          0          0
         Export updates:              0          0          0        ---          0
         Export withdraws:            0        ---        ---        ---          0

1002-kernel1  Kernel   master   up     04:40:14
1006-  Preference:     10
       Input filter:   ACCEPT
       Output filter:  calico_kernel_programming
       Routes:         9 imported, 5 exported, 9 preferred
       Route change stats:     received   rejected   filtered    ignored   accepted
         Import updates:             11          0          0          0         11
         Import withdraws:            2          0        ---          0          2
         Export updates:             19         14          0        ---          5
         Export withdraws:            2        ---        ---        ---          0

1002-device1  Device   master   up     04:40:14
1006-  Preference:     240
       Input filter:   ACCEPT
       Output filter:  REJECT
       Routes:         0 imported, 0 exported, 0 preferred
       Route change stats:     received   rejected   filtered    ignored   accepted
         Import updates:              0          0          0          0          0
         Import withdraws:            0          0        ---          0          0
         Export updates:              0          0          0        ---          0
         Export withdraws:            0        ---        ---        ---          0

1002-direct1  Direct   master   up     04:40:14
1006-  Preference:     240
       Input filter:   ACCEPT
       Output filter:  REJECT
       Routes:         3 imported, 0 exported, 3 preferred
       Route change stats:     received   rejected   filtered    ignored   accepted
         Import updates:              3          0          0          0          3
         Import withdraws:            0          0        ---          0          0
         Export updates:              0          0          0        ---          0
         Export withdraws:            0        ---        ---        ---          0

1002-bfd1     BFD      master   up     04:40:14
1006-  Preference:     0
       Input filter:   ACCEPT
       Output filter:  REJECT
       Routes:         0 imported, 0 exported, 0 preferred
       Route change stats:     received   rejected   filtered    ignored   accepted
         Import updates:              0          0          0          0          0
         Import withdraws:            0          0        ---          0          0
         Export updates:              0          0          0        ---          0
         Export withdraws:            0        ---        ---        ---          0

1002-Mesh_10_128_0_202 BGP      master   up     04:41:04    Established
1006-  Description:    Connection to BGP peer
       Preference:     100
       Input filter:   ACCEPT
       Output filter:  calico_export_to_bgp_peers
       Routes:         1 imported, 1 exported, 1 preferred
       Route change stats:     received   rejected   filtered    ignored   accepted
         Import updates:              1          0          0          0          1
         Import withdraws:            0          0        ---          0          0
         Export updates:             19          4         14        ---          1
         Export withdraws:            2        ---        ---        ---          0
       BGP state:          Established
         Neighbor address: 10.128.0.202
         Neighbor AS:      64512
         Neighbor ID:      10.128.0.202
         Neighbor caps:    refresh enhanced-refresh restart-able llgr-aware AS4 add-path-rx add-path-tx
         Session:          internal multihop AS4 add-path-rx add-path-tx
         Source address:   10.128.0.200
         Hold timer:       175/240
         Keepalive timer:  67/80

1002-Node_10_128_0_199 BGP      master   up     04:40:16    Established
1006-  Description:    Connection to BGP peer
       Preference:     100
       Input filter:   ACCEPT
       Output filter:  calico_export_to_bgp_peers
       Routes:         1 imported, 1 exported, 1 preferred
       Route change stats:     received   rejected   filtered    ignored   accepted
         Import updates:              1          0          0          0          1
         Import withdraws:            0          0        ---          0          0
         Export updates:             19          4         14        ---          1
         Export withdraws:            2        ---        ---        ---          0
       BGP state:          Established
         Neighbor address: 10.128.0.199
         Neighbor AS:      64512
         Neighbor ID:      10.128.0.199
         Neighbor caps:    refresh enhanced-refresh restart-able llgr-aware AS4 add-path-rx add-path-tx
         Session:          internal multihop AS4 add-path-rx add-path-tx
         Source address:   10.128.0.200
         Hold timer:       204/240
         Keepalive timer:  4/80

1002-Node_10_128_0_201 BGP      master   up     04:40:14    Established
1006-  Description:    Connection to BGP peer
       Preference:     100
       Input filter:   ACCEPT
       Output filter:  calico_export_to_bgp_peers
       Routes:         1 imported, 1 exported, 1 preferred
       Route change stats:     received   rejected   filtered    ignored   accepted
         Import updates:              1          0          0          0          1
         Import withdraws:            0          0        ---          0          0
         Export updates:             21          6         14        ---          1
         Export withdraws:            2        ---        ---        ---          0
       BGP state:          Established
         Neighbor address: 10.128.0.201
         Neighbor AS:      64512
         Neighbor ID:      10.128.0.201
         Neighbor caps:    refresh enhanced-refresh restart-able llgr-aware AS4 add-path-rx add-path-tx
         Session:          internal multihop AS4 add-path-rx add-path-tx
         Source address:   10.128.0.200
         Hold timer:       141/240
         Keepalive timer:  19/80

1002-Global_10_128_0_203 BGP      master   up     04:40:14    Established
1006-  Description:    Connection to BGP peer
       Preference:     100
       Input filter:   ACCEPT
       Output filter:  calico_export_to_bgp_peers
       Routes:         1 imported, 1 exported, 1 preferred
       Route change stats:     received   rejected   filtered    ignored   accepted
         Import updates:              1          0          0          0          1
         Import withdraws:            0          0        ---          0          0
         Export updates:             21          6         14        ---          1
         Export withdraws:            2        ---        ---        ---          0
       BGP state:          Established
         Neighbor address: 10.128.0.203
         Neighbor AS:      64512
         Neighbor ID:      10.128.0.203
         Neighbor caps:    refresh enhanced-refresh restart-able llgr-aware AS4 add-path-rx add-path-tx
         Session:          internal multihop AS4 add-path-rx add-path-tx
         Source address:   10.128.0.200
         Hold timer:       178/240
         Keepalive timer:  47/80
`

	expectedMeshNodeGlobalPeers = `Mesh_10_128_0_202 BGP      master   up     04:41:04    Established
  Description:    Connection to BGP peer
  Preference:     100
  Input filter:   ACCEPT
  Output filter:  calico_export_to_bgp_peers
  Routes:         1 imported, 1 exported, 1 preferred
  Route change stats:     received   rejected   filtered    ignored   accepted
    Import updates:              1          0          0          0          1
    Import withdraws:            0          0        ---          0          0
    Export updates:             19          4         14        ---          1
    Export withdraws:            2        ---        ---        ---          0
  BGP state:          Established
    Neighbor address: 10.128.0.202
    Neighbor AS:      64512
    Neighbor ID:      10.128.0.202
    Neighbor caps:    refresh enhanced-refresh restart-able llgr-aware AS4 add-path-rx add-path-tx
    Session:          internal multihop AS4 add-path-rx add-path-tx
    Source address:   10.128.0.200
    Hold timer:       175/240
    Keepalive timer:  67/80

Node_10_128_0_199 BGP      master   up     04:40:16    Established
  Description:    Connection to BGP peer
  Preference:     100
  Input filter:   ACCEPT
  Output filter:  calico_export_to_bgp_peers
  Routes:         1 imported, 1 exported, 1 preferred
  Route change stats:     received   rejected   filtered    ignored   accepted
    Import updates:              1          0          0          0          1
    Import withdraws:            0          0        ---          0          0
    Export updates:             19          4         14        ---          1
    Export withdraws:            2        ---        ---        ---          0
  BGP state:          Established
    Neighbor address: 10.128.0.199
    Neighbor AS:      64512
    Neighbor ID:      10.128.0.199
    Neighbor caps:    refresh enhanced-refresh restart-able llgr-aware AS4 add-path-rx add-path-tx
    Session:          internal multihop AS4 add-path-rx add-path-tx
    Source address:   10.128.0.200
    Hold timer:       204/240
    Keepalive timer:  4/80

Node_10_128_0_201 BGP      master   up     04:40:14    Established
  Description:    Connection to BGP peer
  Preference:     100
  Input filter:   ACCEPT
  Output filter:  calico_export_to_bgp_peers
  Routes:         1 imported, 1 exported, 1 preferred
  Route change stats:     received   rejected   filtered    ignored   accepted
    Import updates:              1          0          0          0          1
    Import withdraws:            0          0        ---          0          0
    Export updates:             21          6         14        ---          1
    Export withdraws:            2        ---        ---        ---          0
  BGP state:          Established
    Neighbor address: 10.128.0.201
    Neighbor AS:      64512
    Neighbor ID:      10.128.0.201
    Neighbor caps:    refresh enhanced-refresh restart-able llgr-aware AS4 add-path-rx add-path-tx
    Session:          internal multihop AS4 add-path-rx add-path-tx
    Source address:   10.128.0.200
    Hold timer:       141/240
    Keepalive timer:  19/80

Global_10_128_0_203 BGP      master   up     04:40:14    Established
  Description:    Connection to BGP peer
  Preference:     100
  Input filter:   ACCEPT
  Output filter:  calico_export_to_bgp_peers
  Routes:         1 imported, 1 exported, 1 preferred
  Route change stats:     received   rejected   filtered    ignored   accepted
    Import updates:              1          0          0          0          1
    Import withdraws:            0          0        ---          0          0
    Export updates:             21          6         14        ---          1
    Export withdraws:            2        ---        ---        ---          0
  BGP state:          Established
    Neighbor address: 10.128.0.203
    Neighbor AS:      64512
    Neighbor ID:      10.128.0.203
    Neighbor caps:    refresh enhanced-refresh restart-able llgr-aware AS4 add-path-rx add-path-tx
    Session:          internal multihop AS4 add-path-rx add-path-tx
    Source address:   10.128.0.200
    Hold timer:       178/240
    Keepalive timer:  47/80
`
)

func Test_validateAndPrint(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		expected string
	}{
		{
			name:     "Should have empty output (data contains no BGP protocols)",
			data:     outputNoPeers,
			expected: "",
		},
		{
			name:     "Should filter output to only Mesh peers",
			data:     outputMeshPeers,
			expected: expectedMeshPeers,
		},
		{
			name:     "Should filter output to only Mesh/Node/Global peers",
			data:     outputMeshNodeGlobalPeers,
			expected: expectedMeshNodeGlobalPeers,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log.Debugf("Run unit test [%s]", tt.name)
			w := &bytes.Buffer{}
			validateAndPrint(tt.data, w)
			log.Debugf("Actual result: [%s]", w.String())
			if actual := w.String(); actual != tt.expected {
				t.Errorf("validateAndPrint() = [%v], want [%v]", actual, tt.expected)
			}
		})
	}
}
