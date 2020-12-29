// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package collector

import (
	"sync"
	"time"
)

// Aggregation variables
type L7SvcAggregationKind int
type L7URLAggregationKind int
type L7ErrAggregationKind int

const (
	L7SvcNameNone L7SvcAggregationKind = iota
	L7DstSvcName
)

const (
	L7URLNone L7URLAggregationKind = iota
	L7URLQuery
	L7URLQueryPath
	L7URLQueryPathBase
)

const (
	L7ErrorCodeNone L7ErrAggregationKind = iota
	L7ErrorCode
)

// l7LogAggregator builds and implements the L7LogAggregator and the
// L7LogGetter interfaces.
// The l7LogAggregator is responsible for creating, aggregating, and
// storing the aggregated L7 logs until they are exported.
type l7LogAggregator struct {
	svcAggregation       L7SvcAggregationKind
	urlAggregation       L7URLAggregationKind
	errorCodeAggregation L7ErrAggregationKind
	l7Store              map[L7Meta]L7Spec
	l7Mutex              sync.Mutex
	aggregationStartTime time.Time
}

// New L7LogAggregator constructs a L7LogAggregator
func NewL7LogAggregator() L7LogAggregator {
	return &l7LogAggregator{
		svcAggregation:       L7DstSvcName,
		urlAggregation:       L7URLQueryPathBase,
		errorCodeAggregation: L7ErrorCode,
		l7Store:              make(map[L7Meta]L7Spec),
		aggregationStartTime: time.Now(),
	}
}

func (la *l7LogAggregator) AggregateOver(sak L7SvcAggregationKind, uak L7URLAggregationKind, eak L7ErrAggregationKind) L7LogAggregator {
	la.svcAggregation = sak
	la.urlAggregation = uak
	la.errorCodeAggregation = eak
	return la
}

func (la *l7LogAggregator) FeedUpdate(update L7Update) error {
	meta, spec, err := NewL7MetaSpecFromUpdate(update, la.svcAggregation, la.urlAggregation, la.errorCodeAggregation)
	if err != nil {
		return err
	}

	// Ensure that we cannot add or aggregate new logs into the store at
	// the same time that existing logs are being flushed out.
	la.l7Mutex.Lock()
	defer la.l7Mutex.Unlock()

	if _, ok := la.l7Store[meta]; ok {
		existing := la.l7Store[meta]
		existing.Merge(spec)
		la.l7Store[meta] = existing
	} else {
		// TODO MattL: Do we need to add in a log limit of some sort for rate limiting?
		la.l7Store[meta] = spec
	}

	return nil
}

func (la *l7LogAggregator) Get() []*L7Log {
	var l7Logs []*L7Log
	aggregationEndTime := time.Now()

	// Ensure that we can't add or aggregate new logs into the store at the
	// same time as existing logs are being flushed out.
	la.l7Mutex.Lock()
	defer la.l7Mutex.Unlock()

	for meta, spec := range la.l7Store {
		l7Data := L7Data{meta, spec}
		l7Logs = append(l7Logs, l7Data.ToL7Log(
			la.aggregationStartTime,
			aggregationEndTime,
		))
	}

	la.l7Store = make(map[L7Meta]L7Spec)
	la.aggregationStartTime = aggregationEndTime
	return l7Logs
}
