// Copyright (c) 2019 Tigera Inc. All rights reserved.

package elastic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/olivere/elastic/v7"

	"github.com/tigera/intrusion-detection/controller/pkg/db"
)

// https://github.com/elastic/elasticsearch/blob/master/client/rest-high-level/src/main/java/org/elasticsearch/client/watcher/ExecutionState.java
const (
	// WatchExecutionStateExecutionNotNeeded the condition of the watch was not met
	WatchExecutionStateExecutionNotNeeded = "execution_not_needed"

	// WatchExecutionStateThrottled Execution has been throttled due to time-based throttling - this might only affect
	// a single action though
	WatchExecutionStateThrottled = "throttled"

	// WatchExecutionStateAcknowledged Execution has been throttled due to ack-based throttling/muting of an action
	// - this might only affect a single action though
	WatchExecutionStateAcknowledged = "acknowledged"

	// WatchExecutionStateExecuted regular execution
	WatchExecutionStateExecuted = "executed"

	// WatchExecutionStateFailed an error in the condition or the execution of the input
	WatchExecutionStateFailed = "failed"

	// WatchExecutionStateThreadpoolRejection a rejection due to a filled up threadpool
	WatchExecutionStateThreadpoolRejection = "threadpool_rejection"

	// WatchExecutionStateNotExecutedWatchMissing the execution was scheduled, but in between the watch was deleted
	WatchExecutionStateNotExecutedWatchMissing = "not_executed_watch_missing"

	// WatchExecutionStateNotExecutedAlreadyQueued even though the execution was scheduled, it was not executed,
	// because the watch was already queued in the thread pool
	WatchExecutionStateNotExecutedAlreadyQueued = "not_executed_already_queued"

	// WatchExecutionStateExecutedMultipleTimes this can happen when a watch was executed, but not completely finished
	// (the triggered watch entry was not deleted), and then watcher is restarted (manually or due to host switch) -
	// the triggered watch will be executed but the history entry already exists
	WatchExecutionStateExecutedMultipleTimes = "executed_multiple_times"
)

type XPackWatcher interface {
	ListWatches(ctx context.Context) ([]db.Meta, error)
	ExecuteWatch(ctx context.Context, body *ExecuteWatchBody) (*elastic.XPackWatchRecord, error)
	PutWatch(ctx context.Context, name string, body *PutWatchBody) error
	GetWatchStatus(ctx context.Context, name string) (*elastic.XPackWatchStatus, error)
	DeleteWatch(ctx context.Context, m db.Meta) error
}

type PutWatchBody struct {
	Trigger        Trigger           `json:"trigger"`
	Input          *Input            `json:"input,omitempty"`
	Condition      *Condition        `json:"condition,omitempty"`
	Transform      *Transform        `json:"transform,omitempty"`
	Actions        map[string]Action `json:"actions,omitempty"`
	Metadata       interface{}       `json:"metadata,omitempty"`
	ThrottlePeriod time.Duration     `json:"throttle_period,omitempty"`
}

type Trigger struct {
	Schedule Schedule `json:"schedule"`
}

type Schedule struct {
	// Omitted hourly, daily, weekly, monthly, yearly, cron
	Interval *Interval `json:"interval,omitempty"`
}

type Interval struct {
	time.Duration
}

func (i Interval) MarshalJSON() ([]byte, error) {
	val := fmt.Sprintf("%ds", int64(i.Seconds()))
	return json.Marshal(&val)
}

func (i *Interval) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	m := regexp.MustCompile(`^(\d+)([smhdw]?)$`).FindStringSubmatch(raw)
	if m == nil {
		return fmt.Errorf("error parsing interval %q", raw)
	}
	count, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil {
		return fmt.Errorf("error parsing interval %q: %s", raw, err)
	}
	var multiplier time.Duration
	switch m[2] {
	case "", "s":
		multiplier = time.Second
	case "m":
		multiplier = time.Minute
	case "h":
		multiplier = time.Hour
	case "d":
		multiplier = time.Hour * 24
	case "w":
		multiplier = time.Hour * 24 * 7
	}
	i.Duration = time.Duration(count) * multiplier
	return nil
}

type Input struct {
	Simple *Simple `json:"simple,omitempty"`
	Search *Search `json:"search,omitempty"`
}

type Simple map[string]interface{}

type Search struct {
	Request SearchRequest `json:"request"`
	Extract []string      `json:"extract,omitempty"`
	Timeout *Time         `json:"timeout,omitempty"`
}

type SearchRequest struct {
	SearchType     string          `json:"search_type,omitempty"`
	Indices        []string        `json:"indices,omitempty"`
	Body           interface{}     `json:"body,omitempty"`
	IndicesOptions *IndicesOptions `json:"indices_options,omitempty"`
}

type IndicesOptions struct {
	ExpandWildcards   string `json:"expand_wildcards,omitempty"`
	IgnoreUnavailable *bool  `json:"ignore_unavailable,omitempty"`
	AllowNoIndices    *bool  `json:"allow_no_indices,omitempty"`
}

type Condition struct {
	Always       BoolMap       `json:"always,omitempty"`
	Never        BoolMap       `json:"never,omitempty"`
	Compare      *Comparison   `json:"compare,omitempty"`
	ArrayCompare *ArrayCompare `json:"array_compare,omitempty"`
	Script       *Script       `json:"script,omitempty"`
}

type BoolMap bool

func (b BoolMap) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{})
}

func (b *BoolMap) UnmarshalJSON([]byte) error {
	*b = true
	return nil
}

type Comparison struct {
	Key       string
	Operation string
	Value     interface{}
}

func (c Comparison) MarshalJSON() ([]byte, error) {
	x := map[string]map[string]interface{}{
		c.Key: {
			c.Operation: c.Value,
		},
	}
	return json.Marshal(x)
}

func (c *Comparison) UnmarshalJSON(data []byte) error {
	var x map[string]map[string]interface{}
	err := json.Unmarshal(data, &x)
	if err != nil {
		return err
	}
	if len(x) != 1 {
		return errors.New("invalid comparison")
	}
	for k, ov := range x {
		if len(ov) != 1 {
			return errors.New("invalid comparison")
		}
		c.Key = k
		for o, v := range ov {
			c.Operation = o
			c.Value = v
		}
	}
	return nil
}

type ArrayCompare struct {
	ArrayPath  string
	Path       string
	Quantifier string
	Value      interface{}
}

func (a ArrayCompare) MarshalJSON() ([]byte, error) {
	x := map[string]interface{}{
		a.ArrayPath: map[string]interface{}{
			"path": a.Path,
			a.Quantifier: map[string]interface{}{
				"value": a.Value,
			},
		},
	}
	return json.Marshal(x)
}

func (a *ArrayCompare) UnmarshalJSON(data []byte) error {
	var x map[string]map[string]interface{}
	if err := json.Unmarshal(data, &x); err != nil {
		return err
	}
	if len(x) != 1 {
		return errors.New("invalid array compare")
	}

	for k, v := range x {
		if len(v) != 2 {
			return errors.New("invalid array compare")
		}
		arrayPath := k
		i, ok := v["path"]
		if !ok {
			return errors.New("missing path")
		}
		path, ok := i.(string)
		if !ok {
			return errors.New("invalid path")
		}
		for quantifier, i := range v {
			if quantifier == "path" {
				continue
			}

			comparison, ok := i.(map[string]interface{})
			if !ok {
				return errors.New("invalid comparison")
			}

			value, ok := comparison["value"]
			if !ok {
				return errors.New("missing value")
			}

			*a = ArrayCompare{arrayPath, path, quantifier, value}
			return nil
		}
	}

	panic("unreachable")
}

type Script struct {
	Id       string                 `json:"id,omitempty"`
	Language string                 `json:"lang,omitempty"`
	Source   string                 `json:"source,omitempty"`
	Params   map[string]interface{} `json:"params,omitempty"`
}

type Action struct {
	Condition      *Condition    `json:"condition,omitempty"`
	Transform      *Transform    `json:"transform,omitempty"`
	Index          *IndexAction  `json:"index,omitempty"`
	ThrottlePeriod time.Duration `json:"throttle_period,omitempty"`
}

type IndexAction struct {
	Index              string    `json:"index"`
	DocId              string    `json:"doc_id,omitempty"`
	ExecutionTimeField string    `json:"execution_time_field,omitempty"`
	Timeout            *Interval `json:"timeout,omitempty"`
	Refresh            string    `json:"refresh,omitempty"`
}

type Transform struct {
	Script
}

func (t Transform) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]Script{"script": t.Script})
}

func (t *Transform) UnmarshalJSON(data []byte) error {
	var v map[string]Script
	err := json.Unmarshal(data, &v)
	if err != nil {
		return err
	}
	if script, ok := v["script"]; ok {
		t.Script = script
	}
	return errors.New("missing script key")
}

type ExecuteWatchBody struct {
	TriggerData      interface{}   `json:"trigger_data,omitempty"`
	IgnoreCondition  bool          `json:"ignore_condition,omitempty"`
	AlternativeInput interface{}   `json:"alternative_input,omitempty"`
	ActionModes      []string      `json:"action_modes,omitempty"`
	RecordExecution  bool          `json:"record_execution,omitempty"`
	Watch            *PutWatchBody `json:"watch,omitempty"`
}
