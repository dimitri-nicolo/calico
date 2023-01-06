package template

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/projectcalico/calico/confd/pkg/backends"
	cnet "github.com/projectcalico/calico/libcalico-go/lib/net"

	"github.com/kelseyhightower/memkv"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
)

const (
	maxFuncNameLen       = 66 //Max BIRD symbol length of 64 + 2 for bookending single quotes
	v4GlobalPeerIP1Str   = "77.0.0.1"
	v4GlobalPeerIP2Str   = "77.0.0.2"
	v6GlobalPeerIP1Str   = "7700::1"
	v6GlobalPeerIP2Str   = "7700::2"
	v4ExplicitPeerIP1Str = "44.0.0.1"
	v4ExplicitPeerIP2Str = "44.0.0.2"
	v6ExplicitPeerIP1Str = "4400::1"
	v6ExplicitPeerIP2Str = "4400::2"
)

func Test_EmitBGPFilterFunctionName(t *testing.T) {
	str := "should-not-be-truncated"
	direction := "import"
	version := "4"
	output, err := EmitBGPFilterFunctionName(str, direction, version)
	if err != nil {
		t.Errorf("Unexpected error calling EmitFunctionName(%s, %s, %s): %s", str, direction, version, err)
	}
	if len(output) > maxFuncNameLen {
		t.Errorf(`EmitFunctionName(%s, %s, %s) has length %d which is greater than the maximum allowed of %d`,
			str, direction, version, len(output), maxFuncNameLen)
	}

	str = "very-long-name-that-should-be-truncated-because-it-is-longer-than-the-max-bird-symbol-length-of-64-chars"
	output, err = EmitBGPFilterFunctionName(str, direction, version)
	if err != nil {
		t.Errorf("Unexpected error calling EmitFunctionName(%s, %s, %s): %s", str, direction, version, err)
	}
	if len(output) > maxFuncNameLen {
		t.Errorf(`EmitFunctionName(%s, %s, %s) has length %d which is greater than the maximum allowed of %d`,
			str, direction, version, len(output), maxFuncNameLen)
	}
}

func Test_EmitBIRDBGPFilterFuncs(t *testing.T) {
	testFilter := v3.BGPFilter{}
	testFilter.ObjectMeta.Name = "test-bgpfilter"
	testFilter.Spec = v3.BGPFilterSpec{
		ImportV4: []v3.BGPFilterRuleV4{
			{Action: "reject", MatchOperator: "Equal", CIDR: "44.4.0.0/16"},
		},
		ExportV4: []v3.BGPFilterRuleV4{
			{Action: "accept", MatchOperator: "In", CIDR: "77.7.0.0/16"},
		},
		ImportV6: []v3.BGPFilterRuleV6{
			{Action: "reject", MatchOperator: "NotEqual", CIDR: "7000:1::0/64"},
		},
		ExportV6: []v3.BGPFilterRuleV6{
			{Action: "accept", MatchOperator: "NotIn", CIDR: "9000:1::0/64"},
		},
	}
	expectedBIRDCfgStrV4 := []string{
		"# v4 BGPFilter test-bgpfilter",
		"function 'bgp_test-bgpfilter_importFilterV4'() {",
		"  if ( net = 44.4.0.0/16 ) then { reject; }",
		"}",
		"function 'bgp_test-bgpfilter_exportFilterV4'() {",
		"  if ( net ~ 77.7.0.0/16 ) then { accept; }",
		"}",
	}
	expectedBIRDCfgStrV6 := []string{
		"# v6 BGPFilter test-bgpfilter",
		"function 'bgp_test-bgpfilter_importFilterV6'() {",
		"  if ( net != 7000:1::0/64 ) then { reject; }",
		"}",
		"function 'bgp_test-bgpfilter_exportFilterV6'() {",
		"  if ( net !~ 9000:1::0/64 ) then { accept; }",
		"}",
	}

	jsonFilter, err := json.Marshal(testFilter)
	if err != nil {
		t.Errorf("Error formatting BGPFilter into JSON: %s", err)
	}
	kvps := []memkv.KVPair{
		{Key: "test-bgpfilter", Value: string(jsonFilter)},
	}

	v4BIRDCfgResult, err := EmitBIRDBGPFilterFuncs(kvps, 4)
	if err != nil {
		t.Errorf("Unexpected error while generating v4 BIRD BGPFilter functions: %s", err)
	}
	if !reflect.DeepEqual(v4BIRDCfgResult, expectedBIRDCfgStrV4) {
		t.Errorf("Generated v4 BIRD config differs from expectation: Generated = %s, Expected = %s",
			v4BIRDCfgResult, expectedBIRDCfgStrV4)
	}

	v6BIRDCfgResult, err := EmitBIRDBGPFilterFuncs(kvps, 6)
	if err != nil {
		t.Errorf("Unexpected error while generating v6 BIRD BGPFilter functions: %s", err)
	}
	if !reflect.DeepEqual(v6BIRDCfgResult, expectedBIRDCfgStrV6) {
		t.Errorf("Generated v6 BIRD config differs from expectation: Generated = %s, Expected = %s",
			v6BIRDCfgResult, expectedBIRDCfgStrV6)
	}
}

func resultCheckerForEmitBIRDExternalNetworkConfig(externalNetworksKVP, globalPeersKVP, explicitPeersKVP memkv.KVPairs, expected []string, t *testing.T) {
	result, err := EmitBIRDExternalNetworkConfig("dontcare", externalNetworksKVP, globalPeersKVP, explicitPeersKVP)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected did not match result.\nGenerated: %s\nExpected: %s", result, expected)
	}
}

func constructExternalNetworkKVPs(idxs []uint32, t *testing.T) memkv.KVPairs {
	var kvps memkv.KVPairs
	for i, idx := range idxs {
		testEnet := v3.ExternalNetwork{
			Spec: v3.ExternalNetworkSpec{
				RouteTableIndex: &idx,
			},
		}
		enetJSON, err := json.Marshal(testEnet)
		if err != nil {
			t.Errorf("Error marshalling ExternalNetwork into JSON: %s", err)
		}
		kvp := memkv.KVPair{
			Key:   fmt.Sprintf("test-enet-%d", i+1),
			Value: string(enetJSON),
		}
		kvps = append(kvps, kvp)
	}
	return kvps
}

func constructBGPPeerKVPs(peerIPStrs []string, enet string, port uint16, t *testing.T) memkv.KVPairs {
	var kvps memkv.KVPairs
	for _, peerIPStr := range peerIPStrs {
		peerIP := cnet.ParseIP(peerIPStr)
		peer := backends.BGPPeer{
			PeerIP:          *peerIP,
			ExternalNetwork: enet,
			Port:            port,
		}

		peerJSON, err := json.Marshal(peer)
		if err != nil {
			t.Errorf("Error marshalling peer into JSON: %s", err)
		}

		kvp := memkv.KVPair{
			Key:   "dontcare",
			Value: string(peerJSON),
		}
		kvps = append(kvps, kvp)
	}
	return kvps
}

func Test_BIRDExternalNetworkConfig_NoExternalNetworks(t *testing.T) {
	expectedEmptyBIRDCfgStr := []string{
		"# No ExternalNetworks configured",
	}

	resultCheckerForEmitBIRDExternalNetworkConfig(memkv.KVPairs{}, memkv.KVPairs{}, memkv.KVPairs{},
		expectedEmptyBIRDCfgStr, t)
}

func Test_EmitBIRDExternalNetworkConfig_EmptyAllPeers(t *testing.T) {
	routeTableIdxs := []uint32{7}
	externalNetworkKVPs := constructExternalNetworkKVPs(routeTableIdxs, t)

	expectedBIRDCfgStr := []string{
		"# ExternalNetwork test-enet-1",
		"table 'T_test-enet-1';",
		"protocol kernel 'K_test-enet-1' from kernel_template {",
		"  device routes yes;",
		"  table 'T_test-enet-1';",
		"  kernel table 7;",
		"  export filter {",
		"    print \"route: \", net, \", from, \", \", \", proto, \", \", bgp_next_hop;",
		"    reject;",
		"  };",
		"}",
	}

	resultCheckerForEmitBIRDExternalNetworkConfig(externalNetworkKVPs, memkv.KVPairs{}, memkv.KVPairs{},
		expectedBIRDCfgStr, t)
}

func Test_EmitBIRDExternalNetworkConfig_MultiplePeersSomeWithExternalNetworksSomeWithout(t *testing.T) {
	routeTableIdx1 := uint32(7)
	routeTableIdxs := []uint32{routeTableIdx1}
	externalNetworkKVPs := constructExternalNetworkKVPs(routeTableIdxs, t)

	globalPeerIPStrs1 := []string{v4GlobalPeerIP1Str, v6GlobalPeerIP1Str}
	globalPeersKVPs1 := constructBGPPeerKVPs(globalPeerIPStrs1, "NonExistentExternalNetwork", 0, t)
	globalPeerIPStrs2 := []string{v4GlobalPeerIP2Str, v6GlobalPeerIP2Str}
	globalPeersKVPs2 := constructBGPPeerKVPs(globalPeerIPStrs2, externalNetworkKVPs[0].Key, 0, t)
	globalPeersKVPs := append(globalPeersKVPs1, globalPeersKVPs2...)

	explicitPeerIPStrs1 := []string{v4ExplicitPeerIP1Str, v6ExplicitPeerIP1Str}
	explicitPeersKVPs1 := constructBGPPeerKVPs(explicitPeerIPStrs1, "", 0, t)
	explicitPeerIPStrs2 := []string{v4ExplicitPeerIP2Str, v6ExplicitPeerIP2Str}
	explicitPeersKVPs2 := constructBGPPeerKVPs(explicitPeerIPStrs2, externalNetworkKVPs[0].Key, 0, t)
	explicitPeersKVPs := append(explicitPeersKVPs1, explicitPeersKVPs2...)

	expectedBIRDCfgStr := []string{
		"# ExternalNetwork test-enet-1",
		"table 'T_test-enet-1';",
		"protocol kernel 'K_test-enet-1' from kernel_template {",
		"  device routes yes;",
		"  table 'T_test-enet-1';",
		"  kernel table 7;",
		"  export filter {",
		"    print \"route: \", net, \", from, \", \", \", proto, \", \", bgp_next_hop;",
		"    if proto = \"Global_77_0_0_2\" then accept;",
		"    if proto = \"Global_7700__2\" then accept;",
		"    if proto = \"Node_44_0_0_2\" then accept;",
		"    if proto = \"Node_4400__2\" then accept;",
		"    reject;",
		"  };",
		"}",
	}

	resultCheckerForEmitBIRDExternalNetworkConfig(externalNetworkKVPs, globalPeersKVPs, explicitPeersKVPs,
		expectedBIRDCfgStr, t)
}

func Test_EmitBIRDExternalNetworkConfig_PeersWithPorts(t *testing.T) {
	routeTableIdx1 := uint32(7)
	routeTableIdxs := []uint32{routeTableIdx1}
	externalNetworkKVPs := constructExternalNetworkKVPs(routeTableIdxs, t)

	globalPeerIPStrs := []string{v4GlobalPeerIP1Str, v6GlobalPeerIP1Str}
	globalPeersKVPs := constructBGPPeerKVPs(globalPeerIPStrs, externalNetworkKVPs[0].Key, 77, t)

	explicitPeerIPStrs := []string{v4ExplicitPeerIP1Str, v6ExplicitPeerIP1Str}
	explicitPeersKVPs := constructBGPPeerKVPs(explicitPeerIPStrs, externalNetworkKVPs[0].Key, 44, t)

	expectedBIRDCfgStr := []string{
		"# ExternalNetwork test-enet-1",
		"table 'T_test-enet-1';",
		"protocol kernel 'K_test-enet-1' from kernel_template {",
		"  device routes yes;",
		"  table 'T_test-enet-1';",
		"  kernel table 7;",
		"  export filter {",
		"    print \"route: \", net, \", from, \", \", \", proto, \", \", bgp_next_hop;",
		"    if proto = \"Global_77_0_0_1_port_77\" then accept;",
		"    if proto = \"Global_7700__1_port_77\" then accept;",
		"    if proto = \"Node_44_0_0_1_port_44\" then accept;",
		"    if proto = \"Node_4400__1_port_44\" then accept;",
		"    reject;",
		"  };",
		"}",
	}

	resultCheckerForEmitBIRDExternalNetworkConfig(externalNetworkKVPs, globalPeersKVPs, explicitPeersKVPs,
		expectedBIRDCfgStr, t)
}

func Test_EmitExternalNetworkTableName(t *testing.T) {
	str := "should-not-be-truncated"
	output, err := EmitExternalNetworkTableName(str)
	if err != nil {
		t.Errorf("Unexpected error calling EmitExternalNetworkTableName(%s): %s", str, err)
	}
	if len(output) > maxFuncNameLen {
		t.Errorf(`EmitExternalNetworkTableName(%s) has length %d which is greater than the maximum allowed of %d`,
			str, len(output), maxFuncNameLen)
	}
	expectedName := "'T_should-not-be-truncated'"
	if output != expectedName {
		t.Errorf("Expected %s to equal %s", output, expectedName)
	}

	str = "very-long-name-that-should-be-truncated-because-it-is-longer-than-the-max-bird-symbol-length-of-64-chars"
	output, err = EmitExternalNetworkTableName(str)
	if err != nil {
		t.Errorf("Unexpected error calling EmitExternalNetworkTableName(%s): %s", str, err)
	}
	if len(output) > maxFuncNameLen {
		t.Errorf(`EmitExternalNetworkTableName(%s) has length %d which is greater than the maximum allowed of %d`,
			str, len(output), maxFuncNameLen)
	}
}
