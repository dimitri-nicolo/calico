// Copyright (c) 2019 Tigera, Inc. All rights reserved.
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
	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
)

type Set interface {
	Len() int
	Add(apiv3.ResourceID)
	AddSet(Set)
	AddAll(items []apiv3.ResourceID)
	Discard(apiv3.ResourceID)
	Clear()
	Contains(apiv3.ResourceID) bool
	Iter(func(item apiv3.ResourceID) error)
	IterDifferences(other Set, notInOther, onlyInOther func(apiv3.ResourceID) error)
	Copy() Set
	Equals(Set) bool
	ContainsAll(Set) bool
	ToSlice() []apiv3.ResourceID
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

func SetFrom(members ...apiv3.ResourceID) Set {
	s := NewSet()
	s.AddAll(members)
	return s
}

func SetFromArray(membersArray []apiv3.ResourceID) Set {
	s := NewSet()
	s.AddAll(membersArray)
	return s
}

func EmptySet() Set {
	return mapSet(nil)
}

type mapSet map[apiv3.ResourceID]empty

func (set mapSet) Len() int {
	return len(set)
}

func (set mapSet) Add(item apiv3.ResourceID) {
	set[item] = emptyValue
}

func (set mapSet) AddSet(other Set) {
	other.Iter(func(item apiv3.ResourceID) error {
		set.Add(item)
		return nil
	})
}

func (set mapSet) AddAll(items []apiv3.ResourceID) {

	for _, r := range items {
		set.Add(r)
	}
}

func (set mapSet) Discard(item apiv3.ResourceID) {
	delete(set, item)
}

func (set mapSet) Clear() {
	for item := range set {
		delete(set, item)
	}
}

func (set mapSet) Contains(item apiv3.ResourceID) bool {
	_, present := set[item]
	return present
}

func (set mapSet) Iter(visitor func(item apiv3.ResourceID) error) {
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

func (set mapSet) IterDifferences(other Set, notInOther, onlyInOther func(apiv3.ResourceID) error) {
	set.Iter(func(id apiv3.ResourceID) error {
		if !other.Contains(id) {
			return notInOther(id)
		}
		return nil
	})
	other.Iter(func(id apiv3.ResourceID) error {
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
	other.Iter(func(item apiv3.ResourceID) error {
		if !set.Contains(item) {
			result = false
			return StopIteration
		}
		return nil
	})
	return result
}

func (set mapSet) ToSlice() []apiv3.ResourceID {
	ids := make([]apiv3.ResourceID, 0, len(set))
	for id := range set {
		ids = append(ids, id)
	}
	return ids
}
