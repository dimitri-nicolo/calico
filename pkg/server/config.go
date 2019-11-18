package server

import (
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
)

const (
	configUpdatePeriod = 10 * time.Second
)

// getReportTypes returns a map of the ReportTypeSpec against the report name. If the cache is out of date or
// has not yet been initialized, the map is updated from source. The returned map should not be modified.
func (s *server) getReportTypes() (map[string]*v3.ReportTypeSpec, error) {
	if rt := s.getStoredReportTypes(); rt != nil {
		return rt, nil
	}

	// The report types have either not been intialized, or need refreshing. Grab the full RW lock and update the
	// report types.
	s.reportLock.Lock()
	defer s.reportLock.Unlock()
	if s.reportTypes != nil && time.Now().Sub(s.lastUpdate) < configUpdatePeriod {
		// Another request must have pipped up to the post, no need to refresh.
		return s.reportTypes, nil
	}

	// Get the latest set of report types.
	grt, err := s.rcg.GlobalReportTypes().List(v1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// Transfer the specs into a new map keyed off the name.
	s.reportTypes = make(map[string]*v3.ReportTypeSpec, 0)
	for idx := range grt.Items {
		s.reportTypes[grt.Items[idx].Name] = &grt.Items[idx].Spec
	}
	return s.reportTypes, nil
}

// getStoredReportTypes returns the stored report types or nil if they are out of date. This is used internally
// by getReportTypes.
func (s *server) getStoredReportTypes() map[string]*v3.ReportTypeSpec {
	s.reportLock.RLock()
	defer s.reportLock.RUnlock()

	if time.Now().Sub(s.lastUpdate) > configUpdatePeriod {
		return nil
	}
	return s.reportTypes
}
