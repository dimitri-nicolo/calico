package legacy

import (
	"log"

	"github.com/sirupsen/logrus"

	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

// contextLogger returns a suitable context logger for use in a request to the backend.
func contextLogger(i bapi.ClusterInfo) *logrus.Entry {
	f := logrus.Fields{
		"cluster": i.Cluster,
		"tenant":  i.Tenant,
	}
	return logrus.WithFields(f)
}

// singleDashToBlank returns an empty string instead of a "-".
// We store empty fields in ES as "-" to indicate that the field was present
// but empty (as opposed to the client simply not writing it at all).
func singleDashToBlank(s string) string {
	if s == "-" {
		return ""
	}
	return s
}

// fieldTracker is a helper for returning data from a CompositeAggregationKey.
//
// The order of results in the key matter, and match the order of the composite source
// information on the request. fieldTracker correlates the given composite sources with
// the returned aggregation key, providing easy methods to extract values of different
// data types.
type fieldTracker struct {
	fieldToIndex map[string]int
}

func (f *fieldTracker) Index(field string) int {
	i, ok := f.fieldToIndex[field]
	if !ok {
		log.Fatalf("Attempt to access unknown field: %s", field)
	}
	return i
}

func (f *fieldTracker) ValueString(key lmaelastic.CompositeAggregationKey, field string) string {
	return singleDashToBlank(key[f.Index(field)].String())
}

func (f *fieldTracker) ValueInt64(key lmaelastic.CompositeAggregationKey, field string) int64 {
	return int64(key[f.Index(field)].Float64())
}

func (f *fieldTracker) ValueInt32(key lmaelastic.CompositeAggregationKey, field string) int32 {
	return int32(key[f.Index(field)].Float64())
}

func newFieldTracker(sources []lmaelastic.AggCompositeSourceInfo) *fieldTracker {
	t := fieldTracker{fieldToIndex: map[string]int{}}
	for idx, source := range sources {
		t.fieldToIndex[source.Field] = idx
	}
	return &t
}
