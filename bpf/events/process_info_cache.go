// Copyright (c) 2020-2021 Tigera, Inc. All rights reserved.

package events

import (
	"bytes"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/felix/collector"
	"github.com/projectcalico/felix/jitter"
)

type ProcessEntry struct {
	collector.ProcessInfo
	expiresAt time.Time
}

// BPFProcessInfoCache reads process information from Linux via kprobes.
type BPFProcessInfoCache struct {
	// Read-Write mutex for process info
	lock sync.RWMutex
	// Map of tuple to process information
	cache map[collector.Tuple]ProcessEntry

	// Ticker for running the GC thread that reaps expired entries.
	expireTicker jitter.JitterTicker
	// Max time for which an entry is retained.
	entryTTL time.Duration

	stopOnce     sync.Once
	wg           sync.WaitGroup
	stopC        chan struct{}
	eventC       <-chan EventProtoStats
	processInfoC chan collector.ProcessInfo
}

// NewBPFProcessInfoCache returns a new BPFProcessInfoCache
func NewBPFProcessInfoCache(eventChan <-chan EventProtoStats, gcInterval time.Duration, entryTTL time.Duration) *BPFProcessInfoCache {
	return &BPFProcessInfoCache{
		stopC:        make(chan struct{}),
		eventC:       eventChan,
		expireTicker: jitter.NewTicker(gcInterval, gcInterval/10),
		entryTTL:     entryTTL,
		cache:        make(map[collector.Tuple]ProcessEntry),
		lock:         sync.RWMutex{},
	}
}

func (r *BPFProcessInfoCache) Start() error {
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		r.run()
	}()

	return nil
}

func (r *BPFProcessInfoCache) run() {
	defer r.expireTicker.Stop()
	for {
		select {
		case <-r.stopC:
			return
		case event := <-r.eventC:
			info := convertProtoEventToProcessInfo(event)
			log.Debugf("Converted event %+v to process info %+v", event, info)
			r.updateCache(info)
		case <-r.expireTicker.Channel():
			r.expireCacheEntries()
		}
	}
}

func (r *BPFProcessInfoCache) Stop() {
	r.stopOnce.Do(func() {
		close(r.stopC)
	})
	r.wg.Wait()
}

func (r *BPFProcessInfoCache) Lookup(tuple collector.Tuple, direction collector.TrafficDirection) (collector.ProcessInfo, bool) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	t := tuple
	if direction == collector.TrafficDirInbound {
		// Inbound data is stored in the reverse order.
		t = t.GetReverseTuple()
	}
	log.Debugf("Looking up process info for tuple %+v in direction %v", tuple, direction)
	if entry, ok := r.cache[t]; ok {
		log.Debugf("Found process info %+v for tuple %+v in direction %v", entry.ProcessInfo, tuple, direction)
		return entry.ProcessInfo, true
	}
	log.Debugf("Process info not found for tuple %+v in direction %v", tuple, direction)
	return collector.ProcessInfo{}, false
}

func (r *BPFProcessInfoCache) updateCache(info collector.ProcessInfo) {
	r.lock.Lock()
	defer r.lock.Unlock()
	log.Debugf("Updating process info %+v", info)
	entry := ProcessEntry{
		ProcessInfo: info,
		expiresAt:   time.Now().Add(r.entryTTL),
	}
	r.cache[info.Tuple] = entry
	return
}

func (r *BPFProcessInfoCache) expireCacheEntries() {
	r.lock.Lock()
	defer r.lock.Unlock()

	for tuple, entry := range r.cache {
		if time.Until(entry.expiresAt) <= 0 {
			log.Debugf("Expiring process info %+v. Time until expiration %v", entry, time.Until(entry.expiresAt))
			delete(r.cache, tuple)
			continue
		}
	}
}

func convertProtoEventToProcessInfo(event EventProtoStats) collector.ProcessInfo {
	srcIP := event.Saddr
	dstIP := event.Daddr
	sport := int(event.Sport)
	dport := int(event.Dport)
	tuple := collector.MakeTuple(srcIP, dstIP, int(event.Proto), sport, dport)
	pname := bytes.Trim(event.ProcessName[:], "\x00")
	return collector.ProcessInfo{
		Tuple: tuple,
		ProcessData: collector.ProcessData{
			Name: string(pname),
			Pid:  int(event.Pid),
		},
	}
}
