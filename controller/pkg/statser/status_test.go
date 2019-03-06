package statser

import (
	"reflect"
	"testing"
	"time"
)

func TestStatusClearError(t *testing.T) {
	expected := []ErrorCondition{
		{
			Type:    "keep",
			Message: "should be kept",
		},
	}
	status := &Status{
		ErrorConditions:      append(expected, ErrorCondition{"remove", "should not be kept"}),
	}

	status.ClearError("remove")

	if !reflect.DeepEqual(expected, status.ErrorConditions) {
		t.Errorf("Output mismatch: %v != %v", expected, status.ErrorConditions)
	}
}

func TestStatusSuccessfulSearch(t *testing.T) {
	status := &Status{}
	status.SuccessfulSearch()
	if status.LastSuccessfulSearch.Equal(time.Time{}) {
		t.Errorf("LastSuccessfulSearch not set")
	}
	if !status.LastSuccessfulSync.Equal(time.Time{}) {
		t.Errorf("LastSuccessfulSync set to %v", status.LastSuccessfulSync)
	}
}

func TestStatusSuccessfulSync(t *testing.T) {
	status := &Status{}
	status.SuccessfulSync()
	if status.LastSuccessfulSync.Equal(time.Time{}) {
		t.Errorf("LastSuccessfulSync not set")
	}
	if !status.LastSuccessfulSearch.Equal(time.Time{}) {
		t.Errorf("LastSuccessfulSearch set to %v", status.LastSuccessfulSearch)
	}
}
