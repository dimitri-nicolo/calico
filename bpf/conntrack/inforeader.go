// Copyright (c) 2020-2021 Tigera, Inc. All rights reserved.

package conntrack

import (
	"net"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/projectcalico/felix/collector"
	"github.com/projectcalico/felix/timeshim"
)

var (
	conntrackInfoReaderBlocks = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "felix_bpf_conntrack_inforeader_blocks",
		Help: "Conntrack InfoReader blocks",
	})
)

func init() {
	prometheus.MustRegister(conntrackInfoReaderBlocks)
}

const (
	conntrackInfoReaderBatchSize = 1024
)

// InfoReader is an EntryScannerSynced that provides information to Collector as
// collector.ConntrackInfo.
type InfoReader struct {
	timeouts Timeouts
	dsr      bool
	time     timeshim.Interface

	// goTimeOfLastKTimeLookup is the go timestamp of the last time we looked up the kernel time.
	// We cache the kernel time because it's expensive to look up (vs looking up a go timestamp which uses vdso).
	goTimeOfLastKTimeLookup time.Time
	// cachedKTime is the most recent kernel time.
	cachedKTime int64

	outC chan []collector.ConntrackInfo

	bufferedConntrackInfo []collector.ConntrackInfo
}

// NewInfoReader returns a new instance of InfoReader that can be used as a
// EntryScannerSynced with Scanner and as ConntrackInfoReader with
// collector.Collector.
func NewInfoReader(timeouts Timeouts, dsr bool, time timeshim.Interface) *InfoReader {
	r := &InfoReader{
		timeouts: timeouts,
		dsr:      dsr,
		time:     time,

		outC: make(chan []collector.ConntrackInfo, 1000),
	}

	if r.time == nil {
		r.time = timeshim.RealTime()
	}

	return r
}

// Check checks a conntrack entry and translates to collector.ConntrackInfo.
func (r *InfoReader) Check(key Key, val Value, get EntryGet) ScanVerdict {

	switch val.Type() {
	case TypeNATReverse:
		r.pushOut(r.makeConntrackInfo(key, val, true))

	case TypeNATForward:
		// Do nothing, all the relevant info is in the reverce entry that we
		// must hit as well.

	case TypeNormal:
		r.pushOut(r.makeConntrackInfo(key, val, false))
	}

	// We never delete
	return ScanVerdictOK
}

func makeTuple(ipSrc, ipDst net.IP, portSrc, portDst uint16, proto uint8) collector.Tuple {
	var src, dst [16]byte

	copy(src[:], ipSrc.To16())
	copy(dst[:], ipDst.To16())

	return collector.MakeTuple(src, dst, int(proto), int(portSrc), int(portDst))
}

func (r *InfoReader) makeConntrackInfo(key Key, val Value, dnat bool) collector.ConntrackInfo {
	_, expired := r.timeouts.EntryExpired(r.cachedKTime, key.Proto(), val)

	proto := key.Proto()
	ipSrc := key.AddrA()
	ipDst := key.AddrB()

	portSrc := key.PortA()
	portDst := key.PortB()

	data := val.Data()

	coutersSrc := collector.ConntrackCounters{
		Packets: int(data.A2B.Packets),
		Bytes:   int(data.A2B.Bytes),
	}

	coutersDst := collector.ConntrackCounters{
		Packets: int(data.B2A.Packets),
		Bytes:   int(data.B2A.Bytes),
	}

	if data.B2A.Opener {
		// We assume that one of the legs has the opener. If none or both, we
		// cannot tell the direction anyway.
		ipSrc, ipDst = ipDst, ipSrc
		portSrc, portDst = portDst, portSrc
		coutersSrc, coutersDst = coutersDst, coutersSrc
	}

	info := collector.ConntrackInfo{
		Expired:       expired,
		IsDNAT:        dnat,
		Tuple:         makeTuple(ipSrc, ipDst, portSrc, portDst, proto),
		Counters:      coutersSrc,
		ReplyCounters: coutersDst,
	}

	if dnat {
		info.PreDNATTuple = makeTuple(ipSrc, data.OrigDst, portSrc, data.OrigPort, proto)
	}

	return info
}

func (r *InfoReader) pushOut(i collector.ConntrackInfo) {
	r.bufferedConntrackInfo = append(r.bufferedConntrackInfo, i)
	if len(r.bufferedConntrackInfo) >= conntrackInfoReaderBatchSize {
		select {
		case r.outC <- r.bufferedConntrackInfo:
			r.bufferedConntrackInfo = make([]collector.ConntrackInfo, 0, conntrackInfoReaderBatchSize)
		default:
			conntrackInfoReaderBlocks.Inc()
			// keep buffering
		}
	}
}

// IterationStart is called and Scanner starts iterating over the conntrack table.
func (r *InfoReader) IterationStart() {
	if r.cachedKTime == 0 || r.time.Since(r.goTimeOfLastKTimeLookup) > time.Second {
		r.cachedKTime = r.time.KTimeNanos()
		r.goTimeOfLastKTimeLookup = r.time.Now()
	}

	r.bufferedConntrackInfo = make([]collector.ConntrackInfo, 0, conntrackInfoReaderBatchSize)
}

// IterationEnd is called and Scanner ends iterating over the conntrack table.
func (r *InfoReader) IterationEnd() {
	if len(r.bufferedConntrackInfo) > 0 {
		r.outC <- r.bufferedConntrackInfo
	}
}

// Start is called by collector to start consuming data.
func (r *InfoReader) Start() error { return nil }

// ConntrackInfoChan returns a channel for collector to consume data.
func (r *InfoReader) ConntrackInfoChan() <-chan []collector.ConntrackInfo {
	return r.outC
}
