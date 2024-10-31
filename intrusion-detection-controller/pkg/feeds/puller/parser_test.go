// Copyright (c) 2019 Tigera Inc. All rights reserved.

package puller

import (
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/util"
)

type entryHandlerInput struct {
	n int
	l string
}

func TestGetParserForFormat(t *testing.T) {
	f := func(fmt v3.ThreatFeedFormat, input string, output []entryHandlerInput) func(t *testing.T) {
		return func(t *testing.T) {
			g := NewWithT(t)
			r := strings.NewReader(input)
			c := 0
			h := func(n int, l string) {
				g.Expect(n).Should(Equal(output[c].n))
				g.Expect(l).Should(Equal(output[c].l))
				c++
			}

			err := getParserForFormat(fmt)(r, h)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(c).Should(Equal(len(output)))
		}
	}

	t.Run("json", f(v3.ThreatFeedFormat{JSON: &v3.ThreatFeedFormatJSON{Path: "$"}},
		`["a", "b"]`, []entryHandlerInput{{1, "a"}, {2, "b"}}))
	t.Run("csv", f(v3.ThreatFeedFormat{CSV: &v3.ThreatFeedFormatCSV{FieldNum: util.UintPtr(0)}},
		"a,b\nc,d", []entryHandlerInput{{1, "a"}, {2, "c"}}))
	t.Run("newlineDelimited", f(v3.ThreatFeedFormat{NewlineDelimited: &v3.ThreatFeedFormatNewlineDelimited{}},
		"a\nb", []entryHandlerInput{{1, "a"}, {2, "b"}}))
	t.Run("default", f(v3.ThreatFeedFormat{},
		"a\nb", []entryHandlerInput{{1, "a"}, {2, "b"}}))
}

func TestJSONParser(t *testing.T) {
	f := func(path, input string, output []entryHandlerInput, ok bool) func(t *testing.T) {
		return func(t *testing.T) {
			g := NewWithT(t)
			r := strings.NewReader(input)
			c := 0
			h := func(n int, l string) {
				g.Expect(n).Should(Equal(output[c].n))
				g.Expect(l).Should(Equal(output[c].l))
				c++
			}

			err := jsonParser(path)(r, h)
			if ok {
				g.Expect(err).ShouldNot(HaveOccurred())
				g.Expect(c).Should(Equal(len(output)))
			} else {
				g.Expect(err).Should(HaveOccurred())
			}
		}
	}

	t.Run("empty input", f("$", ``, nil, false))
	t.Run("invalid json", f("$", `{`, nil, false))
	t.Run("invalid jsonpath", f("", `{}`, nil, false))
	t.Run("empty input", f("$", `[]`, nil, true))
	t.Run("not an array", f("$", `{}`, nil, false))
	t.Run("array", f("$", `["a", "b"]`, []entryHandlerInput{{1, "a"}, {2, "b"}}, true))
	t.Run("path to array", f("$.a", `{"a": ["a", "b"]}`,
		[]entryHandlerInput{{1, "a"}, {2, "b"}}, true))
	t.Run("array with invalid objects", f("$",
		`["a", 1, "b", {}, [2]]`,
		[]entryHandlerInput{{1, "a"}, {3, "b"}}, true))
}

func TestCSVParser(t *testing.T) {
	f := func(fmt *v3.ThreatFeedFormatCSV, input string, output []entryHandlerInput, ok bool) func(t *testing.T) {
		return func(t *testing.T) {
			g := NewWithT(t)
			r := strings.NewReader(input)
			c := 0
			h := func(n int, l string) {
				g.Expect(n).Should(Equal(output[c].n))
				g.Expect(l).Should(Equal(output[c].l))
				c++
			}

			err := csvParser(fmt)(r, h)
			if ok {
				g.Expect(err).ShouldNot(HaveOccurred())
				g.Expect(c).Should(Equal(len(output)))
			} else {
				g.Expect(err).Should(HaveOccurred())
			}
		}
	}

	t.Run("empty input", f(&v3.ThreatFeedFormatCSV{FieldNum: util.UintPtr(0)}, "", nil, true))
	t.Run("only linebreak", f(&v3.ThreatFeedFormatCSV{FieldNum: util.UintPtr(0)}, "\n", nil, true))
	t.Run("only header", f(
		&v3.ThreatFeedFormatCSV{FieldNum: util.UintPtr(0), Header: true}, "a,b\n", nil, true))
	t.Run("fieldNum=1", f(
		&v3.ThreatFeedFormatCSV{FieldNum: util.UintPtr(1)},
		"foo,bar\na,b", []entryHandlerInput{{1, "bar"}, {2, "b"}}, true))
	t.Run("fieldNum and header", f(
		&v3.ThreatFeedFormatCSV{FieldNum: util.UintPtr(0), Header: true},
		"a,b\nfoo,bar", []entryHandlerInput{{1, "foo"}}, true))
	t.Run("fieldName and header", f(
		&v3.ThreatFeedFormatCSV{FieldName: "b", Header: true},
		"a,b\nfoo,bar\nc,d", []entryHandlerInput{{1, "bar"}, {2, "d"}}, true))
	t.Run("alternate column delimiter", f(
		&v3.ThreatFeedFormatCSV{FieldNum: util.UintPtr(1), ColumnDelimiter: "|"},
		"a|b", []entryHandlerInput{{1, "b"}}, true))
	t.Run("comment", f(
		&v3.ThreatFeedFormatCSV{FieldNum: util.UintPtr(1), CommentDelimiter: "#", Header: true},
		"#comment\na,b\nfoo,bar", []entryHandlerInput{{1, "bar"}}, true))
	t.Run("uneven record sizes", f(
		&v3.ThreatFeedFormatCSV{FieldNum: util.UintPtr(0)},
		"a,b\nc,d,e", []entryHandlerInput{{1, "a"}}, false))
	t.Run("disabled record size validation", f(
		&v3.ThreatFeedFormatCSV{FieldNum: util.UintPtr(0), DisableRecordSizeValidation: true},
		"a,b\nc,d,e", []entryHandlerInput{{1, "a"}, {2, "c"}}, true))
	t.Run("correct record size", f(
		&v3.ThreatFeedFormatCSV{FieldNum: util.UintPtr(0), RecordSize: 2},
		"a,b\nc,d", []entryHandlerInput{{1, "a"}, {2, "c"}}, true))
	t.Run("incorrect record size", f(
		&v3.ThreatFeedFormatCSV{FieldNum: util.UintPtr(0), RecordSize: 2},
		"a,b,c", nil, false))
}

func TestNewlineDelimitedParser(t *testing.T) {
	f := func(input string, output []entryHandlerInput) func(t *testing.T) {
		return func(t *testing.T) {
			g := NewWithT(t)
			r := strings.NewReader(input)
			c := 0
			h := func(n int, l string) {
				g.Expect(n).Should(Equal(output[c].n))
				g.Expect(l).Should(Equal(output[c].l))
				c++
			}

			err := newlineDelimitedParser(r, h)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(c).Should(Equal(len(output)))
		}
	}

	t.Run("empty input", f("", nil))
	t.Run("strip whitespace", f(" foo ", []entryHandlerInput{{1, "foo"}}))
	t.Run("filter comments", f("#a\nfoo\n#b\nbar", []entryHandlerInput{{2, "foo"}, {4, "bar"}}))
	t.Run("filter blank lines", f("\nfoo\n\n\nbar\n\n\n", []entryHandlerInput{{2, "foo"}, {5, "bar"}}))
}
