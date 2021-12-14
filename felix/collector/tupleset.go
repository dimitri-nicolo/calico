// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package collector

type empty struct{}

var emptyValue = empty{}

type tupleSet map[Tuple]int

func NewTupleSet() tupleSet {
	return make(tupleSet)
}

func (set tupleSet) Len() int {
	return len(set)
}

func (set tupleSet) Add(t Tuple) {
	set[t] = 0
}

// AddWithValue assigns a value to the tuple key. This is useful for saving space when you need to store additional
// information on a tuple but you don't want to create another Tuple to value map in addition to this set. If a non
// empty value has been set for the Tuple key subsequent calls to change the value are ignored. This prevents updates
// that don't have the natOutgoingPort from removing the value.
//
// Note that the only information we currently want to store with a tuple is the post SNAT port. If we start storing
// more information then the value parameter should be changed to a more generic struct.
func (set tupleSet) AddWithValue(t Tuple, natOutgoingPort int) {
	if set[t] == 0 {
		set[t] = natOutgoingPort
	}
}

func (set tupleSet) Discard(t Tuple) {
	delete(set, t)
}

func (set tupleSet) Contains(t Tuple) bool {
	_, present := set[t]
	return present
}

func (set tupleSet) Copy() tupleSet {
	ts := NewTupleSet()
	for tuple, value := range set {
		ts.AddWithValue(tuple, value)
	}
	return ts
}
