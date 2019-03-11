package sync

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/tigera/intrusion-detection/controller/pkg/feed"
	"github.com/tigera/intrusion-detection/controller/pkg/statser"
)

func TestSync(t *testing.T) {
	expected := feed.IPSet{
		"1.2.3.4",
		"2.3.4.5",
	}
	testSync(t, true, expected, nil)
}

func TestSyncEmpty(t *testing.T) {
	expected := feed.IPSet{}
	testSync(t, true, expected, nil)
}

func TestSyncError(t *testing.T) {
	expected := feed.IPSet{}
	testSync(t, false, expected, errors.New("test error"))
}

func testSync(t *testing.T, successful bool, expected feed.IPSet, err error) {
	feed := feed.NewFeed("test", "test-namespace")
	ipSet := &mockDB{
		err: err,
	}

	syncer := NewSyncer(feed, ipSet).(*syncer)

	statser := statser.NewStatser()
	failFunc := &mockFailFunc{}

	syncer.sync(context.TODO(), expected, failFunc.Fail, statser, 1, 0)

	if !reflect.DeepEqual(expected, ipSet.set) {
		t.Errorf("Feed contents mismatch: %v != %v", expected, ipSet.set)
	}

	status := statser.Status()
	if successful {
		if status.LastSuccessfulSync.Equal(time.Time{}) {
			t.Errorf("Sync was not marked as successful when it should have been.")
		}
		if failFunc.called {
			t.Errorf("Fail function was called when it should not have been")
		}
	} else {
		if !status.LastSuccessfulSync.Equal(time.Time{}) {
			t.Errorf("Sync was marked as successful when it should not have been.")
		}
		if !failFunc.called {
			t.Errorf("Fail function was not called when it should have been")
		}
	}
	if !status.LastSuccessfulSearch.Equal(time.Time{}) {
		t.Errorf("Search was marked as successful when it should not have been.")
	}
	if err == nil {
		if len(status.ErrorConditions) != 0 {
			t.Errorf("Status errors reported: %v", status.ErrorConditions)
		}
	} else {
		if len(status.ErrorConditions) == 0 {
			t.Errorf("Status error not reported.")
		} else {
			if status.ErrorConditions[0].Type != statserType {
				t.Errorf("ErrorConditions type mismatch: %v != %v", status.ErrorConditions[0].Type, statserType)
			}
		}
	}
}

type mockFailFunc struct {
	called bool
}

func (m *mockFailFunc) Fail() {
	m.called = true
}

type mockDB struct {
	name string
	set  feed.IPSet
	err  error
}

func (m *mockDB) GetIPSet(name string) ([]string, error) {
	panic("not implemented")
}

func (m *mockDB) PutIPSet(ctx context.Context, name string, set feed.IPSet) error {
	m.name = name
	m.set = set

	return m.err
}
