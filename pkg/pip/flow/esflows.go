package flow

// The structs in this file are elastic search based flow structures
// they are unexported they are used internatlly by the flow package
// strictly for marshaling Flow objects to and from elastic search
// based flow structures

import (
	"fmt"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

const (
	EndpointTypeWep = "wep"
	EndpointTypeHep = "hep"
	EndpointTypeNs  = "ns"
	EndpointTypeNet = "net"
)

// es flow is used to marshal/unmarshal from elastic search
type es_flow struct {
	Key           es_key           `json:"key"`
	Policies      es_flow_policies `json:"policies"`
	Source_labels es_labels        `json:"source_labels"`
	Dest_labels   es_labels        `json:"dest_labels"`
}

// convert this elastic search flow to a real Flow
func (es *es_flow) toFlow() Flow {
	F := Flow{
		Reporter:      es.Key.Reporter,
		Src_type:      es.Key.Src_type,
		Src_NS:        es.Key.Src_NS,
		Src_name:      es.Key.Src_name,
		Dest_type:     es.Key.Dest_type,
		Dest_NS:       es.Key.Dest_NS,
		Dest_name:     es.Key.Dest_name,
		Dest_port:     es.Key.Dest_port,
		Action:        es.Key.Action,
		PreviewAction: es.Key.PreviewAction,
		Proto:         es.Key.Proto,
	}

	F.Src_labels = es.Source_labels.toFlowLabelMap()
	F.Dest_labels = es.Dest_labels.toFlowLabelMap()

	F.Policies = es.Policies.toFlowPolicies()

	return F
}

// populate this elastic search from from a real Flow
func (es *es_flow) fromFlow(F Flow) {
	es.Key = es_key{
		Reporter:      F.Reporter,
		Src_type:      F.Src_type,
		Src_NS:        F.Src_NS,
		Src_name:      F.Src_name,
		Dest_type:     F.Dest_type,
		Dest_NS:       F.Dest_NS,
		Dest_name:     F.Dest_name,
		Dest_port:     F.Dest_port,
		Action:        F.Action,
		PreviewAction: F.PreviewAction,
		Proto:         F.Proto,
	}
	es.Source_labels.fromFlowLabelMap(F.Src_labels)
	es.Dest_labels.fromFlowLabelMap(F.Dest_labels)
	es.Policies.fromFlowPolicies(F.Policies)
}

func (es *es_labels) toFlowLabelMap() map[string]string {
	out := make(map[string]string)
	if es.By_kvpair.Buckets != nil {
		for _, ckv := range es.By_kvpair.Buckets {
			s := strings.Split(ckv.CompositKV, "=")
			out[s[0]] = s[1]
		}
	}
	return out
}

func (es *es_labels) fromFlowLabelMap(labels map[string]string) {
	kvs := make([]es_compositKV, len(labels))
	i := 0
	for k, v := range labels {
		kvs[i].CompositKV = fmt.Sprintf("%s=%s", k, v)
		i++
	}
	es.By_kvpair.Buckets = kvs
}

func (es *es_flow_policies) toFlowPolicies() []FlowPolicy {
	out := make([]FlowPolicy, len(es.TierdPolicies.Buckets))
	i := 0
	for _, p := range es.TierdPolicies.Buckets {
		s := strings.Split(p.Key, "|")
		if len(s) != 4 {
			log.Warn("Skipping invalid flow policy ", p.Key)
			continue
		}
		o, err := strconv.ParseInt(s[0], 10, 8)
		if err != nil {
			log.Warn("Skipping invalid flow policy ", p.Key)
			continue
		}
		out[i] = FlowPolicy{
			Order:  o,
			Tier:   s[1],
			Name:   s[2],
			Action: s[3],
		}
		i++
	}
	return out
}

func (es *es_flow_policies) fromFlowPolicies(pol []FlowPolicy) {
	tfp := make([]es_flow_policy, len(pol))
	i := 0
	for _, p := range pol {
		tfp[i].Key = fmt.Sprintf("%d|%s|%s|%s", p.Order, p.Tier, p.Name, p.Action)
		i++
	}
	es.TierdPolicies.Buckets = tfp
}

// es flow sub structs -------------------------------------------------
type es_key struct {
	Reporter      string `json:"reporter"`
	Src_type      string `json:"source_type"`
	Src_NS        string `json:"source_namespace"`
	Src_name      string `json:"source_name"`
	Dest_type     string `json:"dest_type"`
	Dest_NS       string `json:"dest_namespace"`
	Dest_name     string `json:"dest_name"`
	Dest_port     string `json:"dest_port"`
	Action        string `json:"action"`
	PreviewAction string `json:"preview_action,omitempty"`
	Proto         string `json:"proto",omitempty`
}

type es_flog_buckets struct {
	Flows []es_flow `json:"buckets"`
}

type es_labels struct {
	By_kvpair es_by_kvpair `json:"by_kvpair,omitempty"`
}

type es_by_kvpair struct {
	Buckets []es_compositKV `json:"buckets"`
}

type es_compositKV struct {
	CompositKV string `json:"key"`
}

type es_flow_policies struct {
	TierdPolicies es_tiered_policies `json:"by_tiered_policy"`
}

type es_tiered_policies struct {
	Buckets []es_flow_policy `json:"buckets"`
}

type es_flow_policy struct {
	Key string `json:"key,omitempty"`
}
