package searcher

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/tigera/intrusion-detection/controller/pkg/db"
	"github.com/tigera/intrusion-detection/controller/pkg/feed"
	"github.com/tigera/intrusion-detection/controller/pkg/statser"
)

// TestDoIPSet tests the case where everything is working
func TestDoIPSet(t *testing.T) {
	expected := []db.FlowLog{
		{
			SourceIP:   "1.2.3.4",
			SourceName: "source",
			DestIP:     "2.3.4.5",
			DestName:   "dest",
			StartTime:  1,
			EndTime:    2,
		},
		{
			SourceIP:   "5.6.7.8",
			SourceName: "source",
			DestIP:     "2.3.4.5",
			DestName:   "dest",
			StartTime:  2,
			EndTime:    3,
		},
	}
	runTest(t, true, expected, nil, -1, -1)
}

// TestDoIPSetNoResults tests the case where no results are returned
func TestDoIPSetNoResults(t *testing.T) {
	expected := []db.FlowLog{}
	runTest(t, true, expected, nil, -1, -1)
}

// TestDoIPSetSuspiciousIPFails tests the case where suspiciousIP fails after the first result
func TestDoIPSetSuspiciousIPFails(t *testing.T) {
	expected := []db.FlowLog{}
	runTest(t, false, expected, errors.New("fail"), -1, -1)
}

func TestDoIPSetSuspiciousIPIterationFails(t *testing.T) {
	expected := []db.FlowLog{
		{
			SourceIP:   "1.2.3.4",
			SourceName: "source",
			DestIP:     "2.3.4.5",
			DestName:   "dest",
			StartTime:  1,
			EndTime:    2,
		},
		{
			SourceIP:   "5.6.7.8",
			SourceName: "source",
			DestIP:     "2.3.4.5",
			DestName:   "dest",
			StartTime:  2,
			EndTime:    3,
		},
	}
	runTest(t, false, expected, nil, 1, -1)
}

// TestDoIPSetEventsFails tests the case where the first call to events.PutFlowLog fails but the second does not
func TestDoIPSetEventsFails(t *testing.T) {
	expected := []db.FlowLog{
		{
			SourceIP:   "1.2.3.4",
			SourceName: "source",
			DestIP:     "2.3.4.5",
			DestName:   "dest",
			StartTime:  1,
			EndTime:    2,
		},
		{
			SourceIP:   "5.6.7.8",
			SourceName: "source",
			DestIP:     "2.3.4.5",
			DestName:   "dest",
			StartTime:  2,
			EndTime:    3,
		},
	}
	runTest(t, false, expected, nil, -1, 0)
}

func runTest(t *testing.T, successful bool, expected []db.FlowLog, err error, suspiciousErrorIdx, eventsErrorIdx int) {
	f := feed.NewFeed("test", "test-namespace")
	suspiciousIP := &mockDB{err: err, errorIdx: suspiciousErrorIdx, flowLogs: expected}
	events := &mockDB{errorIdx: eventsErrorIdx}
	searcher := NewFlowSearcher(f, 0, suspiciousIP, events).(*flowSearcher)

	ctx := context.TODO()
	s := statser.NewStatser()

	searcher.doIPSet(ctx, s)

	if successful {
		if !reflect.DeepEqual(expected, events.flowLogs) && !(len(expected) == 0 && len(events.flowLogs) == 0) {
			t.Errorf("Logs in DB mismatch: %v != %v", expected, events.flowLogs)
		}
		if len(suspiciousIP.flowLogs) != 0 {
			t.Errorf("Did not consume all flowLogs from suspiciousIP: %v", suspiciousIP.flowLogs)
		}
	} else {
		if eventsErrorIdx >= 0 && len(events.flowLogs) != len(expected)-1 {
			t.Errorf("Logs in DB mismatch: %v (skip %d) != %v", expected, eventsErrorIdx, events.flowLogs)
		}
		if suspiciousErrorIdx >= 0 && len(events.flowLogs) != suspiciousErrorIdx {
			t.Errorf("Logs in DB mismatch: %v != %v", expected[:suspiciousErrorIdx], events.flowLogs)
		}
	}

	status := s.Status()
	if !status.LastSuccessfulSync.Equal(time.Time{}) {
		t.Errorf("Sync was marked as successful when it should not have been.")
	}
	if successful {
		if status.LastSuccessfulSearch.Equal(time.Time{}) {
			t.Errorf("Search was not marked as successful.")
		}
		if len(status.ErrorConditions) > 0 {
			t.Errorf("Status errors reported: %v", status.ErrorConditions)
		}
	} else {
		if !status.LastSuccessfulSearch.Equal(time.Time{}) {
			t.Errorf("Search was marked as successful when it should not have been.")
		}
		if len(status.ErrorConditions) == 0 {
			t.Errorf("No status errors reported.")
		}
	}
}

type mockDB struct {
	err           error
	errorIdx      int
	errorReturned bool
	flowLogs      []db.FlowLog
	value         db.FlowLog
}

func (m *mockDB) QueryIPSet(ctx context.Context, name string) (db.FlowLogIterator, error) {
	return m, m.err
}

func (m *mockDB) Next() bool {
	if len(m.flowLogs) == m.errorIdx {
		return false
	}
	if len(m.flowLogs) > 0 {
		m.value = m.flowLogs[0]
		m.flowLogs = m.flowLogs[1:]
		return true
	}
	return false
}

func (m *mockDB) Value() db.FlowLog {
	return m.value
}

func (m *mockDB) Err() error {
	if m.errorIdx >= 0 {
		return errors.New("Err error")
	}
	return nil
}

func (m *mockDB) PutFlowLog(ctx context.Context, l db.FlowLog) error {
	if len(m.flowLogs) == m.errorIdx && !m.errorReturned {
		m.errorReturned = true
		return errors.New("PutFlowLog error")
	}
	m.flowLogs = append(m.flowLogs, l)
	return nil
}
