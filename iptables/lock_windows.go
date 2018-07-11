// Copyright (c) 2018 Tigera, Inc. All rights reserved.
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

package iptables

import (
	"time"

	"io"
	"sync"
)

func NewSharedLock(lockFilePath string, lockTimeout, lockProbeInterval time.Duration) *SharedLock {
	return &SharedLock{
		lockFilePath:      lockFilePath,
		lockTimeout:       lockTimeout,
		lockProbeInterval: lockProbeInterval,
		GrabIptablesLocks: GrabIptablesLocks,
	}
}

// SharedLock allows for multiple goroutines to share the iptables lock without blocking on each
// other.  That is safe because each of our goroutines is accessing a different iptables table, so
// they do not conflict.
type SharedLock struct {
	lock           sync.Mutex
	referenceCount int

	iptablesLockHandle io.Closer

	lockFilePath      string
	lockTimeout       time.Duration
	lockProbeInterval time.Duration

	GrabIptablesLocks func(lockFilePath, socketName string, timeout, probeInterval time.Duration) (io.Closer, error)
}

func (l *SharedLock) Lock() {
	l.lock.Lock()
	defer l.lock.Unlock()

}

func (l *SharedLock) Unlock() {
	l.lock.Lock()
	defer l.lock.Unlock()

}

type Locker struct {
	Lock16 io.Closer
	Lock14 io.Closer
}

func (l *Locker) Close() error {
	return nil
}

func GrabIptablesLocks(lockFilePath, socketName string, timeout, probeInterval time.Duration) (io.Closer, error) {
	l := &Locker{}
	return l, nil
}
