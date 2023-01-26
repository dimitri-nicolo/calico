package legacy

import (
	"log"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"

	elastic "github.com/olivere/elastic/v7"

	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

// contextLogger returns a suitable context logger for use in a request to the backend.
func contextLogger(i bapi.ClusterInfo) *logrus.Entry {
	f := logrus.Fields{
		"cluster": i.Cluster,
	}
	if i.Tenant != "" {
		f["tenant"] = i.Tenant
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
		log.Fatalf("Attempt to access unknown field in ES aggregation result: %s", field)
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
	switch v := key[f.Index(field)].Value.(type) {
	case string:
		i, err := strconv.Atoi(v)
		if err != nil {
			logrus.WithField("field", field).WithError(err).Error("Error parsing field as int")
			return 0
		}
		return int32(i)
	case float64:
		return int32(key[f.Index(field)].Float64())
	default:
		logrus.WithField("field", field).Errorf("Field is of unhandled type %T", v)
		return 0
	}
}

func newFieldTracker(sources []lmaelastic.AggCompositeSourceInfo) *fieldTracker {
	t := fieldTracker{fieldToIndex: map[string]int{}}
	for idx, source := range sources {
		t.fieldToIndex[source.Field] = idx
	}
	return &t
}

func newTimeRangeQuery(from, to time.Time) elastic.Query {
	return elastic.NewRangeQuery("end_time").Gt(from.Unix()).Lte(to.Unix())
}
