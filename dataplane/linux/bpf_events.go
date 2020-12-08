// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package intdataplane

import (
	"errors"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/felix/bpf/events"
)

type bpfEventSink func(e events.Event)

type bpfEventPoller struct {
	events events.Events
	sinks  map[events.Type][]bpfEventSink
}

func newBpfEventPoller(e events.Events) *bpfEventPoller {
	return &bpfEventPoller{
		events: e,
		sinks:  make(map[events.Type][]bpfEventSink),
	}
}

func (p *bpfEventPoller) Register(t events.Type, sink bpfEventSink) {
	p.sinks[t] = append(p.sinks[t], sink)
}

func (p *bpfEventPoller) Start() error {
	if len(p.sinks) == 0 {
		return errors.New("no event sinks registered")
	}

	go p.run()
	return nil
}

func (p *bpfEventPoller) Stop() {
	p.events.Close()
}

func (p *bpfEventPoller) run() {
	for {
		event, err := p.events.Next()
		if err != nil {
			log.WithError(err).Warn("Failed to get next event")
			continue
		}

		sinks := p.sinks[event.Type()]
		if len(sinks) == 0 {
			log.Warnf("Event type %d without a sink", event.Type())
			continue
		}

		for _, sink := range sinks {
			sink(event)
		}
	}
}
