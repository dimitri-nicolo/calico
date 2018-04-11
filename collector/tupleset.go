// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package collector

type empty struct{}

var emptyValue = empty{}

type tupleSet map[Tuple]empty

func NewTupleSet() tupleSet {
	return make(tupleSet)
}

func (set tupleSet) Len() int {
	return len(set)
}

func (set tupleSet) Add(t Tuple) {
	set[t] = emptyValue
}

func (set tupleSet) Discard(t Tuple) {
	delete(set, t)
}

func (set tupleSet) Contains(t Tuple) bool {
	_, present := set[t]
	return present
}
