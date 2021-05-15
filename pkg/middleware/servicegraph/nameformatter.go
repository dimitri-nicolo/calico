// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package servicegraph

import (
	"context"

	log "github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/tigera/es-proxy/pkg/apis/v1"
	"github.com/tigera/es-proxy/pkg/middleware/k8s"
)

type NameFormatter struct {
	nodes map[string]string
}

func GetNameFormatter(ctx context.Context, client k8s.ClientSet, sgv v1.GraphView) (*NameFormatter, error) {
	if len(sgv.HostAggregationSelectors) == 0 {
		return nil, nil
	}

	nodes, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	nf := &NameFormatter{
		nodes: make(map[string]string),
	}
	for _, node := range nodes.Items {
		for aggrName, s := range sgv.HostAggregationSelectors {
			if s.Evaluate(node.Labels) {
				log.Debugf("Host to aggregated name mapping: %s -> %s", node.Name, aggrName)
				nf.nodes[node.Name] = aggrName
				break
			}
		}
	}

	return nf, nil
}

func (nf *NameFormatter) UpdateL3Flow(f *L3Flow) {
	if nf == nil {
		return
	}
	if f.Edge.Source.Type == v1.GraphNodeTypeHostEndpoint {
		if nameAggr := nf.nodes[f.Edge.Source.Name]; nameAggr != "" {
			f.Edge.Source.NameAggr = nameAggr
		}
	}
	if f.Edge.Dest.Type == v1.GraphNodeTypeHostEndpoint {
		if nameAggr := nf.nodes[f.Edge.Dest.Name]; nameAggr != "" {
			f.Edge.Dest.NameAggr = nameAggr
		}
	}
}

func (nf *NameFormatter) UpdateL7Flow(f *L7Flow) {
	if nf == nil {
		return
	}
	if f.Edge.Source.Type == v1.GraphNodeTypeHostEndpoint {
		if nameAggr := nf.nodes[f.Edge.Source.Name]; nameAggr != "" {
			f.Edge.Source.NameAggr = nameAggr
		}
	}
	if f.Edge.Dest.Type == v1.GraphNodeTypeHostEndpoint {
		if nameAggr := nf.nodes[f.Edge.Dest.Name]; nameAggr != "" {
			f.Edge.Dest.NameAggr = nameAggr
		}
	}
}

func (nf *NameFormatter) UpdateEvent(e *EventID) {
	if nf == nil {
		return
	}
	for i := range e.EventEndpoints {
		if e.EventEndpoints[i].Type == v1.GraphNodeTypeHostEndpoint {
			if nameAggr := nf.nodes[e.EventEndpoints[i].Name]; nameAggr != "" {
				e.EventEndpoints[i].NameAggr = nameAggr
			}
		}
	}
}
