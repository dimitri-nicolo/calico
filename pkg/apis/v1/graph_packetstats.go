// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package v1

type GraphPacketStats struct {
	PacketsIn  int64 `json:"packet_in,omitempty"`
	PacketsOut int64 `json:"packet_out,omitempty"`
	BytesIn    int64 `json:"bytes_in,omitempty"`
	BytesOut   int64 `json:"bytes_out,omitempty"`
}

func (p GraphPacketStats) Add(p2 GraphPacketStats) GraphPacketStats {
	p.PacketsIn += p2.PacketsIn
	p.PacketsOut += p2.PacketsOut
	p.BytesIn += p2.BytesIn
	p.BytesOut += p2.BytesOut
	return p
}

func (p GraphPacketStats) Sub(p2 GraphPacketStats) GraphPacketStats {
	p.PacketsIn -= p2.PacketsIn
	p.PacketsOut -= p2.PacketsOut
	p.BytesIn -= p2.BytesIn
	p.BytesOut -= p2.BytesOut
	return p
}

func (p GraphPacketStats) Prop(p2 GraphPacketStats) GraphPacketStats {
	pt := p.Add(p2)
	if pt.PacketsIn > 0 {
		p.PacketsIn /= pt.PacketsIn
	}
	if pt.PacketsOut > 0 {
		p.PacketsOut /= pt.PacketsOut
	}
	if pt.BytesIn > 0 {
		p.BytesIn /= pt.BytesIn
	}
	if pt.BytesOut > 0 {
		p.BytesOut /= pt.BytesOut
	}
	return p
}

func (p GraphPacketStats) Multiply(p2 GraphPacketStats) GraphPacketStats {
	p.PacketsIn *= p2.PacketsIn
	p.PacketsOut *= p2.PacketsOut
	p.BytesIn *= p2.BytesIn
	p.BytesOut *= p2.BytesOut
	return p
}

type GraphConnectionStats struct {
	Started   int64 `json:"started"`
	Completed int64 `json:"completed"`
}

func (c GraphConnectionStats) Add(c2 GraphConnectionStats) GraphConnectionStats {
	c.Started += c2.Started
	c.Completed += c2.Completed
	return c
}

type GraphTrafficStats struct {
	Exists         bool                 `json:"exists"`
	Allowed        GraphPacketStats     `json:"allowed"`
	DeniedAtSource GraphPacketStats     `json:"denied_at_source"`
	DeniedAtDest   GraphPacketStats     `json:"denied_at_dest"`
	Connections    GraphConnectionStats `json:"connections"`
}

func (t GraphTrafficStats) Add(t2 GraphTrafficStats) GraphTrafficStats {
	t.Exists = t.Exists || t2.Exists
	t.Allowed = t.Allowed.Add(t2.Allowed)
	t.DeniedAtSource = t.DeniedAtSource.Add(t2.DeniedAtSource)
	t.DeniedAtDest = t.DeniedAtDest.Add(t2.DeniedAtDest)
	t.Connections = t.Connections.Add(t2.Connections)
	return t
}
