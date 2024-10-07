package v1

import (
	"encoding/json"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"
)

type AggregationRequest struct {
	// The cluster name. Defaults to "cluster".
	Cluster string `json:"cluster" validate:"omitempty"`

	// Time range used to limit the record selection by time.
	TimeRange *lmav1.TimeRange `json:"time_range" validate:"required"`

	// The selector used to select the records to be aggregated. The dataset is further limited by the time range and
	// by the users RBAC.
	Selector string `json:"selector" validate:"omitempty"`

	// Whether a time series is requested. If this is set to true, the time interval will be automatically
	// determined based on the time range.  Note that for small time ranges the interval may contain only one
	// time bucket.
	IncludeTimeSeries bool `json:"include_time_series" validate:"omitempty"`

	// The aggregation to perform. This is a set of named aggregations, each aggregation is a raw elasticsearch aggregation type.
	Aggregations map[string]json.RawMessage `json:"aggregations" validate:"required"`

	// Timeout for the request. Defaults to 60s.
	Timeout time.Duration `json:"timeout" validate:"omitempty"`
}

type AggregationResponse struct {
	// The results organized by time bucket. Depending on the requested time range, or if IncludeTimeSeries is false,
	// there may only be a single bucket which would correspond to the full time range requested.
	Buckets []AggregationTimeBucket `json:"buckets"`
}

type AggregationTimeBucket struct {
	// The start time of this bucket.
	StartTime metav1.Time `json:"start_time"`

	// The aggregation results returned as a raw elasticsearch type.
	Aggregations map[string]json.RawMessage `json:"aggregations"`
}
