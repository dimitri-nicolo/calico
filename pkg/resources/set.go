// Copyright (c) 2016-2017 Tigera, Inc. All rights reserved.
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

package resources

import (
	"errors"

	log "github.com/sirupsen/logrus"
)

type Set interface {
	Len() int
	Add(ResourceID)
	AddAll(items []ResourceID)
	Discard(ResourceID)
	Clear()
	Contains(ResourceID) bool
	Iter(func(item ResourceID) error)
	IterDifferences(other Set, notInOther, onlyInOther func(ResourceID) error)
	Copy() Set
	Equals(Set) bool
	ContainsAll(Set) bool
}

type empty struct{}

var emptyValue = empty{}

var (
	StopIteration = errors.New("Stop iteration")
	RemoveItem    = errors.New("Remove item")
)

func NewSet() Set {
	return make(mapSet)
}

func SetFrom(members ...ResourceID) Set {
	s := NewSet()
	s.AddAll(members)
	return s
}

func SetFromArray(membersArray []ResourceID) Set {
	s := NewSet()
	s.AddAll(membersArray)
	return s
}

func EmptySet() Set {
	return mapSet(nil)
}

type mapSet map[ResourceID]empty

func (set mapSet) Len() int {
	return len(set)
}

func (set mapSet) Add(item ResourceID) {
	set[item] = emptyValue
}

func (set mapSet) AddAll(items []ResourceID) {

	for _, r := range items {
		set.Add(r)
	}
}

func (set mapSet) Discard(item ResourceID) {
	delete(set, item)
}

func (set mapSet) Clear() {
	for item := range set {
		delete(set, item)
	}
}

func (set mapSet) Contains(item ResourceID) bool {
	_, present := set[item]
	return present
}

func (set mapSet) Iter(visitor func(item ResourceID) error) {
loop:
	for item := range set {
		err := visitor(item)
		switch err {
		case StopIteration:
			break loop
		case RemoveItem:
			delete(set, item)
		case nil:
		default:
			log.WithError(err).Panic("Unexpected iteration error")
		}
	}
}

func (set mapSet) IterDifferences(other Set, notInOther, onlyInOther func(ResourceID) error) {
	set.Iter(func(id ResourceID) error {
		if !other.Contains(id) {
			return notInOther(id)
		}
		return nil
	})
	other.Iter(func(id ResourceID) error {
		if !set.Contains(id) {
			return onlyInOther(id)
		}
		return nil
	})
}

func (set mapSet) Copy() Set {
	cpy := NewSet()
	for item := range set {
		cpy.Add(item)
	}
	return cpy
}

func (set mapSet) Equals(other Set) bool {
	if set.Len() != other.Len() {
		return false
	}
	for item := range set {
		if !other.Contains(item) {
			return false
		}
	}
	return true
}

func (set mapSet) ContainsAll(other Set) bool {
	result := true
	other.Iter(func(item ResourceID) error {
		if !set.Contains(item) {
			result = false
			return StopIteration
		}
		return nil
	})
	return result
}
