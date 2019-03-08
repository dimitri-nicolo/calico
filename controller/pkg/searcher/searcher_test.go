package searcher

import (
	"context"
	"errors"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/tigera/intrusion-detection/controller/pkg/db"
	"github.com/tigera/intrusion-detection/controller/pkg/events"
	"github.com/tigera/intrusion-detection/controller/pkg/feed"
	"github.com/tigera/intrusion-detection/controller/pkg/statser"
	"github.com/tigera/intrusion-detection/controller/pkg/util"
)

// TestDoIPSet tests the case where everything is working
func TestDoIPSet(t *testing.T) {
	expected := []events.SecurityEvent{
		{
			SourceIP:   util.Sptr("1.2.3.4"),
			SourceName: "source",
			DestIP:     util.Sptr("2.3.4.5"),
			DestName:   "dest",
		},
		{
			SourceIP:   util.Sptr("5.6.7.8"),
			SourceName: "source",
			DestIP:     util.Sptr("2.3.4.5"),
			DestName:   "dest",
		},
	}
	runTest(t, true, expected, nil, -1, -1)
}

// TestDoIPSetNoResults tests the case where no results are returned
func TestDoIPSetNoResults(t *testing.T) {
	expected := []events.SecurityEvent{}
	runTest(t, true, expected, nil, -1, -1)
}

// TestDoIPSetSuspiciousIPFails tests the case where suspiciousIP fails after the first result
func TestDoIPSetSuspiciousIPFails(t *testing.T) {
	expected := []events.SecurityEvent{}
	runTest(t, false, expected, errors.New("fail"), -1, -1)
}

func TestDoIPSetSuspiciousIPIterationFails(t *testing.T) {
	expected := []events.SecurityEvent{
		{
			SourceIP:   util.Sptr("1.2.3.4"),
			SourceName: "source",
			DestIP:     util.Sptr("2.3.4.5"),
			DestName:   "dest",
		},
		{
			SourceIP:   util.Sptr("5.6.7.8"),
			SourceName: "source",
			DestIP:     util.Sptr("2.3.4.5"),
			DestName:   "dest",
		},
	}
	runTest(t, false, expected, nil, 1, -1)
}

// TestDoIPSetEventsFails tests the case where the first call to events.PutFlowLog fails but the second does not
func TestDoIPSetEventsFails(t *testing.T) {
	expected := []events.SecurityEvent{
		{
			SourceIP:   util.Sptr("1.2.3.4"),
			SourceName: "source",
			DestIP:     util.Sptr("2.3.4.5"),
			DestName:   "dest",
		},
		{
			SourceIP:   util.Sptr("5.6.7.8"),
			SourceName: "source",
			DestIP:     util.Sptr("2.3.4.5"),
			DestName:   "dest",
		},
	}
	runTest(t, false, expected, nil, -1, 0)
}

func runTest(t *testing.T, successful bool, expected []events.SecurityEvent, err error, suspiciousErrorIdx, eventsErrorIdx int) {
	g := NewGomegaWithT(t)

	f := feed.NewFeed("test", "test-namespace")
	suspiciousIP := &mockDB{err: err, errorIdx: suspiciousErrorIdx, flowLogs: expected}
	events := &mockDB{errorIdx: eventsErrorIdx, flowLogs: []events.SecurityEvent{}}
	searcher := NewFlowSearcher(f, 0, suspiciousIP, events).(*flowSearcher)

	ctx := context.TODO()
	s := statser.NewStatser()

	searcher.doIPSet(ctx, s)

	if successful {
		g.Expect(events.flowLogs).Should(Equal(expected), "Logs in DB should match expected")
		g.Expect(suspiciousIP.flowLogs).Should(HaveLen(0), "All flowLogs from suspiciousIP were consumed")
	} else {
		if eventsErrorIdx >= 0 {
			g.Expect(events.flowLogs).Should(HaveLen(len(expected)-1), "Logs in DB should have skipped 1 from input")
		}
		if suspiciousErrorIdx >= 0 {
			g.Expect(events.flowLogs).Should(HaveLen(suspiciousErrorIdx), "Logs in DB should stop at the first error")
		}
	}

	status := s.Status()
	g.Expect(status.LastSuccessfulSync).Should(Equal(time.Time{}), "Sync should not be marked as successful")
	if successful {
		g.Expect(status.LastSuccessfulSearch).ShouldNot(Equal(time.Time{}), "Search should be marked as successful")
		g.Expect(status.ErrorConditions).Should(HaveLen(0), "No errors should be reported")
	} else {
		g.Expect(status.LastSuccessfulSearch).Should(Equal(time.Time{}), "Search should be not marked as successful")
		g.Expect(status.ErrorConditions).ShouldNot(HaveLen(0), "Errors should be reported")
	}
}

type mockDB struct {
	err           error
	errorIdx      int
	errorReturned bool
	flowLogs      []events.SecurityEvent
	value         events.SecurityEvent
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

func (m *mockDB) Value() events.SecurityEvent {
	return m.value
}

func (m *mockDB) Err() error {
	if m.errorIdx >= 0 {
		return errors.New("Err error")
	}
	return nil
}

func (m *mockDB) PutFlowLog(ctx context.Context, l events.SecurityEvent) error {
	if len(m.flowLogs) == m.errorIdx && !m.errorReturned {
		m.errorReturned = true
		return errors.New("PutFlowLog error")
	}
	m.flowLogs = append(m.flowLogs, l)
	return nil
}
