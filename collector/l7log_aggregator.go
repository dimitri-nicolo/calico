// Copyright (c) 2020-2021 Tigera, Inc. All rights reserved.

package collector

import (
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/felix/config"
)

// TODO: If named aggregation levels works better, refactor all these levels to only be strings
// Aggregation variables
type L7HTTPHeaderAggregationKind int
type L7HTTPMethodAggregationKind int
type L7ServiceAggregationKind int
type L7DestinationAggregationKind int
type L7SourceAggregationKind int
type L7URLAggregationKind int
type L7ResponseCodeAggregationKind int

const (
	L7HTTPHeaderInfo L7HTTPHeaderAggregationKind = iota
	L7HTTPHeaderInfoNone
)

const (
	L7HTTPMethod L7HTTPMethodAggregationKind = iota
	L7HTTPMethodNone
)

const (
	L7ServiceInfo L7ServiceAggregationKind = iota
	L7ServiceInfoNone
)

const (
	L7DestinationInfo L7DestinationAggregationKind = iota
	L7DestinationInfoNone
)

const (
	L7SourceInfo L7SourceAggregationKind = iota
	L7SourceInfoNone
)

const (
	L7FullURL L7URLAggregationKind = iota
	L7URLWithoutQuery
	L7BaseURL
	L7URLNone
)

const (
	L7ResponseCode L7ResponseCodeAggregationKind = iota
	L7ResponseCodeNone
)

var l7AggregationKindTypeMap map[string]int = map[string]int{
	"ExcludeL7HTTPHeaderInfo":  int(L7HTTPHeaderInfoNone),
	"IncludeL7HTTPHeaderInfo":  int(L7HTTPHeaderInfo),
	"ExcludeL7HTTPMethod":      int(L7HTTPMethodNone),
	"IncludeL7HTTPMethod":      int(L7HTTPMethod),
	"ExcludeL7ServiceInfo":     int(L7ServiceInfoNone),
	"IncludeL7ServiceInfo":     int(L7ServiceInfo),
	"ExcludeL7DestinationInfo": int(L7DestinationInfoNone),
	"IncludeL7DestinationInfo": int(L7DestinationInfo),
	"ExcludeL7SourceInfo":      int(L7SourceInfoNone),
	"IncludeL7SourceInfo":      int(L7SourceInfo),
	"ExcludeL7URL":             int(L7URLNone),
	"TrimURLQuery":             int(L7URLWithoutQuery),
	"TrimURLQueryAndPath":      int(L7BaseURL),
	"IncludeL7FullURL":         int(L7FullURL),
	"ExcludeL7ResponseCode":    int(L7ResponseCodeNone),
	"IncludeL7ResponseCode":    int(L7ResponseCode),
}

// L7AggregationKind is a collection of all the different types of aggregation
// values that make up L7 aggregation.
type L7AggregationKind struct {
	HTTPHeader      L7HTTPHeaderAggregationKind
	HTTPMethod      L7HTTPMethodAggregationKind
	Service         L7ServiceAggregationKind
	Destination     L7DestinationAggregationKind
	Source          L7SourceAggregationKind
	TrimURL         L7URLAggregationKind
	ResponseCode    L7ResponseCodeAggregationKind
	NumURLPathParts int
	URLCharLimit    int
}

// Sets the default L7 Aggregation levels. By default, everything is allowed
// except for the Src/Dst details and the extra HTTP header fields.
func DefaultL7AggregationKind() L7AggregationKind {
	return L7AggregationKind{
		HTTPHeader:      L7HTTPHeaderInfoNone,
		HTTPMethod:      L7HTTPMethod,
		Service:         L7ServiceInfo,
		Destination:     L7DestinationInfo,
		Source:          L7SourceInfo,
		TrimURL:         L7FullURL,
		ResponseCode:    L7ResponseCode,
		NumURLPathParts: 5,
		URLCharLimit:    250,
	}
}

// l7LogAggregator builds and implements the L7LogAggregator and the
// L7LogGetter interfaces.
// The l7LogAggregator is responsible for creating, aggregating, and
// storing the aggregated L7 logs until they are exported.
type l7LogAggregator struct {
	kind                 L7AggregationKind
	l7Store              map[L7Meta]L7Spec
	l7Mutex              sync.Mutex
	aggregationStartTime time.Time
	perNodeLimit         int
	numUnLoggedUpdates   int
}

// New L7LogAggregator constructs a L7LogAggregator
func NewL7LogAggregator() L7LogAggregator {
	return &l7LogAggregator{
		kind:                 DefaultL7AggregationKind(),
		l7Store:              make(map[L7Meta]L7Spec),
		aggregationStartTime: time.Now(),
	}
}

func (la *l7LogAggregator) AggregateOver(ak L7AggregationKind) L7LogAggregator {
	la.kind = ak
	return la
}

func (la *l7LogAggregator) PerNodeLimit(l int) L7LogAggregator {
	la.perNodeLimit = l
	return la
}

func (la *l7LogAggregator) FeedUpdate(update L7Update) error {
	meta, spec, err := NewL7MetaSpecFromUpdate(update, la.kind)
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
	} else if (la.perNodeLimit == 0) || (len(la.l7Store) < la.perNodeLimit) {
		// TODO: Add another store for dealing with overflow logs
		// (logs with only counts that represent rate limiting in
		// the L7 collector.
		// Since we expect there to be too many L7 logs, trim out
		// overflow logs since we do not want to use up our log limit
		// to record them since they have less data. Overflow logs will
		// not have a type.
		if meta.Type != "" {
			la.l7Store[meta] = spec
		}
	} else {
		la.numUnLoggedUpdates++
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
	if la.numUnLoggedUpdates > 0 {
		log.Warningf(
			"%v L7 logs were not logged, because of perNodeLimit being set to %v",
			la.numUnLoggedUpdates,
			la.perNodeLimit,
		)
		// Emit an Elastic log to alert about the un logged updates.  This log has no content
		// except for the time period and the number of updates that could not be fully
		// logged.
		excessLog := &L7Log{
			StartTime: la.aggregationStartTime.Unix(),
			EndTime:   aggregationEndTime.Unix(),
			Count:     la.numUnLoggedUpdates,
			Type:      L7LogTypeUnLogged, // Type is otherwise the protocol tcp, tls, http1.1 etc
		}
		l7Logs = append(l7Logs, excessLog)
	}

	la.l7Store = make(map[L7Meta]L7Spec)
	la.aggregationStartTime = aggregationEndTime
	return l7Logs
}

func translateAggregationKind(aggStr string) (int, error) {
	val, ok := l7AggregationKindTypeMap[aggStr]
	if !ok {
		// Unrecognized aggregation level string provided.
		return val, fmt.Errorf("Invalid aggregation kind provided: %s", aggStr)
	}
	return val, nil
}

func getL7AggregationKindFromConfigParams(cfg *config.Config) L7AggregationKind {
	agg := DefaultL7AggregationKind()
	headerInfoLevel, err := translateAggregationKind(cfg.L7LogsFileAggregationHTTPHeaderInfo)
	if err != nil {
		log.Errorf("Unrecognized L7 aggregation parameter for header info: %s", cfg.L7LogsFileAggregationHTTPHeaderInfo)
	} else {
		agg.HTTPHeader = L7HTTPHeaderAggregationKind(headerInfoLevel)
	}
	methodLevel, err := translateAggregationKind(cfg.L7LogsFileAggregationHTTPMethod)
	if err != nil {
		log.Errorf("Unrecognized L7 aggregation parameter for method: %s", cfg.L7LogsFileAggregationHTTPMethod)
	} else {
		agg.HTTPMethod = L7HTTPMethodAggregationKind(methodLevel)
	}
	serviceLevel, err := translateAggregationKind(cfg.L7LogsFileAggregationServiceInfo)
	if err != nil {
		log.Errorf("Unrecognized L7 aggregation parameter for service info: %s", cfg.L7LogsFileAggregationServiceInfo)
	} else {
		agg.Service = L7ServiceAggregationKind(serviceLevel)
	}
	destLevel, err := translateAggregationKind(cfg.L7LogsFileAggregationDestinationInfo)
	if err != nil {
		log.Errorf("Unrecognized L7 aggregation parameter for destination info: %s", cfg.L7LogsFileAggregationDestinationInfo)
	} else {
		agg.Destination = L7DestinationAggregationKind(destLevel)
	}
	srcLevel, err := translateAggregationKind(cfg.L7LogsFileAggregationSourceInfo)
	if err != nil {
		log.Errorf("Unrecognized L7 aggregation parameter for source info: %s", cfg.L7LogsFileAggregationSourceInfo)
	} else {
		agg.Source = L7SourceAggregationKind(srcLevel)
	}
	rcLevel, err := translateAggregationKind(cfg.L7LogsFileAggregationResponseCode)
	if err != nil {
		log.Errorf("Unrecognized L7 aggregation parameter for response code: %s", cfg.L7LogsFileAggregationResponseCode)
	} else {
		agg.ResponseCode = L7ResponseCodeAggregationKind(rcLevel)
	}
	urlLevel, err := translateAggregationKind(cfg.L7LogsFileAggregationTrimURL)
	if err != nil {
		log.Errorf("Unrecognized L7 aggregation parameter for URL: %s", cfg.L7LogsFileAggregationTrimURL)
	} else {
		agg.TrimURL = L7URLAggregationKind(urlLevel)
	}
	agg.NumURLPathParts = cfg.L7LogsFileAggregationNumURLPath
	agg.URLCharLimit = cfg.L7LogsFileAggregationURLCharLimit

	return agg
}
