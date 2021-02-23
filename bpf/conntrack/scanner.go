// Copyright (c) 2020-2021 Tigera, Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package conntrack

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/felix/bpf"
	"github.com/projectcalico/felix/jitter"
)

var (
	conntrackCounterSweeps = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "felix_bpf_conntrack_sweeps",
		Help: "Number of contrack table sweeps made so far",
	})
	conntrackGaugeUsed = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "felix_bpf_conntrack_used",
		Help: "Number of used entries visited during a conntrack table sweep",
	})
	conntrackGaugeCleaned = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "felix_bpf_conntrack_cleaned",
		Help: "Number of entries cleaned during a conntrack table sweep",
	})
)

func init() {
	prometheus.MustRegister(conntrackCounterSweeps)
	prometheus.MustRegister(conntrackGaugeUsed)
	prometheus.MustRegister(conntrackGaugeCleaned)
}

// ScanVerdict represents the set of values returned by EntryScan
type ScanVerdict int

const (
	// ScanVerdictOK means entry is fine and should remain
	ScanVerdictOK ScanVerdict = iota
	// ScanVerdictDelete means entry should be deleted
	ScanVerdictDelete

	// ScanPeriod determines how often we iterate over the conntrack table.
	ScanPeriod = 10 * time.Second
)

// EntryGet is a function prototype provided to EntryScanner in case it needs to
// evaluate other entries to make a verdict
type EntryGet func(Key) (Value, error)

// EntryScanner is a function prototype to be called on every entry by the scanner
type EntryScanner interface {
	Check(Key, Value, EntryGet) ScanVerdict
}

// EntryScannerSynced is a scaner synchronized with the iteration start/end.
type EntryScannerSynced interface {
	EntryScanner
	IterationStart()
	IterationEnd()
}

// Scanner iterates over a provided conntrack map and call a set of EntryScanner
// functions on each entry in the order as they were passed to NewScanner. If
// any of the EntryScanner returns ScanVerdictDelete, it deletes the entry, does
// not call any other EntryScanner and continues the iteration.
//
// It provides a delete-save iteration over the conntrack table for multiple
// evaluation functions, to keep their implementation simpler.
type Scanner struct {
	ctMap    bpf.Map
	scanners []EntryScanner

	wg       sync.WaitGroup
	stopCh   chan struct{}
	stopOnce sync.Once
}

// NewScanner returns a scanner for the given conntrack map and the set of
// EntryScanner. They are executed in the provided order on each entry.
func NewScanner(ctMap bpf.Map, scanners ...EntryScanner) *Scanner {
	return &Scanner{
		ctMap:    ctMap,
		scanners: scanners,
		stopCh:   make(chan struct{}),
	}
}

// Scan executes a scanning iteration
func (s *Scanner) Scan() {
	s.iterStart()
	defer s.iterEnd()

	debug := log.GetLevel() >= log.DebugLevel

	var ctKey Key
	var ctVal Value

	used := 0
	cleaned := 0

	err := s.ctMap.Iter(func(k, v []byte) bpf.IteratorAction {
		copy(ctKey[:], k[:])
		copy(ctVal[:], v[:])

		used++

		if debug {
			log.WithFields(log.Fields{
				"key":   ctKey,
				"entry": ctVal,
			}).Debug("Examining conntrack entry")
		}

		for _, scanner := range s.scanners {
			if verdict := scanner.Check(ctKey, ctVal, s.get); verdict == ScanVerdictDelete {
				if debug {
					log.Debug("Deleting conntrack entry.")
				}
				cleaned++
				return bpf.IterDelete
			}
		}
		return bpf.IterNone
	})

	conntrackCounterSweeps.Inc()
	conntrackGaugeUsed.Set(float64(used))
	conntrackGaugeCleaned.Set(float64(cleaned))

	if err != nil {
		log.WithError(err).Warn("Failed to iterate over conntrack map")
	}
}

func (s *Scanner) get(k Key) (Value, error) {
	v, err := s.ctMap.Get(k.AsBytes())

	if err != nil {
		return Value{}, err
	}

	return ValueFromBytes(v), nil
}

// Start the periodic scanner
func (s *Scanner) Start() {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		log.Debug("Conntrack scanner thread started")
		defer log.Debug("Conntrack scanner thread stopped")

		ticker := jitter.NewTicker(ScanPeriod, 100*time.Millisecond)

		for {
			s.Scan()

			select {
			case <-ticker.C:
				log.Debug("Conntrack cleanup timer popped")
			case <-s.stopCh:
				log.Debug("Conntrack cleanup got stop signal")
				return
			}
		}
	}()
}

func (s *Scanner) iterStart() {
	for _, scanner := range s.scanners {
		if synced, ok := scanner.(EntryScannerSynced); ok {
			synced.IterationStart()
		}
	}
}

func (s *Scanner) iterEnd() {
	for _, scanner := range s.scanners {
		if synced, ok := scanner.(EntryScannerSynced); ok {
			synced.IterationEnd()
		}
	}
}

// Stop stops the Scanner and waits for it finishing.
func (s *Scanner) Stop() {
	s.stopOnce.Do(func() {
		close(s.stopCh)
		s.wg.Wait()
	})
}

// AddUnlocked adds an additional EntryScanner to a non-running Scanner
func (s *Scanner) AddUnlocked(scanner EntryScanner) {
	s.scanners = append(s.scanners, scanner)
}

// AddFirstUnlocked adds an additional EntryScanner to a non-running Scanner as
// the first scanner to be called.
func (s *Scanner) AddFirstUnlocked(scanner EntryScanner) {
	s.scanners = append([]EntryScanner{scanner}, s.scanners...)
}
