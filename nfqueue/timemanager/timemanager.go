// Copyright (c) 2021 Tigera, Inc. All rights reserved.
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

package timemanager

import "time"

const (
	defaultExpireTickDuration = 5 * time.Second
	defaultExpireDuration     = 10 * time.Second
)

type addTimeRequest struct {
	id     string
	timeCh chan time.Time
}

type existsRequest struct {
	id     string
	boolCh chan bool
}

type TimeManager interface {
	// AddTime sets the time for the given id if one doesn't already exist. If a time already exists for the given id
	// it is not overwritten and it is returned.
	AddTime(string) time.Time

	Exists(string) bool

	Start()
	Stop()
}

type timeManager struct {
	requestChan        chan interface{}
	expireTickDuration time.Duration
	expireDuration     time.Duration
}

func New(options ...Option) TimeManager {
	manager := &timeManager{
		requestChan:        make(chan interface{}, 100),
		expireTickDuration: defaultExpireTickDuration,
		expireDuration:     defaultExpireDuration,
	}

	for _, option := range options {
		option(manager)
	}

	return manager
}

func (manager *timeManager) Start() {
	go func() {
		idToTimesMap := make(map[string]time.Time)
		ticker := time.NewTicker(manager.expireTickDuration)

		defer ticker.Stop()
	done:
		for {
			select {
			case request, ok := <-manager.requestChan:
				if !ok {
					break done
				}

				switch r := request.(type) {
				case addTimeRequest:
					if _, exists := idToTimesMap[r.id]; !exists {
						idToTimesMap[r.id] = time.Now()
					}
					r.timeCh <- idToTimesMap[r.id]
					close(r.timeCh)
				case existsRequest:
					_, exists := idToTimesMap[r.id]
					r.boolCh <- exists
					close(r.boolCh)
				}
			case <-ticker.C:
				now := time.Now()
				for id, t := range idToTimesMap {
					if now.Sub(t) > manager.expireDuration {
						delete(idToTimesMap, id)
					}
				}
			}
		}
	}()
}

func (manager *timeManager) Stop() {
	close(manager.requestChan)
}

func (manager *timeManager) AddTime(id string) time.Time {
	ch := make(chan time.Time)
	manager.requestChan <- addTimeRequest{
		id:     id,
		timeCh: ch,
	}
	return <-ch
}

func (manager *timeManager) Exists(id string) bool {
	ch := make(chan bool)
	manager.requestChan <- existsRequest{
		id:     id,
		boolCh: ch,
	}

	return <-ch
}
