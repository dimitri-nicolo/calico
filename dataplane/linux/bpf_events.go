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

func startEventPoller(e events.Events) error {
	go func() {
		for {
			event, err := e.Next()
			if err != nil {
				log.WithError(err).Warn("Failed to get next event")
				continue
			}
			if event == nil {
				continue
			}
			switch event.Type() {
			case events.TypeTcpv4Events:
				log.WithField("event", event).Debug("Received TCP v4 event")
			case events.TypeUdpv4Events:
				log.WithField("event", event).Debug("Received UDP v4 event")
			default:
				log.Warn("Unknown event type")
				continue
			}
		}
	}()
	return nil
}
