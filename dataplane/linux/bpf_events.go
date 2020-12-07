// Copyright (c) 2020 Tigera, Inc. All rights reserved.
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

package intdataplane

import (
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

func (p *bpfEventPoller) Start() {
	if len(p.sinks) > 0 {
		go p.run()
	} else {
		log.Warn("No event sinks registered, exiting")
		p.events.Close()
	}
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
