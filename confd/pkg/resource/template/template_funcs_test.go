package template

import (
	"encoding/json"
	"github.com/projectcalico/calico/confd/pkg/backends"
	cnet "github.com/projectcalico/calico/libcalico-go/lib/net"
	"reflect"
	"testing"

	"github.com/kelseyhightower/memkv"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
)

const maxFuncNameLen = 66 //Max BIRD symbol length of 64 + 2 for bookending single quotes

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

func Test_BIRDExternalNetworkConfig(t *testing.T) {
	testRouteTableIdx := uint32(7)
	testEnet := v3.ExternalNetwork{
		Spec: v3.ExternalNetworkSpec{
			RouteTableIndex: &testRouteTableIdx,
		},
	}

	enetJSON, err := json.Marshal(testEnet)
	if err != nil {
		t.Errorf("Error marshalling ExternalNetwork into JSON: %s", err)
	}

	externalNetworkKVPs := memkv.KVPairs{
		memkv.KVPair{
			Key:   "/test-enet",
			Value: string(enetJSON),
		},
	}

	v4GlobalPeerIP1 := cnet.ParseIP("77.0.0.1")
	v4GlobalPeer1 := backends.BGPPeer{
		PeerIP:          *v4GlobalPeerIP1,
		ExternalNetwork: "test-enet",
	}

	v4GlobalPeerIP2 := cnet.ParseIP("77.0.0.2")
	v4GlobalPeer2 := backends.BGPPeer{
		PeerIP:          *v4GlobalPeerIP2,
		Port:            uint16(777),
		ExternalNetwork: "test-enet",
	}

	v4GlobalPeer1JSON, err := json.Marshal(v4GlobalPeer1)
	if err != nil {
		t.Errorf("Error marshalling v4 global peer 1 into JSON: %s", err)
	}

	v4GlobalPeer2JSON, err := json.Marshal(v4GlobalPeer2)
	if err != nil {
		t.Errorf("Error marshalling v4 global peer 2 into JSON: %s", err)
	}

	v4GlobalPeersKVP := memkv.KVPairs{
		memkv.KVPair{
			Key:   "dontcare",
			Value: string(v4GlobalPeer1JSON),
		},
		memkv.KVPair{
			Key:   "dontcare",
			Value: string(v4GlobalPeer2JSON),
		},
	}

	v6GlobalPeerIP1 := cnet.ParseIP("7700::1")
	v6GlobalPeer1 := backends.BGPPeer{
		PeerIP:          *v6GlobalPeerIP1,
		ExternalNetwork: "test-enet",
	}

	v6GlobalPeerIP2 := cnet.ParseIP("7700::2")
	v6GlobalPeer2 := backends.BGPPeer{
		PeerIP:          *v6GlobalPeerIP2,
		Port:            uint16(777),
		ExternalNetwork: "test-enet",
	}

	v6GlobalPeer1JSON, err := json.Marshal(v6GlobalPeer1)
	if err != nil {
		t.Errorf("Error marshalling v6 global peer 1 into JSON: %s", err)
	}

	v6GlobalPeer2JSON, err := json.Marshal(v6GlobalPeer2)
	if err != nil {
		t.Errorf("Error marshalling v6 global peer 2 into JSON: %s", err)
	}

	v6GlobalPeersKVP := memkv.KVPairs{
		memkv.KVPair{
			Key:   "dontcare",
			Value: string(v6GlobalPeer1JSON),
		},
		memkv.KVPair{
			Key:   "dontcare",
			Value: string(v6GlobalPeer2JSON),
		},
	}

	v4ExplicitPeerIP1 := cnet.ParseIP("44.0.0.1")
	v4ExplicitPeer1 := backends.BGPPeer{
		PeerIP:          *v4ExplicitPeerIP1,
		ExternalNetwork: "test-enet",
	}

	v4ExplicitPeerIP2 := cnet.ParseIP("44.0.0.2")
	v4ExplicitPeer2 := backends.BGPPeer{
		PeerIP:          *v4ExplicitPeerIP2,
		Port:            uint16(444),
		ExternalNetwork: "test-enet",
	}

	v4ExplicitPeer1JSON, err := json.Marshal(v4ExplicitPeer1)
	if err != nil {
		t.Errorf("Error marshalling v4 explicit peer 1 into JSON: %s", err)
	}

	v4ExplicitPeer2JSON, err := json.Marshal(v4ExplicitPeer2)
	if err != nil {
		t.Errorf("Error marshalling v4 explicit peer 2 into JSON: %s", err)
	}

	v4NodeSpecificPeersKVP := memkv.KVPairs{
		memkv.KVPair{
			Key:   "dontcare",
			Value: string(v4ExplicitPeer1JSON),
		},
		memkv.KVPair{
			Key:   "dontcare",
			Value: string(v4ExplicitPeer2JSON),
		},
	}

	v6ExplicitPeerIP1 := cnet.ParseIP("4400::1")
	v6ExplicitPeer1 := backends.BGPPeer{
		PeerIP:          *v6ExplicitPeerIP1,
		ExternalNetwork: "test-enet",
	}

	v6ExplicitPeerIP2 := cnet.ParseIP("4400::2")
	v6ExplicitPeer2 := backends.BGPPeer{
		PeerIP:          *v6ExplicitPeerIP2,
		Port:            uint16(444),
		ExternalNetwork: "test-enet",
	}

	v6ExplicitPeer1JSON, err := json.Marshal(v6ExplicitPeer1)
	if err != nil {
		t.Errorf("Error marshalling v6 explicit peer 1 into JSON: %s", err)
	}

	v6ExplicitPeer2JSON, err := json.Marshal(v6ExplicitPeer2)
	if err != nil {
		t.Errorf("Error marshalling v6 explicit peer 2 into JSON: %s", err)
	}

	v6NodeSpecificPeersKVP := memkv.KVPairs{
		memkv.KVPair{
			Key:   "dontcare",
			Value: string(v6ExplicitPeer1JSON),
		},
		memkv.KVPair{
			Key:   "dontcare",
			Value: string(v6ExplicitPeer2JSON),
		},
	}

	expectedEmptyBIRDCfgStr := []string{
		"# No ExternalNetworks configured",
	}

	expectedBIRDCfgStrV4 := []string{
		"# ExternalNetwork test-enet",
		"table 'T_test-enet';",
		"protocol kernel 'K_test-enet' from kernel_template {",
		"  device routes yes;",
		"  table 'T_test-enet';",
		"  kernel table 7;",
		"  export filter {",
		"    print \"route: \", net, \", from, \", \", \", proto, \", \", bgp_next_hop;",
		"    if proto = \"Global_77_0_0_1\" then accept;",
		"    if proto = \"Global_77_0_0_2_port_777\" then accept;",
		"    if proto = \"Node_44_0_0_1\" then accept;",
		"    if proto = \"Node_44_0_0_2_port_444\" then accept;",
		"    reject;",
		"  };",
		"}",
	}

	expectedBIRDCfgStrV6 := []string{
		"# ExternalNetwork test-enet",
		"table 'T_test-enet';",
		"protocol kernel 'K_test-enet' from kernel_template {",
		"  device routes yes;",
		"  table 'T_test-enet';",
		"  kernel table 7;",
		"  export filter {",
		"    print \"route: \", net, \", from, \", \", \", proto, \", \", bgp_next_hop;",
		"    if proto = \"Global_7700__1\" then accept;",
		"    if proto = \"Global_7700__2_port_777\" then accept;",
		"    if proto = \"Node_4400__1\" then accept;",
		"    if proto = \"Node_4400__2_port_444\" then accept;",
		"    reject;",
		"  };",
		"}",
	}

	emptyCfgResult, err := EmitBIRDExternalNetworkConfig("dontcare", memkv.KVPairs{}, memkv.KVPairs{}, memkv.KVPairs{})
	if err != nil {
		t.Errorf("Unexpected error while calling EmitBIRDExternalNetworkConfig on empty KVPairs: %s", err)
	}
	if !reflect.DeepEqual(emptyCfgResult, expectedEmptyBIRDCfgStr) {
		t.Errorf("Generated ExternalNetwork BIRD config when ExternalNetwork KVPairs struct is empty differs from expectation:\nGenerated: %s\nExpected = %s",
			emptyCfgResult, expectedEmptyBIRDCfgStr)
	}

	v4BIRDCfgResult, err := EmitBIRDExternalNetworkConfig("dontcare", externalNetworkKVPs, v4GlobalPeersKVP,
		v4NodeSpecificPeersKVP)
	if err != nil {
		t.Errorf("Unexpected error while generating v4 BIRD ExternalNetwork config: %s", err)
	}
	if !reflect.DeepEqual(v4BIRDCfgResult, expectedBIRDCfgStrV4) {
		t.Errorf("Generated v4 ExternalNetwork BIRD config differs from expectation:\nGenerated = %s\nExpected = %s",
			v4BIRDCfgResult, expectedBIRDCfgStrV4)
	}

	v6BIRDCfgResult, err := EmitBIRDExternalNetworkConfig("dontcare", externalNetworkKVPs, v6GlobalPeersKVP,
		v6NodeSpecificPeersKVP)
	if err != nil {
		t.Errorf("Unexpected error while generating v6 BIRD ExternalNetwork config: %s", err)
	}
	if !reflect.DeepEqual(v6BIRDCfgResult, expectedBIRDCfgStrV6) {
		t.Errorf("Generated v6 ExternalNetwork BIRD config differs from expectation:\nGenerated = %s\nExpected = %s",
			v6BIRDCfgResult, expectedBIRDCfgStrV6)
	}
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
