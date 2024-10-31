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

package mocknetlink

import (
	"context"
	"sync"

	gonfqueue "github.com/florianl/go-nfqueue"

	"github.com/projectcalico/calico/felix/netlinkshim"
)

type NfQueueFactory struct {
	// Control input
	OpenErr     error
	RegisterErr error
	CloseErr    error

	// Instantiated mock NfQueue.
	lock        sync.Mutex
	openCalls   int
	MockNfQueue *MockNfQueue
}

func (n *NfQueueFactory) New(config *gonfqueue.Config) (netlinkshim.NfQueue, error) {
	n.lock.Lock()
	defer n.lock.Unlock()
	if err := n.OpenErr; err != nil {
		n.OpenErr = nil
		return nil, err
	}
	m := &MockNfQueue{
		closeErr:      n.CloseErr,
		registerErr:   n.RegisterErr,
		Verdicts:      make(map[uint32]int),
		BatchVerdicts: make(map[uint32]int),
		Marks:         make(map[uint32]int),
	}
	n.MockNfQueue = m
	n.openCalls++
	n.CloseErr = nil
	n.RegisterErr = nil
	return m, nil
}

func (n *NfQueueFactory) NumOpenCalls() int {
	n.lock.Lock()
	defer n.lock.Unlock()
	return n.openCalls
}

// Validate the mock netlink adheres to the netlink interface.
var _ netlinkshim.NfQueue = (*MockNfQueue)(nil)

type MockNfQueue struct {
	Verdicts       map[uint32]int
	BatchVerdicts  map[uint32]int
	Marks          map[uint32]int
	SetVerdictErrs []error
	Closed         bool

	lock        sync.Mutex
	packetID    uint32
	closeErr    error
	registerErr error
	hookFunc    gonfqueue.HookFunc
	errFunc     gonfqueue.ErrorFunc
}

func (m *MockNfQueue) IsRegistered() bool {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.hookFunc != nil
}

func (m *MockNfQueue) RegisterWithErrorFunc(ctx context.Context, fn gonfqueue.HookFunc, errfn gonfqueue.ErrorFunc) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.hookFunc, m.errFunc = fn, errfn
	return m.registerErr
}

func (m *MockNfQueue) SetVerdict(id uint32, verdict int) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	if len(m.SetVerdictErrs) > 0 {
		err := m.SetVerdictErrs[0]
		m.SetVerdictErrs = m.SetVerdictErrs[1:]
		return err
	}
	m.Verdicts[id] = verdict
	return nil
}

func (m *MockNfQueue) SetVerdictWithMark(id uint32, verdict, mark int) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	if len(m.SetVerdictErrs) > 0 {
		err := m.SetVerdictErrs[0]
		m.SetVerdictErrs = m.SetVerdictErrs[1:]
		return err
	}
	m.Verdicts[id] = verdict
	m.Marks[id] = mark
	return nil
}

func (m *MockNfQueue) SetVerdictBatch(id uint32, verdict int) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	if len(m.SetVerdictErrs) > 0 {
		err := m.SetVerdictErrs[0]
		m.SetVerdictErrs = m.SetVerdictErrs[1:]
		return err
	}
	m.BatchVerdicts[id] = verdict
	return nil
}

func (m *MockNfQueue) Close() error {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.Closed = true
	return m.closeErr
}

func (m *MockNfQueue) DebugKillConnection() error {
	return nil
}

func (m *MockNfQueue) SendAttributes(a gonfqueue.Attribute) int {
	m.lock.Lock()
	m.packetID++
	id := m.packetID
	a.PacketID = &id
	m.lock.Unlock()
	return m.hookFunc(a)
}

func (m *MockNfQueue) SendError(e error) int {
	return m.errFunc(e)
}

func (m *MockNfQueue) HasVerdict(id uint32) bool {
	m.lock.Lock()
	defer m.lock.Unlock()
	_, ok := m.Verdicts[id]
	return ok
}

func (m *MockNfQueue) HasBatchVerdict(id uint32) bool {
	m.lock.Lock()
	defer m.lock.Unlock()
	_, ok := m.BatchVerdicts[id]
	return ok
}

func (m *MockNfQueue) GetVerdict(id uint32) int {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.Verdicts[id]
}

func (m *MockNfQueue) GetBatchVerdict(id uint32) int {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.BatchVerdicts[id]
}

func (m *MockNfQueue) GetMark(id uint32) int {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.Marks[id]
}
