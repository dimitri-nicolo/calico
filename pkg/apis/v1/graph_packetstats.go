// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package v1

import "math"

// GraphByteStats contains byte statistics.
type GraphByteStats struct {
	BytesIn  int64 `json:"bytes_in"`
	BytesOut int64 `json:"bytes_out"`
}

func (p GraphByteStats) Add(p2 GraphByteStats) GraphByteStats {
	p.BytesIn += p2.BytesIn
	p.BytesOut += p2.BytesOut
	return p
}

// GraphPacketStats contains packet and byte statistics.
type GraphPacketStats struct {
	PacketsIn  int64 `json:"packet_in"`
	PacketsOut int64 `json:"packet_out"`
	BytesIn    int64 `json:"bytes_in"`
	BytesOut   int64 `json:"bytes_out"`
}

// Add returns a pointer to a GraphPacketStats that contains the sum of the stats: p+p2.
func (p *GraphPacketStats) Add(p2 *GraphPacketStats) *GraphPacketStats {
	if p == nil {
		return p2
	} else if p2 == nil {
		return p
	}

	return &GraphPacketStats{
		PacketsIn:  p.PacketsIn + p2.PacketsIn,
		PacketsOut: p.PacketsOut + p2.PacketsOut,
		BytesIn:    p.BytesIn + p2.BytesIn,
		BytesOut:   p.BytesOut + p2.BytesOut,
	}
}

// Sub returns a pointer to a GraphPacketStats that contains the difference in the stats: p-p2.
func (p *GraphPacketStats) Sub(p2 *GraphPacketStats) *GraphPacketStats {
	if p2 == nil {
		return p
	} else if p == nil {
		// We do not need to handle negative numbers for packet counts.
		return nil
	}

	return &GraphPacketStats{
		PacketsIn:  p.PacketsIn - p2.PacketsIn,
		PacketsOut: p.PacketsOut - p2.PacketsOut,
		BytesIn:    p.BytesIn - p2.BytesIn,
		BytesOut:   p.BytesOut - p2.BytesOut,
	}
}

// Prop takes the stats in p and p2 and returns a pointer to a GraphPacketStats that contains the proportional value
// of p to the sum of p and p2, i.e.  p/(p+p2)
func (p *GraphPacketStats) Prop(p2 *GraphPacketStats) *GraphPacketStats {
	if p == nil {
		return nil
	}

	n := &GraphPacketStats{}
	pt := p.Add(p2)
	if pt.PacketsIn > 0 {
		n.PacketsIn = p.PacketsIn / pt.PacketsIn
	}
	if pt.PacketsOut > 0 {
		n.PacketsOut = p.PacketsOut / pt.PacketsOut
	}
	if pt.BytesIn > 0 {
		n.BytesIn = p.BytesIn / pt.BytesIn
	}
	if pt.BytesOut > 0 {
		n.BytesOut = p.BytesOut / pt.BytesOut
	}
	return n
}

// Multiply takes the stats in p and p2 and returns a pointer to a GraphPacketStats that contains the multiple of
// p and p2:  p*p2
func (p *GraphPacketStats) Multiply(p2 *GraphPacketStats) *GraphPacketStats {
	if p == nil || p2 == nil {
		return nil
	}

	return &GraphPacketStats{
		PacketsIn:  p.PacketsIn * p2.PacketsIn,
		PacketsOut: p.PacketsOut * p2.PacketsOut,
		BytesIn:    p.BytesIn * p2.BytesIn,
		BytesOut:   p.BytesOut * p2.BytesOut,
	}
}

// GraphConnectionStats contains connection statistics.
type GraphConnectionStats struct {
	Started   int64 `json:"started"`
	Completed int64 `json:"completed"`
}

// Add returns a pointer to a GraphConnectionStats that contains the sum of the stats: c+c2.
func (c GraphConnectionStats) Add(c2 GraphConnectionStats) GraphConnectionStats {
	c.Started += c2.Started
	c.Completed += c2.Completed
	return c
}

// GraphTCPStats contains TCP statistics.
type GraphTCPStats struct {
	SumTotalRetransmissions int64 `json:"sum_total_retransmissions"`
	SumLostPackets          int64 `json:"sum_lost_packets"`
	SumUnrecoveredTo        int64 `json:"sum_unrecovered_to"`

	MinSendCongestionWindow float64 `json:"min_send_congestion_window"`
	MinSendMSS              float64 `json:"min_mss"`

	MaxSmoothRTT float64 `json:"max_smooth_rtt"`
	MaxMinRTT    float64 `json:"max_min_rtt"`

	MeanSendCongestionWindow float64 `json:"mean_send_congestion_window"`
	MeanSmoothRTT            float64 `json:"mean_smooth_rtt"`
	MeanMinRTT               float64 `json:"mean_min_mss"`
	MeanMSS                  float64 `json:"mean_mss"`

	Count int64 `json:"count"`
}

// Combine returns a pointer to a GraphPacketStats that combines the stats from t and t2. Depending on the stat
// this may either be the max or min value, the average (weighted by count) or the sum.
func (t *GraphTCPStats) Combine(t2 *GraphTCPStats) *GraphTCPStats {
	// Police against nil and zero counts so that we don't include the data in mean and min values.
	if t == nil || t.Count == 0 {
		return t2
	} else if t2 == nil || t2.Count == 0 {
		return t
	}

	totalCount := t.Count + t2.Count
	return &GraphTCPStats{
		SumTotalRetransmissions: t.SumTotalRetransmissions + t2.SumTotalRetransmissions,
		SumLostPackets:          t.SumLostPackets + t2.SumLostPackets,
		SumUnrecoveredTo:        t.SumUnrecoveredTo + t2.SumUnrecoveredTo,
		MinSendCongestionWindow: math.Min(t.MinSendCongestionWindow, t2.MinSendCongestionWindow),
		MinSendMSS:              math.Min(t.MinSendMSS, t2.MinSendMSS),
		MaxSmoothRTT:            math.Max(t.MaxSmoothRTT, t2.MaxSmoothRTT),
		MaxMinRTT:               math.Max(t.MaxMinRTT, t2.MaxMinRTT),
		MeanSendCongestionWindow: ((float64(t.Count) * t.MeanSendCongestionWindow) / float64(totalCount)) +
			((float64(t2.Count) * t2.MeanSendCongestionWindow) / float64(totalCount)),
		MeanSmoothRTT: ((float64(t.Count) * t.MeanSmoothRTT) / float64(totalCount)) +
			((float64(t2.Count) * t2.MeanSmoothRTT) / float64(totalCount)),
		MeanMinRTT: ((float64(t.Count) * t.MeanMinRTT) / float64(totalCount)) +
			((float64(t2.Count) * t2.MeanMinRTT) / float64(totalCount)),
		MeanMSS: ((float64(t.Count) * t.MeanMSS) / float64(totalCount)) +
			((float64(t2.Count) * t2.MeanMSS) / float64(totalCount)),
		Count: totalCount,
	}
}

// GraphL3Stats contains L3 statistics. Allowed, DeniedAtSource and DeniedAtDest are only included if there were flows
// that recorded these verdicts.  TCP is only included if there were some TCP flows recorded.
type GraphL3Stats struct {
	Allowed        *GraphPacketStats    `json:"allowed,omitempty"`
	DeniedAtSource *GraphPacketStats    `json:"denied_at_source,omitempty"`
	DeniedAtDest   *GraphPacketStats    `json:"denied_at_dest,omitempty"`
	Connections    GraphConnectionStats `json:"connections"`
	TCP            *GraphTCPStats       `json:"tcp,omitempty"`
}

// Combine returns a pointer to a GraphL3Stats that combines the stats from t and t2. Depending on the stat
// this may either be the max or min value, the average (weighted by count) or the sum.
func (t GraphL3Stats) Combine(t2 GraphL3Stats) GraphL3Stats {
	t.Allowed = t.Allowed.Add(t2.Allowed)
	t.DeniedAtSource = t.DeniedAtSource.Add(t2.DeniedAtSource)
	t.DeniedAtDest = t.DeniedAtDest.Add(t2.DeniedAtDest)
	t.Connections = t.Connections.Add(t2.Connections)
	t.TCP = t.TCP.Combine(t2.TCP)
	return t
}

// GraphL7PacketStats contains L7 statistics.
type GraphL7PacketStats struct {
	GraphByteStats `json:",inline"`
	MeanDuration   float64 `json:"mean_duration,omitempty"`
	MaxDuration    float64 `json:"max_duration,omitempty"`
	Count          int64   `json:"count,omitempty"`
}

// Combine returns a GraphL7PacketStats that combines the stats from l and l2. Depending on the stat
// this may either be the max or min value, the average (weighted by count) or the sum.
func (l GraphL7PacketStats) Combine(l2 GraphL7PacketStats) GraphL7PacketStats {
	// Police against zero count so that we don't include the data in mean values.
	if l.Count == 0 {
		return l2
	} else if l2.Count == 0 {
		return l
	}

	l.GraphByteStats = l.GraphByteStats.Add(l2.GraphByteStats)
	l.MaxDuration = math.Max(l.MaxDuration, l2.MaxDuration)
	totalCount := l.Count + l2.Count
	l.MeanDuration =
		((float64(l.Count) * l.MeanDuration) / float64(totalCount)) +
			((float64(l2.Count) * l2.MeanDuration) / float64(totalCount))
	l.Count = totalCount
	return l
}

// GraphL7Stats contains L7 statistics grouped by response code.
type GraphL7Stats struct {
	ResponseCode1xx GraphL7PacketStats `json:"response_code_1xx"`
	ResponseCode2xx GraphL7PacketStats `json:"response_code_2xx"`
	ResponseCode3xx GraphL7PacketStats `json:"response_code_3xx"`
	ResponseCode4xx GraphL7PacketStats `json:"response_code_4xx"`
	ResponseCode5xx GraphL7PacketStats `json:"response_code_5xx"`
}

// Combine returns a GraphL7Stats that combines the stats from l and l2. Depending on the stat
// this may either be the max or min value, the average (weighted by count) or the sum.
func (l GraphL7Stats) Combine(l2 GraphL7Stats) GraphL7Stats {
	l.ResponseCode1xx = l.ResponseCode1xx.Combine(l2.ResponseCode1xx)
	l.ResponseCode2xx = l.ResponseCode2xx.Combine(l2.ResponseCode2xx)
	l.ResponseCode3xx = l.ResponseCode3xx.Combine(l2.ResponseCode3xx)
	l.ResponseCode4xx = l.ResponseCode4xx.Combine(l2.ResponseCode4xx)
	l.ResponseCode5xx = l.ResponseCode5xx.Combine(l2.ResponseCode5xx)
	return l
}

// GraphTrafficStats contains L3 and L7 statistics.  If there were no L3 or L7 statistics recorded then the associated
// value will be nil and omitted entirely from the JSON response.
type GraphTrafficStats struct {
	L3 *GraphL3Stats `json:"l3,omitempty"`
	L7 *GraphL7Stats `json:"l7,omitempty"`
}

// Combine returns a GraphTrafficStats that combines the stats from t and t2.
func (t GraphTrafficStats) Combine(t2 GraphTrafficStats) GraphTrafficStats {
	if t.L3 == nil {
		t.L3 = t2.L3
	} else if t2.L3 != nil {
		l3 := t.L3.Combine(*t2.L3)
		t.L3 = &l3
	}

	if t.L7 == nil {
		t.L7 = t2.L7
	} else if t2.L7 != nil {
		l7 := t.L7.Combine(*t2.L7)
		t.L7 = &l7
	}
	return t
}
