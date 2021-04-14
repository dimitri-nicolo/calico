// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package v1

import "math"

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

type GraphTCPStats struct {
	SumTotalRetransmissions int64 `json:"sum_total_retransmissions,omitempty"`
	SumLostPackets          int64 `json:"sum_lost_packets,omitempty"`
	SumUnrecoveredTo        int64 `json:"sum_unrecovered_to,omitempty"`

	MinSendCongestionWindow float64 `json:"min_send_congestion_window,omitempty"`
	MinSendMSS              float64 `json:"min_mss,omitempty"`

	MaxSmoothRTT float64 `json:"max_smooth_rtt,omitempty"`
	MaxMinRTT    float64 `json:"max_min_rtt,omitempty"`

	MeanSendCongestionWindow float64 `json:"mean_send_congestion_window,omitempty"`
	MeanSmoothRTT            float64 `json:"mean_smooth_rtt,omitempty"`
	MeanMinRTT               float64 `json:"mean_min_mss,omitempty"`
	MeanMSS                  float64 `json:"mean_mss,omitempty"`

	Count int64 `json:"count,omitempty"`
}

func (t GraphTCPStats) Add(t2 GraphTCPStats) GraphTCPStats {
	// Police against zero docs so that we don't include the data in mean and min values.
	if t.Count == 0 {
		return t2
	} else if t2.Count == 0 {
		return t
	}

	t.SumTotalRetransmissions += t2.SumTotalRetransmissions
	t.SumLostPackets += t2.SumLostPackets
	t.SumUnrecoveredTo += t2.SumUnrecoveredTo
	t.MinSendCongestionWindow = math.Min(t.MinSendCongestionWindow, t2.MinSendCongestionWindow)
	t.MinSendMSS = math.Min(t.MinSendMSS, t2.MinSendMSS)
	t.MaxSmoothRTT = math.Max(t.MaxSmoothRTT, t2.MaxSmoothRTT)
	t.MaxMinRTT = math.Max(t.MaxMinRTT, t2.MaxMinRTT)

	totalCount := t.Count + t2.Count
	t.MeanSendCongestionWindow =
		((float64(t.Count) * t.MeanSendCongestionWindow) / float64(totalCount)) +
			((float64(t2.Count) * t2.MeanSendCongestionWindow) / float64(totalCount))
	t.MeanSmoothRTT =
		((float64(t.Count) * t.MeanSmoothRTT) / float64(totalCount)) +
			((float64(t2.Count) * t2.MeanSmoothRTT) / float64(totalCount))
	t.MeanMinRTT =
		((float64(t.Count) * t.MeanMinRTT) / float64(totalCount)) +
			((float64(t2.Count) * t2.MeanMinRTT) / float64(totalCount))
	t.MeanMSS =
		((float64(t.Count) * t.MeanMSS) / float64(totalCount)) +
			((float64(t2.Count) * t2.MeanMSS) / float64(totalCount))
	t.Count = totalCount

	return t
}

type GraphL3Stats struct {
	Allowed        GraphPacketStats     `json:"allowed"`
	DeniedAtSource GraphPacketStats     `json:"denied_at_source"`
	DeniedAtDest   GraphPacketStats     `json:"denied_at_dest"`
	Connections    GraphConnectionStats `json:"connections"`
	TCP            GraphTCPStats        `json:"tcp"`
}

func (t GraphL3Stats) Add(t2 GraphL3Stats) GraphL3Stats {
	t.Allowed = t.Allowed.Add(t2.Allowed)
	t.DeniedAtSource = t.DeniedAtSource.Add(t2.DeniedAtSource)
	t.DeniedAtDest = t.DeniedAtDest.Add(t2.DeniedAtDest)
	t.Connections = t.Connections.Add(t2.Connections)
	t.TCP = t.TCP.Add(t2.TCP)
	return t
}

type GraphL7PacketStats struct {
	GraphPacketStats `json:",inline"`
	MeanDuration     float64 `json:"mean_duration,omitempty"`
	MaxDuration      float64 `json:"max_duration,omitempty"`
	Count            int64   `json:"count,omitempty"`
}

func (l GraphL7PacketStats) Add(l2 GraphL7PacketStats) GraphL7PacketStats {
	// Police against zero count so that we don't include the data in mean values.
	if l.Count == 0 {
		return l2
	} else if l2.Count == 0 {
		return l
	}

	l.GraphPacketStats = l.GraphPacketStats.Add(l2.GraphPacketStats)
	l.MaxDuration = math.Max(l.MaxDuration, l2.MaxDuration)
	totalCount := l.Count + l2.Count
	l.MeanDuration =
		((float64(l.Count) * l.MeanDuration) / float64(totalCount)) +
			((float64(l2.Count) * l2.MeanDuration) / float64(totalCount))
	l.Count = totalCount
	return l
}

type GraphL7Stats struct {
	ResponseCode1xx GraphL7PacketStats `json:"response_code_1xx"`
	ResponseCode2xx GraphL7PacketStats `json:"response_code_2xx"`
	ResponseCode3xx GraphL7PacketStats `json:"response_code_3xx"`
	ResponseCode4xx GraphL7PacketStats `json:"response_code_4xx"`
	ResponseCode5xx GraphL7PacketStats `json:"response_code_5xx"`
}

func (l GraphL7Stats) Add(l2 GraphL7Stats) GraphL7Stats {
	l.ResponseCode1xx = l.ResponseCode1xx.Add(l2.ResponseCode1xx)
	l.ResponseCode2xx = l.ResponseCode2xx.Add(l2.ResponseCode2xx)
	l.ResponseCode3xx = l.ResponseCode3xx.Add(l2.ResponseCode3xx)
	l.ResponseCode4xx = l.ResponseCode4xx.Add(l2.ResponseCode4xx)
	l.ResponseCode5xx = l.ResponseCode5xx.Add(l2.ResponseCode5xx)
	return l
}

type GraphTrafficStats struct {
	L3 *GraphL3Stats `json:"l3,omitempty"`
	L7 *GraphL7Stats `json:"l7,omitempty"`
}

func (t GraphTrafficStats) Add(t2 GraphTrafficStats) GraphTrafficStats {
	if t.L3 == nil {
		t.L3 = t2.L3
	} else if t2.L3 != nil {
		l3 := t.L3.Add(*t2.L3)
		t.L3 = &l3
	}

	if t.L7 == nil {
		t.L7 = t2.L7
	} else if t2.L7 != nil {
		l7 := t.L7.Add(*t2.L7)
		t.L7 = &l7
	}
	return t
}
