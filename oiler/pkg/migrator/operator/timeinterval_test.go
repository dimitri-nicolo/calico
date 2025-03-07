// Copyright (c) 2025 Tigera, Inc. All rights reserved.

package operator_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/projectcalico/calico/oiler/pkg/migrator/operator"
)

func TestNext(t *testing.T) {
	type args struct {
		cursor            map[string]interface{}
		lastGeneratedTime *time.Time
		current           *time.Time
	}
	tests := []struct {
		name string
		args args
		want *operator.TimeInterval
	}{
		{
			name: "Paginate through data",
			args: args{
				cursor:            map[string]interface{}{"search_after": []string{"1", "2"}},
				lastGeneratedTime: ptrTime(time.Unix(1, 0).UTC()),
				current:           nil,
			},
			want: &operator.TimeInterval{Cursor: map[string]interface{}{"search_after": []string{"1", "2"}}, Start: nil},
		},
		{
			name: "Move to next interval",
			args: args{
				cursor:            nil,
				lastGeneratedTime: ptrTime(time.Unix(1, 0).UTC()),
				current:           nil,
			},
			want: &operator.TimeInterval{Cursor: nil, Start: ptrTime(time.Unix(1, 0).UTC())},
		},
		{
			name: "Redo query with the same values as no new data has been written",
			args: args{
				cursor:            nil,
				lastGeneratedTime: nil,
				current:           ptrTime(time.Unix(1, 0).UTC()),
			},
			want: &operator.TimeInterval{Cursor: nil, Start: ptrTime(time.Unix(1, 0).UTC())},
		},
		{
			name: "Paginate through current interval",
			args: args{
				cursor:            map[string]interface{}{"search_after": []string{"1", "2"}},
				lastGeneratedTime: ptrTime(time.Unix(2, 0).UTC()),
				current:           ptrTime(time.Unix(1, 0).UTC()),
			},
			want: &operator.TimeInterval{Cursor: map[string]interface{}{"search_after": []string{"1", "2"}}, Start: ptrTime(time.Unix(1, 0).UTC())},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := operator.Next(tt.args.cursor, tt.args.lastGeneratedTime, tt.args.current); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Next() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTimeInterval_LastGeneratedTime(t *testing.T) {
	tests := []struct {
		name   string
		Cursor map[string]interface{}
		Start  *time.Time
		want   time.Time
	}{
		{
			name:   "Last generated time from pagination",
			Cursor: map[string]interface{}{"searchFrom": []interface{}{"1.7e+2", "2"}},
			want:   time.Unix(170, 0).UTC(),
		},
		{
			name:   "Last generated time from pagination with start",
			Cursor: map[string]interface{}{"searchFrom": []interface{}{"", ""}},
			Start:  ptrTime(time.Unix(1, 0).UTC()),
			want:   time.Unix(1, 0).UTC(),
		},
		{
			name:   "Malformed",
			Cursor: map[string]interface{}{"searchFrom": []interface{}{"", ""}},
			want:   time.Time{},
		},
		{
			name:  "Last generated time from start of interval",
			Start: ptrTime(time.Unix(1, 0).UTC()),
			want:  time.Unix(1, 0).UTC(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			it := &operator.TimeInterval{
				Cursor: tt.Cursor,
				Start:  tt.Start,
			}
			if got := it.LastGeneratedTime(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("LastGeneratedTime() = %v, want %v", got, tt.want)
			}
		})
	}
}
func ptrTime(time time.Time) *time.Time {
	return &time
}
