package policycalc

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/net"
	"github.com/projectcalico/libcalico-go/lib/numorstring"

	"github.com/tigera/compliance/pkg/resources"
)

// This file contains most of the policy calculation tests, by explicitly testing each match criteria.
// It's a bit tedious.

var (
	typesIngress = []v3.PolicyType{v3.PolicyTypeIngress}
	typesEgress  = []v3.PolicyType{v3.PolicyTypeEgress}
)

var (
	int_1       = int(1)
	int_4       = int(4)
	int_6       = int(6)
	uint16_1000 = uint16(1000)
	uint8_17    = uint8(17)
)

var _ = Describe("Compiled tiers and policies tests", func() {
	var f *Flow
	var np *v3.NetworkPolicy
	var tiers Tiers
	var rd *ResourceData
	var modified ModifiedResources
	var sel *EndpointSelectorHandler
	var compute func() Action

	setup := func(cfg *Config) {
		np = &v3.NetworkPolicy{
			TypeMeta: resources.TypeCalicoGlobalNetworkPolicies,
			ObjectMeta: v1.ObjectMeta{
				Name: "default.policy",
			},
			Spec: v3.NetworkPolicySpec{
				Selector: "all()",
				Types:    typesEgress,
				Ingress: []v3.Rule{{
					Action: v3.Deny,
				}},
				Egress: []v3.Rule{{
					Action: v3.Deny,
				}},
			},
		}
		tiers = Tiers{{np}}
		modified = make(ModifiedResources)
		sel = NewEndpointSelectorHandler()
		rd = &ResourceData{
			Tiers:           tiers,
			Namespaces:      nil,
			ServiceAccounts: nil,
		}
		f = &Flow{
			Action: ActionAllow,
			Source: FlowEndpointData{
				Type:   EndpointTypeNet,
				Labels: map[string]string{},
			},
			Destination: FlowEndpointData{
				Type:   EndpointTypeNet,
				Labels: map[string]string{},
			},
		}

		compute = func() Action {
			compiled := newCompiledTiersAndPolicies(cfg, rd, modified, sel)

			// Tweak our flow reporter to match the policy type.
			if np.Spec.Types[0] == v3.PolicyTypeIngress {
				f.Reporter = ReporterTypeDestination
			} else {
				f.Reporter = ReporterTypeSource
			}
			f.Source.cachedSelectorResults = sel.CreateSelectorCache()
			f.Destination.cachedSelectorResults = sel.CreateSelectorCache()
			return compiled.Action(f)
		}
	}

	BeforeEach(func() {
		setup(&Config{})
	})

	// ---- ICMP/NotICMP matcher ----

	It("checking source egress deny exact match when ICMP is non-nil and protocol is ICMP", func() {
		f.Proto = &ProtoICMP
		f.Source.Namespace = "ns1"
		f.Source.Type = EndpointTypeWep
		np.Spec.Types = typesEgress
		np.Spec.Ingress = nil
		np.Spec.Egress[0].ICMP = &v3.ICMPFields{}
		Expect(compute()).To(Equal(ActionDeny))
	})

	It("checking dest ingress deny exact match deny when ICMP is non-nil and protocol is ICMP", func() {
		f.Proto = &ProtoICMP
		f.Destination.Namespace = "ns1"
		f.Destination.Type = EndpointTypeWep
		np.Spec.Types = typesIngress
		np.Spec.Egress = nil
		np.Spec.Ingress[0].ICMP = &v3.ICMPFields{}
		Expect(compute()).To(Equal(ActionDeny))
	})

	It("checking source egress deny inexact match when ICMP.Code is non-nil and protocol is ICMP", func() {
		f.Proto = &ProtoICMP
		f.Source.Namespace = "ns1"
		f.Source.Type = EndpointTypeWep
		np.Spec.Types = typesEgress
		np.Spec.Ingress = nil
		np.Spec.Egress[0].ICMP = &v3.ICMPFields{Code: &int_1}
		// Inexact deny and exact end of tier deny means overall a deny.
		Expect(compute()).To(Equal(ActionDeny))
	})

	It("checking dest ingress deny inexact match when ICMP.Code is non-nil and protocol is ICMP", func() {
		f.Proto = &ProtoICMP
		f.Destination.Namespace = "ns1"
		f.Destination.Type = EndpointTypeWep
		np.Spec.Types = typesIngress
		np.Spec.Egress = nil
		np.Spec.Ingress[0].ICMP = &v3.ICMPFields{Code: &int_1}
		// Inexact deny and exact end of tier deny means overall a deny.
		Expect(compute()).To(Equal(ActionDeny))
	})

	It("checking source egress deny inexact match when ICMP.Code is non-nil and protocol is ICMP", func() {
		f.Proto = &ProtoICMP
		f.Source.Namespace = "ns1"
		f.Source.Type = EndpointTypeWep
		np.Spec.Types = typesEgress
		np.Spec.Ingress = nil
		np.Spec.Egress[0].Action = v3.Allow
		np.Spec.Egress[0].ICMP = &v3.ICMPFields{Code: &int_1}
		// Inexact allow and exact end of tier deny means overall indeterminate.
		Expect(compute()).To(Equal(ActionIndeterminate))
	})

	It("checking source egress deny inexact match when ICMP.Code is non-nil and protocol is unknown", func() {
		f.Source.Namespace = "ns1"
		f.Source.Type = EndpointTypeWep
		np.Spec.Types = typesEgress
		np.Spec.Ingress = nil
		np.Spec.Egress[0].Action = v3.Allow
		np.Spec.Egress[0].ICMP = &v3.ICMPFields{Code: &int_1}
		// Inexact allow and exact end of tier deny means overall indeterminate.
		Expect(compute()).To(Equal(ActionIndeterminate))
	})

	It("checking source egress deny exact non-match when ICMP.Code is non-nil and protocol is not ICMP", func() {
		f.Proto = &ProtoTCP
		f.Source.Namespace = "ns1"
		f.Source.Type = EndpointTypeWep
		np.Spec.Types = typesEgress
		np.Spec.Ingress = nil
		np.Spec.Egress[0].Action = v3.Allow
		np.Spec.Egress[0].ICMP = &v3.ICMPFields{Code: &int_1}
		Expect(compute()).To(Equal(ActionDeny))
	})

	It("checking dest ingress deny inexact match when ICMP.Code is non-nil and protocol is ICMP", func() {
		f.Proto = &ProtoICMP
		f.Destination.Namespace = "ns1"
		f.Destination.Type = EndpointTypeWep
		np.Spec.Types = typesIngress
		np.Spec.Egress = nil
		np.Spec.Ingress[0].Action = v3.Allow
		np.Spec.Ingress[0].ICMP = &v3.ICMPFields{Code: &int_1}
		// Inexact allow and exact end of tier deny means overall indeterminate.
		Expect(compute()).To(Equal(ActionIndeterminate))
	})

	It("checking source egress allow inexact match when ICMP.Type is non-nil and protocol is ICMP", func() {
		f.Proto = &ProtoICMP
		f.Source.Namespace = "ns1"
		f.Source.Type = EndpointTypeWep
		np.Spec.Types = typesEgress
		np.Spec.Ingress = nil
		np.Spec.Egress[0].Action = v3.Allow
		np.Spec.Egress[0].ICMP = &v3.ICMPFields{Type: &int_1}
		// Inexact allow and exact end of tier deny means overall indeterminate.
		Expect(compute()).To(Equal(ActionIndeterminate))
	})

	It("checking dest ingress allow inexact match when ICMP.Type is non-nil and protocol is ICMP", func() {
		f.Proto = &ProtoICMP
		f.Destination.Namespace = "ns1"
		f.Destination.Type = EndpointTypeWep
		np.Spec.Types = typesIngress
		np.Spec.Egress = nil
		np.Spec.Ingress[0].Action = v3.Allow
		np.Spec.Ingress[0].ICMP = &v3.ICMPFields{Type: &int_1}
		// Inexact allow and exact end of tier deny means overall indeterminate.
		Expect(compute()).To(Equal(ActionIndeterminate))
	})

	It("checking dest ingress allow inexact match when NotICMP.Type is non-nil and protocol is ICMP", func() {
		f.Proto = &ProtoICMP
		f.Destination.Namespace = "ns1"
		f.Destination.Type = EndpointTypeWep
		np.Spec.Types = typesIngress
		np.Spec.Egress = nil
		np.Spec.Ingress[0].Action = v3.Allow
		np.Spec.Ingress[0].NotICMP = &v3.ICMPFields{Type: &int_1}
		// Inexact allow and exact end of tier deny means overall indeterminate.
		Expect(compute()).To(Equal(ActionIndeterminate))
	})

	// ---- HTTP matcher ----

	It("checking source egress deny exact match when HTTP is non-nil", func() {
		f.Source.Namespace = "ns1"
		f.Source.Type = EndpointTypeWep
		np.Spec.Types = typesEgress
		np.Spec.Ingress = nil
		np.Spec.Egress[0].HTTP = &v3.HTTPMatch{}
		Expect(compute()).To(Equal(ActionDeny))
	})

	It("checking dest ingress deny exact match deny when HTTP is non-nil", func() {
		f.Destination.Namespace = "ns1"
		f.Destination.Type = EndpointTypeWep
		np.Spec.Types = typesIngress
		np.Spec.Egress = nil
		np.Spec.Ingress[0].HTTP = &v3.HTTPMatch{}
		Expect(compute()).To(Equal(ActionDeny))
	})

	It("checking source egress deny inexact match when HTTP.Methods is non-nil", func() {
		f.Source.Namespace = "ns1"
		f.Source.Type = EndpointTypeWep
		np.Spec.Types = typesEgress
		np.Spec.Ingress = nil
		np.Spec.Egress[0].HTTP = &v3.HTTPMatch{Methods: []string{"post"}}
		// Inexact deny and exact end of tier deny means overall a deny.
		Expect(compute()).To(Equal(ActionDeny))
	})

	It("checking dest ingress deny inexact match when HTTP.Methods is non-nil", func() {
		f.Destination.Namespace = "ns1"
		f.Destination.Type = EndpointTypeWep
		np.Spec.Types = typesIngress
		np.Spec.Egress = nil
		np.Spec.Ingress[0].HTTP = &v3.HTTPMatch{Methods: []string{"post"}}
		// Inexact deny and exact end of tier deny means overall a deny.
		Expect(compute()).To(Equal(ActionDeny))
	})

	It("checking source egress deny inexact match when HTTP.Methods is non-nil", func() {
		f.Source.Namespace = "ns1"
		f.Source.Type = EndpointTypeWep
		np.Spec.Types = typesEgress
		np.Spec.Ingress = nil
		np.Spec.Egress[0].Action = v3.Allow
		np.Spec.Egress[0].HTTP = &v3.HTTPMatch{Methods: []string{"post"}}
		// Inexact allow and exact end of tier deny means overall indeterminate.
		Expect(compute()).To(Equal(ActionIndeterminate))
	})

	It("checking dest ingress deny inexact match when HTTP.Methods is non-nil", func() {
		f.Destination.Namespace = "ns1"
		f.Destination.Type = EndpointTypeWep
		np.Spec.Types = typesIngress
		np.Spec.Egress = nil
		np.Spec.Ingress[0].Action = v3.Allow
		np.Spec.Ingress[0].HTTP = &v3.HTTPMatch{Methods: []string{"post"}}
		// Inexact allow and exact end of tier deny means overall indeterminate.
		Expect(compute()).To(Equal(ActionIndeterminate))
	})

	It("checking source egress allow inexact match when HTTP.Paths is non-nil", func() {
		f.Source.Namespace = "ns1"
		f.Source.Type = EndpointTypeWep
		np.Spec.Types = typesEgress
		np.Spec.Ingress = nil
		np.Spec.Egress[0].Action = v3.Allow
		np.Spec.Egress[0].HTTP = &v3.HTTPMatch{Paths: []v3.HTTPPath{{Exact: "/url"}}}
		// Inexact allow and exact end of tier deny means overall indeterminate.
		Expect(compute()).To(Equal(ActionIndeterminate))
	})

	It("checking dest ingress allow inexact match when HTTP.Paths is non-nil", func() {
		f.Destination.Namespace = "ns1"
		f.Destination.Type = EndpointTypeWep
		np.Spec.Types = typesIngress
		np.Spec.Egress = nil
		np.Spec.Ingress[0].Action = v3.Allow
		np.Spec.Ingress[0].HTTP = &v3.HTTPMatch{Paths: []v3.HTTPPath{{Exact: "/url"}}}
		// Inexact allow and exact end of tier deny means overall indeterminate.
		Expect(compute()).To(Equal(ActionIndeterminate))
	})

	// ---- Protocol/NotProtocol matcher ----

	It("checking source egress allow exact match when Protocol is non-nil", func() {
		f.Proto = &uint8_17
		f.Source.Namespace = "ns1"
		f.Source.Type = EndpointTypeWep
		np.Spec.Types = typesEgress
		np.Spec.Ingress = nil
		p := numorstring.ProtocolFromString("UDP")
		np.Spec.Egress[0].Action = v3.Allow
		np.Spec.Egress[0].Protocol = &p
		Expect(compute()).To(Equal(ActionAllow))
	})

	It("checking source ingress allow exact match when Protocol is non-nil", func() {
		f.Proto = &uint8_17
		f.Destination.Namespace = "ns1"
		f.Destination.Type = EndpointTypeWep
		np.Spec.Types = typesIngress
		np.Spec.Egress = nil
		p := numorstring.ProtocolFromInt(17)
		np.Spec.Ingress[0].Action = v3.Allow
		np.Spec.Ingress[0].Protocol = &p
		Expect(compute()).To(Equal(ActionAllow))
	})

	It("checking source egress allow non-match when Protocol is non-nil", func() {
		f.Proto = &uint8_17
		f.Source.Namespace = "ns1"
		f.Source.Type = EndpointTypeWep
		np.Spec.Types = typesEgress
		np.Spec.Ingress = nil
		p := numorstring.ProtocolFromString("TCP")
		np.Spec.Egress[0].Action = v3.Allow
		np.Spec.Egress[0].Protocol = &p
		Expect(compute()).To(Equal(ActionDeny))
	})

	It("checking source ingress allow inexact match when Protocol is non-nil", func() {
		f.Destination.Namespace = "ns1"
		f.Destination.Type = EndpointTypeWep
		np.Spec.Types = typesIngress
		np.Spec.Egress = nil
		p := numorstring.ProtocolFromInt(17)
		np.Spec.Ingress[0].Action = v3.Allow
		np.Spec.Ingress[0].Protocol = &p
		Expect(compute()).To(Equal(ActionIndeterminate))
	})

	It("checking source ingress allow exact non-match when NotProtocol is non-nil", func() {
		f.Proto = &uint8_17
		f.Destination.Namespace = "ns1"
		f.Destination.Type = EndpointTypeWep
		np.Spec.Types = typesIngress
		np.Spec.Egress = nil
		p := numorstring.ProtocolFromInt(17)
		np.Spec.Ingress[0].Action = v3.Allow
		np.Spec.Ingress[0].NotProtocol = &p
		Expect(compute()).To(Equal(ActionDeny))
	})

	// ---- IPVersion matcher ----

	It("checking source egress allow exact match when IPVersion is non-nil", func() {
		f.IPVersion = &int_4
		f.Source.Namespace = "ns1"
		f.Source.Type = EndpointTypeWep
		np.Spec.Types = typesEgress
		np.Spec.Ingress = nil
		np.Spec.Egress[0].Action = v3.Allow
		np.Spec.Egress[0].IPVersion = &int_4
		Expect(compute()).To(Equal(ActionAllow))
	})

	It("checking source ingress allow exact match when IPVersion is non-nil", func() {
		f.IPVersion = &int_4
		f.Destination.Namespace = "ns1"
		f.Destination.Type = EndpointTypeWep
		np.Spec.Types = typesIngress
		np.Spec.Egress = nil
		np.Spec.Ingress[0].Action = v3.Allow
		np.Spec.Ingress[0].IPVersion = &int_4
		Expect(compute()).To(Equal(ActionAllow))
	})

	It("checking source egress allow non-match when IPVersion is non-nil", func() {
		f.IPVersion = &int_4
		f.Source.Namespace = "ns1"
		f.Source.Type = EndpointTypeWep
		np.Spec.Types = typesEgress
		np.Spec.Ingress = nil
		np.Spec.Egress[0].Action = v3.Allow
		np.Spec.Egress[0].IPVersion = &int_6
		Expect(compute()).To(Equal(ActionDeny))
	})

	It("checking source ingress allow inexact match when IPVersion is non-nil", func() {
		f.Destination.Namespace = "ns1"
		f.Destination.Type = EndpointTypeWep
		np.Spec.Types = typesIngress
		np.Spec.Egress = nil
		np.Spec.Ingress[0].Action = v3.Allow
		np.Spec.Ingress[0].IPVersion = &int_4
		Expect(compute()).To(Equal(ActionIndeterminate))
	})

	// ---- Source.Nets / Source.NotNets ----

	It("checking dest ingress allow exact match when Source.Nets is non-nil", func() {
		ip := net.MustParseIP("10.0.0.1")
		f.Source.IP = &ip
		f.Destination.Namespace = "ns1"
		f.Destination.Type = EndpointTypeWep
		np.Spec.Types = typesIngress
		np.Spec.Egress = nil
		np.Spec.Ingress[0].Action = v3.Allow
		np.Spec.Ingress[0].Source.Nets = []string{"10.0.0.0/16"}
		Expect(compute()).To(Equal(ActionAllow))
	})

	It("checking source egress allow exact match when Source.Nets is non-nil", func() {
		ip := net.MustParseIP("10.0.0.1")
		f.Source.IP = &ip
		f.Source.Namespace = "ns1"
		f.Source.Type = EndpointTypeWep
		np.Spec.Types = typesEgress
		np.Spec.Ingress = nil
		np.Spec.Egress[0].Action = v3.Allow
		np.Spec.Egress[0].Source.Nets = []string{"10.0.0.0/16"}
		Expect(compute()).To(Equal(ActionAllow))
	})

	It("checking source egress allow inexact match when Source.Nets is non-nil", func() {
		f.Source.Namespace = "ns1"
		f.Source.Type = EndpointTypeWep
		np.Spec.Types = typesEgress
		np.Spec.Ingress = nil
		np.Spec.Egress[0].Action = v3.Allow
		np.Spec.Egress[0].Source.Nets = []string{"10.0.0.0/16"}
		// Inexact allow and exact end of tier deny means overall indeterminate.
		Expect(compute()).To(Equal(ActionIndeterminate))
	})

	It("checking source egress allow non-match when Source.Nets is non-nil", func() {
		ip := net.MustParseIP("10.10.0.1")
		f.Source.IP = &ip
		f.Source.Namespace = "ns1"
		f.Source.Type = EndpointTypeWep
		np.Spec.Types = typesEgress
		np.Spec.Ingress = nil
		np.Spec.Egress[0].Action = v3.Allow
		np.Spec.Egress[0].Source.Nets = []string{"10.0.0.0/16"}
		Expect(compute()).To(Equal(ActionDeny))
	})

	It("checking source egress allow non-match when Source.NotNets is non-nil", func() {
		ip := net.MustParseIP("10.10.0.1")
		f.Source.IP = &ip
		f.Source.Namespace = "ns1"
		f.Source.Type = EndpointTypeWep
		np.Spec.Types = typesEgress
		np.Spec.Ingress = nil
		np.Spec.Egress[0].Action = v3.Allow
		np.Spec.Egress[0].Source.NotNets = []string{"10.0.0.0/16"}
		Expect(compute()).To(Equal(ActionAllow))
	})

	// ---- Destination.Nets / Destination.NotNets ----

	It("checking dest ingress allow exact match when Destination.Nets is non-nil", func() {
		ip := net.MustParseIP("10.0.0.1")
		f.Destination.IP = &ip
		f.Destination.Namespace = "ns1"
		f.Destination.Type = EndpointTypeWep
		np.Spec.Types = typesIngress
		np.Spec.Egress = nil
		np.Spec.Ingress[0].Action = v3.Allow
		np.Spec.Ingress[0].Destination.Nets = []string{"10.0.0.0/16"}
		Expect(compute()).To(Equal(ActionAllow))
	})

	It("checking source egress allow exact match when Destination.Nets is non-nil", func() {
		ip := net.MustParseIP("10.0.0.1")
		f.Destination.IP = &ip
		f.Source.Namespace = "ns1"
		f.Source.Type = EndpointTypeWep
		np.Spec.Types = typesEgress
		np.Spec.Ingress = nil
		np.Spec.Egress[0].Action = v3.Allow
		np.Spec.Egress[0].Destination.Nets = []string{"10.0.0.0/16"}
		Expect(compute()).To(Equal(ActionAllow))
	})

	It("checking source egress allow inexact match when Destination.Nets is non-nil", func() {
		f.Destination.Type = EndpointTypeWep
		f.Source.Namespace = "ns1"
		f.Source.Type = EndpointTypeWep
		np.Spec.Types = typesEgress
		np.Spec.Ingress = nil
		np.Spec.Egress[0].Action = v3.Allow
		np.Spec.Egress[0].Destination.Nets = []string{"10.0.0.0/16"}
		// Inexact allow and exact end of tier deny means overall indeterminate.
		Expect(compute()).To(Equal(ActionIndeterminate))
	})

	It("checking source egress allow non-match when Destination.Nets is non-nil", func() {
		ip := net.MustParseIP("10.10.0.1")
		f.Destination.IP = &ip
		f.Source.Namespace = "ns1"
		f.Source.Type = EndpointTypeWep
		np.Spec.Types = typesEgress
		np.Spec.Ingress = nil
		np.Spec.Egress[0].Action = v3.Allow
		np.Spec.Egress[0].Destination.Nets = []string{"10.0.0.0/16"}
		Expect(compute()).To(Equal(ActionDeny))
	})

	It("checking source egress allow non-match when Destination.NotNets is non-nil", func() {
		ip := net.MustParseIP("10.10.0.1")
		f.Destination.IP = &ip
		f.Source.Namespace = "ns1"
		f.Source.Type = EndpointTypeWep
		np.Spec.Types = typesEgress
		np.Spec.Ingress = nil
		np.Spec.Egress[0].Action = v3.Allow
		np.Spec.Egress[0].Destination.NotNets = []string{"10.0.0.0/16"}
		Expect(compute()).To(Equal(ActionAllow))
	})

	// ---- Source.Nets / Source.NotNets ----

	It("checking dest ingress allow exact match when Source.Nets is non-nil", func() {
		ip := net.MustParseIP("10.0.0.1")
		f.Source.IP = &ip
		f.Destination.Namespace = "ns1"
		f.Destination.Type = EndpointTypeWep
		np.Spec.Types = typesIngress
		np.Spec.Egress = nil
		np.Spec.Ingress[0].Action = v3.Allow
		np.Spec.Ingress[0].Source.Nets = []string{"10.0.0.0/16"}
		Expect(compute()).To(Equal(ActionAllow))
	})

	It("checking source egress allow exact match when Source.Nets is non-nil", func() {
		ip := net.MustParseIP("10.0.0.1")
		f.Source.IP = &ip
		f.Source.Namespace = "ns1"
		f.Source.Type = EndpointTypeWep
		np.Spec.Types = typesEgress
		np.Spec.Ingress = nil
		np.Spec.Egress[0].Action = v3.Allow
		np.Spec.Egress[0].Source.Nets = []string{"10.0.0.0/16"}
		Expect(compute()).To(Equal(ActionAllow))
	})

	It("checking source egress allow inexact match when Source.Nets is non-nil", func() {
		f.Source.Namespace = "ns1"
		f.Source.Type = EndpointTypeWep
		np.Spec.Types = typesEgress
		np.Spec.Ingress = nil
		np.Spec.Egress[0].Action = v3.Allow
		np.Spec.Egress[0].Source.Nets = []string{"10.0.0.0/16"}
		// Inexact allow and exact end of tier deny means overall indeterminate.
		Expect(compute()).To(Equal(ActionIndeterminate))
	})

	It("checking dest egress allow non-match when Source.Nets is non-nil", func() {
		ip := net.MustParseIP("10.10.0.1")
		f.Source.IP = &ip
		f.Source.Namespace = "ns1"
		f.Source.Type = EndpointTypeWep
		np.Spec.Types = typesEgress
		np.Spec.Ingress = nil
		np.Spec.Egress[0].Action = v3.Allow
		np.Spec.Egress[0].Source.Nets = []string{"10.0.0.0/16"}
		Expect(compute()).To(Equal(ActionDeny))
	})

	It("checking dest egress allow non-match when Source.NotNets is non-nil", func() {
		ip := net.MustParseIP("10.10.0.1")
		f.Source.IP = &ip
		f.Source.Namespace = "ns1"
		f.Source.Type = EndpointTypeWep
		np.Spec.Types = typesEgress
		np.Spec.Ingress = nil
		np.Spec.Egress[0].Action = v3.Allow
		np.Spec.Egress[0].Source.NotNets = []string{"10.0.0.0/16"}
		Expect(compute()).To(Equal(ActionAllow))
	})

	// ---- Destination.Ports / Destination.NotPorts ----

	It("checking dest ingress allow exact match when Destination.Ports is non-nil", func() {
		f.Destination.Port = &uint16_1000
		f.Destination.Namespace = "ns1"
		f.Destination.Type = EndpointTypeWep
		np.Spec.Types = typesIngress
		np.Spec.Egress = nil
		np.Spec.Ingress[0].Action = v3.Allow
		p, _ := numorstring.PortFromRange(999, 1000)
		np.Spec.Ingress[0].Destination.Ports = []numorstring.Port{p}
		Expect(compute()).To(Equal(ActionAllow))
	})

	It("checking dest egress allow exact match when Destination.Ports is non-nil (contains named port plus exact numerical port match)", func() {
		f.Destination.Port = &uint16_1000
		f.Source.Namespace = "ns1"
		f.Source.Type = EndpointTypeWep
		np.Spec.Types = typesEgress
		np.Spec.Ingress = nil
		np.Spec.Egress[0].Action = v3.Allow
		p1, _ := numorstring.PortFromRange(1000, 10000)
		p2, _ := numorstring.PortFromString("myport")
		np.Spec.Egress[0].Destination.Ports = []numorstring.Port{p1, p2}
		Expect(compute()).To(Equal(ActionAllow))
	})

	It("checking dest egress allow inexact match when Destination.Ports is non-nil and contains a named port only", func() {
		f.Destination.Port = &uint16_1000
		f.Source.Namespace = "ns1"
		f.Source.Type = EndpointTypeWep
		np.Spec.Types = typesEgress
		np.Spec.Ingress = nil
		np.Spec.Egress[0].Action = v3.Allow
		p, _ := numorstring.PortFromString("myport")
		np.Spec.Egress[0].Destination.Ports = []numorstring.Port{p}
		// Inexact allow and exact end of tier deny means overall indeterminate.
		Expect(compute()).To(Equal(ActionIndeterminate))
	})

	It("checking dest egress allow inexact match when Destination.Ports is non-nil and flow contains no port", func() {
		f.Source.Namespace = "ns1"
		f.Source.Type = EndpointTypeWep
		np.Spec.Types = typesEgress
		np.Spec.Ingress = nil
		np.Spec.Egress[0].Action = v3.Allow
		p, _ := numorstring.PortFromRange(1000, 10000)
		np.Spec.Egress[0].Destination.Ports = []numorstring.Port{p}
		// Inexact allow and exact end of tier deny means overall indeterminate.
		Expect(compute()).To(Equal(ActionIndeterminate))
	})

	It("checking dest egress allow non-match when Destination.Ports is non-nil", func() {
		f.Destination.Port = &uint16_1000
		f.Source.Namespace = "ns1"
		f.Source.Type = EndpointTypeWep
		np.Spec.Types = typesEgress
		np.Spec.Ingress = nil
		p, _ := numorstring.PortFromRange(1001, 10000)
		np.Spec.Egress[0].Destination.Ports = []numorstring.Port{p}
		Expect(compute()).To(Equal(ActionDeny))
	})

	It("checking dest egress allow non-match when Destination.NotPorts is non-nil", func() {
		f.Destination.Port = &uint16_1000
		f.Source.Namespace = "ns1"
		f.Source.Type = EndpointTypeWep
		np.Spec.Types = typesEgress
		np.Spec.Ingress = nil
		np.Spec.Egress[0].Action = v3.Allow
		p, _ := numorstring.PortFromRange(1001, 10000)
		np.Spec.Egress[0].Destination.NotPorts = []numorstring.Port{p}
		Expect(compute()).To(Equal(ActionAllow))
	})

	// ---- Source.Ports / Source.NotPorts ----

	It("checking source egress allow exact match when Source.Ports is non-nil", func() {
		f.Source.Port = &uint16_1000
		f.Source.Namespace = "ns1"
		f.Source.Type = EndpointTypeWep
		np.Spec.Types = typesEgress
		np.Spec.Ingress = nil
		np.Spec.Egress[0].Action = v3.Allow
		p, _ := numorstring.PortFromRange(999, 1000)
		np.Spec.Egress[0].Source.Ports = []numorstring.Port{p}
		Expect(compute()).To(Equal(ActionAllow))
	})

	It("checking source ingress allow exact match when Source.Ports is non-nil (contains named port plus exact numerical port match)", func() {
		f.Source.Port = &uint16_1000
		f.Destination.Namespace = "ns1"
		f.Destination.Type = EndpointTypeWep
		np.Spec.Types = typesIngress
		np.Spec.Egress = nil
		np.Spec.Ingress[0].Action = v3.Allow
		p1, _ := numorstring.PortFromRange(1000, 10000)
		p2, _ := numorstring.PortFromString("myport")
		np.Spec.Ingress[0].Source.Ports = []numorstring.Port{p1, p2}
		Expect(compute()).To(Equal(ActionAllow))
	})

	It("checking source ingress allow inexact match when Source.Ports is non-nil and contains a named port only", func() {
		f.Source.Port = &uint16_1000
		f.Destination.Namespace = "ns1"
		f.Destination.Type = EndpointTypeWep
		np.Spec.Types = typesIngress
		np.Spec.Egress = nil
		np.Spec.Ingress[0].Action = v3.Allow
		p, _ := numorstring.PortFromString("myport")
		np.Spec.Ingress[0].Source.Ports = []numorstring.Port{p}
		// Inexact allow and exact end of tier deny means overall indeterminate.
		Expect(compute()).To(Equal(ActionIndeterminate))
	})

	It("checking source ingress allow inexact match when Source.Ports is non-nil and flow contains no port", func() {
		f.Destination.Namespace = "ns1"
		f.Destination.Type = EndpointTypeWep
		np.Spec.Types = typesIngress
		np.Spec.Egress = nil
		np.Spec.Ingress[0].Action = v3.Allow
		p, _ := numorstring.PortFromRange(1000, 10000)
		np.Spec.Ingress[0].Source.Ports = []numorstring.Port{p}
		// Inexact allow and exact end of tier deny means overall indeterminate.
		Expect(compute()).To(Equal(ActionIndeterminate))
	})

	It("checking source ingress allow non-match when Source.Ports is non-nil", func() {
		f.Source.Port = &uint16_1000
		f.Destination.Namespace = "ns1"
		f.Destination.Type = EndpointTypeWep
		np.Spec.Types = typesIngress
		np.Spec.Egress = nil
		p, _ := numorstring.PortFromRange(1001, 10000)
		np.Spec.Ingress[0].Source.Ports = []numorstring.Port{p}
		Expect(compute()).To(Equal(ActionDeny))
	})

	It("checking source ingress allow non-match when Source.NotPorts is non-nil", func() {
		f.Source.Port = &uint16_1000
		f.Destination.Namespace = "ns1"
		f.Destination.Type = EndpointTypeWep
		np.Spec.Types = typesIngress
		np.Spec.Egress = nil
		np.Spec.Ingress[0].Action = v3.Allow
		p, _ := numorstring.PortFromRange(1001, 10000)
		np.Spec.Ingress[0].Source.NotPorts = []numorstring.Port{p}
		Expect(compute()).To(Equal(ActionAllow))
	})

	// ---- Destination.Domains ----

	It("checking source egress allow exact match when Source.Domains is non-nil but empty", func() {
		f.Source.Namespace = "ns1"
		f.Source.Type = EndpointTypeWep
		np.Spec.Types = typesEgress
		np.Spec.Ingress = nil
		np.Spec.Egress[0].Action = v3.Allow
		np.Spec.Egress[0].Destination.Domains = []string{}
		Expect(compute()).To(Equal(ActionAllow))
	})

	It("checking source egress deny inexact match when Source.Domains has domains", func() {
		f.Source.Namespace = "ns1"
		f.Source.Type = EndpointTypeWep
		np.Spec.Types = typesEgress
		np.Spec.Ingress = nil
		np.Spec.Egress[0].Action = v3.Deny
		np.Spec.Egress[0].Destination.Domains = []string{"thing.com"}
		// Inexact deny and exact end of tier deny means overall a deny.
		Expect(compute()).To(Equal(ActionDeny))
	})

	It("checking source egress allow inexact match when Source.Domains has domains", func() {
		f.Source.Namespace = "ns1"
		f.Source.Type = EndpointTypeWep
		np.Spec.Types = typesEgress
		np.Spec.Ingress = nil
		np.Spec.Egress[0].Action = v3.Allow
		np.Spec.Egress[0].Destination.Domains = []string{"thing.com"}
		// Inexact allow and exact end of tier deny means overall indeterminate.
		Expect(compute()).To(Equal(ActionIndeterminate))
	})

	It("checking dest ingress allow exact match when Source.Domains is non-nil but empty", func() {
		f.Destination.Namespace = "ns1"
		f.Destination.Type = EndpointTypeWep
		np.Spec.Types = typesIngress
		np.Spec.Egress = nil
		np.Spec.Ingress[0].Action = v3.Allow
		np.Spec.Ingress[0].Destination.Domains = []string{}
		Expect(compute()).To(Equal(ActionAllow))
	})

	It("checking dest ingress deny inexact match when Source.Domains has domains", func() {
		f.Destination.Namespace = "ns1"
		f.Destination.Type = EndpointTypeWep
		np.Spec.Types = typesIngress
		np.Spec.Egress = nil
		np.Spec.Ingress[0].Action = v3.Deny
		np.Spec.Ingress[0].Destination.Domains = []string{"thing.com"}
		// Inexact deny and exact end of tier deny means overall a deny.
		Expect(compute()).To(Equal(ActionDeny))
	})

	It("checking dest ingress allow inexact match when Source.Domains has domains", func() {
		f.Destination.Namespace = "ns1"
		f.Destination.Type = EndpointTypeWep
		np.Spec.Types = typesIngress
		np.Spec.Egress = nil
		np.Spec.Ingress[0].Action = v3.Allow
		np.Spec.Ingress[0].Destination.Domains = []string{"thing.com"}
		// Inexact allow and exact end of tier deny means overall indeterminate.
		Expect(compute()).To(Equal(ActionIndeterminate))
	})

	// ---- Source.Domains ----

	It("checking dest ingress allow exact match when Destination.Domains is non-nil but empty", func() {
		f.Destination.Namespace = "ns1"
		f.Destination.Type = EndpointTypeWep
		np.Spec.Types = typesIngress
		np.Spec.Egress = nil
		np.Spec.Ingress[0].Action = v3.Allow
		np.Spec.Ingress[0].Source.Domains = []string{}
		Expect(compute()).To(Equal(ActionAllow))
	})

	It("checking dest ingress deny inexact match when Destination.Domains has domains", func() {
		f.Destination.Namespace = "ns1"
		f.Destination.Type = EndpointTypeWep
		np.Spec.Types = typesIngress
		np.Spec.Egress = nil
		np.Spec.Ingress[0].Action = v3.Deny
		np.Spec.Ingress[0].Source.Domains = []string{"thing.com"}
		// Inexact deny and exact end of tier deny means overall a deny.
		Expect(compute()).To(Equal(ActionDeny))
	})

	It("checking dest ingress allow inexact match when Destination.Domains has domains", func() {
		f.Destination.Namespace = "ns1"
		f.Destination.Type = EndpointTypeWep
		np.Spec.Types = typesIngress
		np.Spec.Egress = nil
		np.Spec.Ingress[0].Action = v3.Allow
		np.Spec.Ingress[0].Source.Domains = []string{"thing.com"}
		// Inexact allow and exact end of tier deny means overall indeterminate.
		Expect(compute()).To(Equal(ActionIndeterminate))
	})

	It("checking source egress allow exact match when Destination.Domains is non-nil but empty", func() {
		f.Source.Namespace = "ns1"
		f.Source.Type = EndpointTypeWep
		np.Spec.Types = typesEgress
		np.Spec.Ingress = nil
		np.Spec.Egress[0].Action = v3.Allow
		np.Spec.Egress[0].Source.Domains = []string{}
		Expect(compute()).To(Equal(ActionAllow))
	})

	It("checking source egress deny inexact match when Destination.Domains has domains", func() {
		f.Source.Namespace = "ns1"
		f.Source.Type = EndpointTypeWep
		np.Spec.Types = typesEgress
		np.Spec.Ingress = nil
		np.Spec.Egress[0].Action = v3.Deny
		np.Spec.Egress[0].Source.Domains = []string{"thing.com"}
		// Inexact deny and exact end of tier deny means overall a deny.
		Expect(compute()).To(Equal(ActionDeny))
	})

	It("checking source egress allow inexact match when Destination.Domains has domains", func() {
		f.Source.Namespace = "ns1"
		f.Source.Type = EndpointTypeWep
		np.Spec.Types = typesEgress
		np.Spec.Ingress = nil
		np.Spec.Egress[0].Action = v3.Allow
		np.Spec.Egress[0].Source.Domains = []string{"thing.com"}
		// Inexact allow and exact end of tier deny means overall indeterminate.
		Expect(compute()).To(Equal(ActionIndeterminate))
	})

	// ---- Destination.ServiceAccounts ----

	It("checking dest ingress allow exact match when Destination.ServiceAccounts is non-nil but empty", func() {
		f.Destination.Namespace = "ns1"
		f.Destination.Type = EndpointTypeWep
		np.Spec.Types = typesIngress
		np.Spec.Egress = nil
		np.Spec.Ingress[0].Action = v3.Allow
		np.Spec.Ingress[0].Destination.ServiceAccounts = &v3.ServiceAccountMatch{}
		Expect(compute()).To(Equal(ActionAllow))
	})

	It("checking dest ingress allow exact match when Destination.ServiceAccounts is non-nil", func() {
		sa := "sa1"
		f.Destination.ServiceAccount = &sa
		f.Destination.Namespace = "ns1"
		f.Destination.Type = EndpointTypeWep
		np.Spec.Types = typesIngress
		np.Spec.Egress = nil
		np.Spec.Ingress[0].Action = v3.Allow
		np.Spec.Ingress[0].Destination.ServiceAccounts = &v3.ServiceAccountMatch{Names: []string{"sa1"}}
		Expect(compute()).To(Equal(ActionAllow))
	})

	It("checking source egress allow exact match when Destination.ServiceAccounts is non-nil", func() {
		sa := "sa1"
		f.Destination.Type = EndpointTypeWep
		f.Destination.ServiceAccount = &sa
		f.Source.Namespace = "ns1"
		f.Source.Type = EndpointTypeWep
		np.Spec.Types = typesEgress
		np.Spec.Ingress = nil
		np.Spec.Egress[0].Action = v3.Allow
		np.Spec.Egress[0].Destination.ServiceAccounts = &v3.ServiceAccountMatch{Names: []string{"sa1"}}
		Expect(compute()).To(Equal(ActionAllow))
	})

	It("checking source egress allow inexact match when Destination.ServiceAccounts is non-nil", func() {
		f.Destination.Type = EndpointTypeWep
		f.Source.Namespace = "ns1"
		f.Source.Type = EndpointTypeWep
		np.Spec.Types = typesEgress
		np.Spec.Ingress = nil
		np.Spec.Egress[0].Action = v3.Allow
		np.Spec.Egress[0].Destination.ServiceAccounts = &v3.ServiceAccountMatch{Names: []string{"sa1"}}
		// Inexact allow and exact end of tier deny means overall indeterminate.
		Expect(compute()).To(Equal(ActionIndeterminate))
	})

	It("checking source egress allow non-match when Destination.ServiceAccounts is non-nil", func() {
		sa := "sa2"
		f.Destination.Type = EndpointTypeWep
		f.Destination.ServiceAccount = &sa
		f.Source.Namespace = "ns1"
		f.Source.Type = EndpointTypeWep
		np.Spec.Types = typesEgress
		np.Spec.Ingress = nil
		np.Spec.Egress[0].Action = v3.Allow
		np.Spec.Egress[0].Destination.ServiceAccounts = &v3.ServiceAccountMatch{Names: []string{"sa1"}}
		Expect(compute()).To(Equal(ActionDeny))
	})

	// ---- Source.ServiceAccounts ----

	It("checking source egress allow exact match when Source.ServiceAccounts is non-nil but empty", func() {
		f.Source.Namespace = "ns1"
		f.Source.Type = EndpointTypeWep
		np.Spec.Types = typesEgress
		np.Spec.Ingress = nil
		np.Spec.Egress[0].Action = v3.Allow
		np.Spec.Egress[0].Source.ServiceAccounts = &v3.ServiceAccountMatch{}
		Expect(compute()).To(Equal(ActionAllow))
	})

	It("checking source egress allow exact match when Source.ServiceAccounts is non-nil", func() {
		sa := "sa1"
		f.Source.ServiceAccount = &sa
		f.Source.Namespace = "ns1"
		f.Source.Type = EndpointTypeWep
		np.Spec.Types = typesEgress
		np.Spec.Ingress = nil
		np.Spec.Egress[0].Action = v3.Allow
		np.Spec.Egress[0].Source.ServiceAccounts = &v3.ServiceAccountMatch{Names: []string{"sa1"}}
		Expect(compute()).To(Equal(ActionAllow))
	})

	It("checking dest ingress allow exact match when Source.ServiceAccounts is non-nil", func() {
		sa := "sa1"
		f.Source.Type = EndpointTypeWep
		f.Source.ServiceAccount = &sa
		f.Destination.Namespace = "ns1"
		f.Destination.Type = EndpointTypeWep
		np.Spec.Types = typesIngress
		np.Spec.Egress = nil
		np.Spec.Ingress[0].Action = v3.Allow
		np.Spec.Ingress[0].Source.ServiceAccounts = &v3.ServiceAccountMatch{Names: []string{"sa1"}}
		Expect(compute()).To(Equal(ActionAllow))
	})

	It("checking dest ingress allow inexact match when Source.ServiceAccounts is non-nil", func() {
		f.Source.Type = EndpointTypeWep
		f.Destination.Namespace = "ns1"
		f.Destination.Type = EndpointTypeWep
		np.Spec.Types = typesIngress
		np.Spec.Egress = nil
		np.Spec.Ingress[0].Action = v3.Allow
		np.Spec.Ingress[0].Source.ServiceAccounts = &v3.ServiceAccountMatch{Names: []string{"sa1"}}
		// Inexact allow and exact end of tier deny means overall indeterminate.
		Expect(compute()).To(Equal(ActionIndeterminate))
	})

	It("checking dest ingress allow non-match when Source.ServiceAccounts is non-nil", func() {
		sa := "sa2"
		f.Source.Type = EndpointTypeWep
		f.Source.ServiceAccount = &sa
		f.Destination.Namespace = "ns1"
		f.Destination.Type = EndpointTypeWep
		np.Spec.Types = typesIngress
		np.Spec.Egress = nil
		np.Spec.Ingress[0].Action = v3.Allow
		np.Spec.Ingress[0].Source.ServiceAccounts = &v3.ServiceAccountMatch{Names: []string{"sa1"}}
		Expect(compute()).To(Equal(ActionDeny))
	})
})
