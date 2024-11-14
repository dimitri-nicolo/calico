// Copyright (c) 2024 Tigera, Inc. All rights reserved.
package checker

import (
	"net"

	"github.com/projectcalico/calico/felix/collector/types/tuple"
)

// TupleToFlowAdapter adapts Tuple to the l4 and l7 flow interfaces for use in the matchers.
type TupleToFlowAdapter struct {
	flow *tuple.Tuple
}

func NewTupleToFlowAdapter(flow *tuple.Tuple) *TupleToFlowAdapter {
	return &TupleToFlowAdapter{flow: flow}
}

func (a *TupleToFlowAdapter) getSourceIP() net.IP {
	return a.flow.SourceNet()
}

func (a *TupleToFlowAdapter) getDestIP() net.IP {
	return a.flow.DestNet()
}

func (a *TupleToFlowAdapter) getSourcePort() int {
	return a.flow.GetSourcePort()
}

func (a *TupleToFlowAdapter) getDestPort() int {
	return a.flow.GetDestPort()
}

func (a *TupleToFlowAdapter) getProtocol() int {
	return a.flow.Proto
}

func (a *TupleToFlowAdapter) getHttpMethod() *string {
	return nil
}

func (a *TupleToFlowAdapter) getHttpPath() *string {
	return nil
}

func (a *TupleToFlowAdapter) getSourcePrincipal() *string {
	return nil
}

func (a *TupleToFlowAdapter) getDestPrincipal() *string {
	return nil
}

func (a *TupleToFlowAdapter) getSourceLabels() map[string]string {
	return nil
}

func (a *TupleToFlowAdapter) getDestLabels() map[string]string {
	return nil
}
