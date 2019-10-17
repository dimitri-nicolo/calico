// Copyright (c) 2019 Tigera Inc. All rights reserved.

package elastic

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

func TestInterval_MarshalJSON(t *testing.T) {
	f := func(input time.Duration, expected string) func(*testing.T) {
		return func(t *testing.T) {
			g := NewWithT(t)

			b, err := json.Marshal(Interval{input})
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(string(b)).Should(Equal(fmt.Sprintf("%q", expected)))
		}
	}

	t.Run("5 seconds", f(time.Second*5, "5s"))
	t.Run("5.5 seconds", f(time.Second*5+500*time.Millisecond, "5s"))
}

func TestInterval_UnmarshalJSON(t *testing.T) {
	f := func(input string, expected time.Duration) func(*testing.T) {
		return func(t *testing.T) {
			g := NewWithT(t)

			var i Interval
			err := json.Unmarshal([]byte(fmt.Sprintf("%q", input)), &i)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(i.Duration).Should(Equal(expected))
		}
	}
	g := func(input string) func(*testing.T) {
		return func(t *testing.T) {
			g := NewWithT(t)

			var i Interval
			err := json.Unmarshal([]byte(fmt.Sprintf("%q", input)), &i)
			g.Expect(err).Should(HaveOccurred())
		}
	}

	t.Run("1", f("5", time.Second*5))
	t.Run("5s", f("5s", time.Second*5))
	t.Run("5m", f("5m", time.Minute*5))
	t.Run("5h", f("5h", time.Hour*5))
	t.Run("5d", f("5d", time.Hour*24*5))
	t.Run("5w", f("5w", time.Hour*24*7*5))

	t.Run("s", g("s"))
	t.Run("1y", g("1y"))
}

func TestCondition_MarshalJSON(t *testing.T) {
	f := func(orig Condition) func(*testing.T) {
		return func(t *testing.T) {
			g := NewWithT(t)

			b, err := json.Marshal(orig)
			g.Expect(err).ShouldNot(HaveOccurred())

			var new Condition
			err = json.Unmarshal(b, &new)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(new).Should(Equal(orig))
		}
	}

	t.Run("empty", f(Condition{}))
	t.Run("always", f(Condition{Always: true}))
	t.Run("never", f(Condition{Never: true}))
	t.Run("comparison", f(Condition{Compare: &Comparison{"a", "eq", "b"}}))
	t.Run("array_compare", f(Condition{ArrayCompare: &ArrayCompare{
		ArrayPath:  "a",
		Path:       "b",
		Quantifier: "c",
		Value:      "d",
	}}))

}

func TestComparison_UnmarshalJSON(t *testing.T) {
	f := func(input string, expected *Comparison, ok bool) func(*testing.T) {
		return func(t *testing.T) {
			g := NewWithT(t)

			var v Comparison
			err := json.Unmarshal([]byte(input), &v)
			if ok {
				g.Expect(err).ShouldNot(HaveOccurred())
				g.Expect(v).Should(Equal(*expected))
			} else {
				g.Expect(err).Should(HaveOccurred())
			}
		}
	}

	t.Run("ok", f(`{"a": {"gte": "b"}}`, &Comparison{"a", "gte", "b"}, true))
	t.Run("empty", f(``, nil, false))
	t.Run("empty object", f(`{}`, nil, false))
	t.Run("bad key type", f(`{1: {"gte": "b"}}`, nil, false))
	t.Run("bad operation type", f(`{"a": {1: "b"}}`, nil, false))
	t.Run("2 keys", f(`{"a": {"gte": "d"}, "c": {"ge", "e"}}`, nil, false))
	t.Run("2 operations", f(`{"a": {"gte": "b", "lt": "c"}}`, nil, false))
}

func TestArrayCompare_UnmarshalJSON(t *testing.T) {
	f := func(input string, expected *ArrayCompare, ok bool) func(*testing.T) {
		return func(t *testing.T) {
			g := NewWithT(t)

			var v ArrayCompare
			err := json.Unmarshal([]byte(input), &v)
			if ok {
				g.Expect(err).ShouldNot(HaveOccurred())
				g.Expect(v).Should(Equal(*expected))
			} else {
				g.Expect(err).Should(HaveOccurred())
			}
		}
	}

	t.Run("ok", f(`{"a":{"path": "b", "c": {"value": "d"}}}`,
		&ArrayCompare{"a", "b", "c", "d"}, true))
	t.Run("empty", f("", nil, false))
	t.Run("empty object", f("{}", nil, false))
	t.Run("missing path", f(`{"a": {"c": {"value": "d"}}}`, nil, false))
	t.Run("missing path 2", f(`{"a": {"c": {"value": "d"}, "e": "f"}}`, nil, false))
	t.Run("bad path type", f(`{"a": {"path": 1, "gte": {"value": "c"}}}`, nil, false))
	t.Run("missing comparison", f(`{"a":{"path": "b"}}`, nil, false))
	t.Run("bad comparison type", f(`{"a":{"path": "b", "c": "d"}}`, nil, false))
	t.Run("missing value", f(`{"a":{"path": "b", "c": {}}}`, nil, false))
}
