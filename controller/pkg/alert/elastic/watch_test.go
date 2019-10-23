// Copyright (c) 2019 Tigera Inc. All rights reserved.

package elastic

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	libcalicov3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	v3 "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/tigera/intrusion-detection/controller/pkg/elastic"
)

func TestCondition(t *testing.T) {
	f := func(field, metric, condition string, aggs []string) func(*testing.T) {
		return func(t *testing.T) {
			g := NewWithT(t)

			c := Condition(v3.GlobalAlert{
				Spec: libcalicov3.GlobalAlertSpec{
					AggregateBy: aggs,
					Metric:      metric,
					Field:       field,
					Condition:   condition,
					Threshold:   5,
				},
			})

			g.Expect(c).ShouldNot(BeNil())
			b, _ := json.MarshalIndent(c, "", "  ")
			fmt.Println(string(b))
		}
	}

	aggs := []string{"a", "b", "c"}
	for _, metric := range []string{"", libcalicov3.GlobalAlertMetricCount,
		libcalicov3.GlobalAlertMetricAvg, libcalicov3.GlobalAlertMetricSum,
		libcalicov3.GlobalAlertMetrixMin, libcalicov3.GlobalAlertMetricMax} {

		field := "d"
		if metric == libcalicov3.GlobalAlertMetricCount {
			field = ""
		}

		for i := 0; i < len(aggs)+1; i++ {
			for comparator := range comparatorMap {
				t.Run(fmt.Sprintf("%s.%s.%s", metric, comparator, strings.Join(aggs[:i], "")), f(field, metric, comparator, aggs[:i]))
			}
		}
	}
}

func TestTransform(t *testing.T) {
	f := func(field, metric, condition string, aggs []string) func(*testing.T) {
		return func(t *testing.T) {
			g := NewWithT(t)

			c := Transform(v3.GlobalAlert{
				Spec: libcalicov3.GlobalAlertSpec{
					AggregateBy: aggs,
					Metric:      metric,
					Field:       field,
					Condition:   condition,
					Threshold:   5,
				},
			})

			g.Expect(c).ShouldNot(BeNil())
			b, _ := json.MarshalIndent(c, "", "  ")
			fmt.Println(string(b))
		}
	}

	aggs := []string{"a", "b", "c"}
	for _, metric := range []string{"", libcalicov3.GlobalAlertMetricCount,
		libcalicov3.GlobalAlertMetricAvg, libcalicov3.GlobalAlertMetricSum,
		libcalicov3.GlobalAlertMetrixMin, libcalicov3.GlobalAlertMetricMax} {

		field := "d"
		if metric == libcalicov3.GlobalAlertMetricCount {
			field = ""
		}

		for i := 0; i < len(aggs)+1; i++ {
			for comparator := range comparatorMap {
				t.Run(fmt.Sprintf("%s.%s.%s", metric, comparator, strings.Join(aggs[:i], "")), f(field, metric, comparator, aggs[:i]))
			}
		}
	}
}

func TestTrigger(t *testing.T) {
	g := NewWithT(t)

	period := 123 * time.Second
	g.Expect(Trigger(period).Schedule.Interval.Duration).Should(BeNumerically("==", period))
}

func TestPeriod(t *testing.T) {
	f := func(i, e time.Duration) func(*testing.T) {
		return func(t *testing.T) {
			g := NewWithT(t)

			g.Expect(Period(v3.GlobalAlert{Spec: libcalicov3.GlobalAlertSpec{Period: &v1.Duration{i}}})).Should(Equal(e))
		}
	}

	t.Run("zero", f(0, DefaultPeriod))
	t.Run("negative", f(-1, DefaultPeriod))
	t.Run("positive", f(1, 1))
}

func TestLookback(t *testing.T) {
	f := func(i, e time.Duration) func(*testing.T) {
		return func(t *testing.T) {
			g := NewWithT(t)

			g.Expect(Lookback(v3.GlobalAlert{Spec: libcalicov3.GlobalAlertSpec{Lookback: &v1.Duration{i}}})).Should(Equal(e))
		}
	}

	t.Run("zero", f(0, DefaultLookback))
	t.Run("negative", f(-1, DefaultLookback))
	t.Run("positive", f(1, 1))
}

func TestInput(t *testing.T) {
	f := func(alert v3.GlobalAlert, hasAgg, ok bool) func(*testing.T) {
		return func(t *testing.T) {
			g := NewWithT(t)

			input, err := Input(alert)
			if ok {
				g.Expect(err).ShouldNot(HaveOccurred())
				g.Expect(input).ShouldNot(BeNil())

				g.Expect(input.Simple).Should(BeNil())
				g.Expect(input.Search).ShouldNot(BeNil())
				g.Expect(input.Search.Request.Indices).ShouldNot((HaveLen(0)))

				g.Expect(input.Search.Request.Body).Should(HaveKey("query"))
				g.Expect(input.Search.Request.Body).Should(HaveKey("size"))

				if hasAgg {
					g.Expect(input.Search.Request.Body).Should(HaveKey("aggs"))
					g.Expect(input.Search.Request.Body.(JsonObject)["size"]).Should(BeNumerically("==", 0))
				} else {
					g.Expect(input.Search.Request.Body).ShouldNot(HaveKey("aggs"))
					g.Expect(input.Search.Request.Body.(JsonObject)["size"]).Should(BeNumerically("==", QuerySize))
				}
			} else {
				g.Expect(err).Should(HaveOccurred())
			}
		}
	}

	t.Run("bad dataset", f(v3.GlobalAlert{
		Spec: libcalicov3.GlobalAlertSpec{
			DataSet: "fail",
		},
	}, false, false))
	t.Run("bad query", f(v3.GlobalAlert{
		Spec: libcalicov3.GlobalAlertSpec{
			DataSet: "audit",
			Query:   "AND",
		},
	}, false, false))
	t.Run("good query", f(v3.GlobalAlert{
		Spec: libcalicov3.GlobalAlertSpec{
			DataSet: "audit",
		},
	}, false, true))
	t.Run("aggs ", f(v3.GlobalAlert{
		Spec: libcalicov3.GlobalAlertSpec{
			DataSet:     "audit",
			AggregateBy: []string{"abc"},
		},
	}, true, true))
}

func TestLookbackFilter(t *testing.T) {
	g := NewWithT(t)

	field := "foo"
	lf := LookbackFilter(123456*time.Millisecond, field)
	g.Expect(lf).Should(HaveKey("range"))
	g.Expect(lf["range"].(JsonObject)[field]).Should(HaveKey("gte"))
	g.Expect(lf["range"].(JsonObject)[field].(JsonObject)["gte"]).Should(HaveSuffix("||-123s"))
}

func TestIndices(t *testing.T) {
	f := func(dataSet string, expected []string, ok bool) func(*testing.T) {
		return func(*testing.T) {
			g := NewWithT(t)

			actual, err := Indices(dataSet)
			if ok {
				g.Expect(err).ShouldNot(HaveOccurred())
				g.Expect(actual).Should(ConsistOf(expected))
			} else {
				g.Expect(err).Should(HaveOccurred())
			}
		}
	}

	t.Run("audit", f("audit", []string{elastic.AuditIndex}, true))
	t.Run("dns", f("dns", []string{elastic.DNSLogIndex}, true))
	t.Run("flows", f("flows", []string{elastic.FlowLogIndex}, true))
	t.Run("junk", f("junk", []string{}, false))
	t.Run("empty", f("", []string{}, false))
}

func TestQuery(t *testing.T) {
	f := func(alert v3.GlobalAlert, ok bool) func(*testing.T) {
		return func(t *testing.T) {
			g := NewWithT(t)

			_, err := Query(alert)
			if ok {
				g.Expect(err).ShouldNot(HaveOccurred())
			} else {
				g.Expect(err).Should(HaveOccurred())
			}
		}
	}

	t.Run("empty query", f(v3.GlobalAlert{
		Spec: libcalicov3.GlobalAlertSpec{
			DataSet: "audit",
			Query:   "",
		},
	}, true))
	t.Run("invalid query", f(v3.GlobalAlert{
		Spec: libcalicov3.GlobalAlertSpec{
			Query: "a AND ",
		},
	}, false))
	t.Run("audit", f(v3.GlobalAlert{
		Spec: libcalicov3.GlobalAlertSpec{
			DataSet: "audit",
			Query:   "apiVersion = abc",
		},
	}, true))
	t.Run("audit invalid", f(v3.GlobalAlert{
		Spec: libcalicov3.GlobalAlertSpec{
			DataSet: "audit",
			Query:   "abc = def",
		},
	}, false))
	t.Run("dns", f(v3.GlobalAlert{
		Spec: libcalicov3.GlobalAlertSpec{
			DataSet: "dns",
			Query:   "qtype = A",
		},
	}, true))
	t.Run("dns invalid", f(v3.GlobalAlert{
		Spec: libcalicov3.GlobalAlertSpec{
			DataSet: "dns",
			Query:   "abc = def",
		},
	}, false))
	t.Run("flows", f(v3.GlobalAlert{
		Spec: libcalicov3.GlobalAlertSpec{
			DataSet: "flows",
			Query:   "source_name = foo",
		},
	}, true))
	t.Run("flows invalid", f(v3.GlobalAlert{
		Spec: libcalicov3.GlobalAlertSpec{
			DataSet: "flows",
			Query:   "abc = def",
		},
	}, false))
	t.Run("invalid dataset", f(v3.GlobalAlert{
		Spec: libcalicov3.GlobalAlertSpec{
			DataSet: "invalid",
			Query:   "",
		},
	}, false))
}

func TestTermQueryAggs(t *testing.T) {
	f := func(aggs []string, baseAgg *QueryAgg) func(*testing.T) {
		return func(t *testing.T) {
			g := NewWithT(t)
			res := TermQueryAggs(aggs, baseAgg)

			for idx := range aggs {
				g.Expect(res.Field).Should(Equal(aggs[idx]))
				res = res.Child
			}

			g.Expect(res).Should(Equal(baseAgg))
		}
	}

	t.Run("empty", f(nil, nil))
	t.Run("one", f([]string{"one"}, nil))
	t.Run("two", f([]string{"one", "two"}, nil))
	t.Run("three", f([]string{"one", "two", "three"}, nil))
	t.Run("base", f([]string{"one", "two", "three"}, &QueryAgg{"a", "b", nil}))
	t.Run("json", func(t *testing.T) {
		g := NewWithT(t)

		agg := QueryAgg{
			Field:       "abc",
			Aggregation: "foo",
			Child: &QueryAgg{
				Field:       "def",
				Aggregation: "bar",
			},
		}

		b, err := json.Marshal(&agg)
		g.Expect(err).ShouldNot(HaveOccurred())

		g.Expect(b).Should(MatchJSON(`{"abc":{"foo":{"field":"abc"},"aggs":{"def":{"bar":{"field":"def"}}}}}`))
	})
}

func TestMetricQueryAggs(t *testing.T) {
	field := "test"
	f := func(metric string, agg, expected *QueryAgg) func(*testing.T) {
		return func(t *testing.T) {
			g := NewWithT(t)

			g.Expect(MetricQueryAggs(field, metric, agg)).Should(Equal(expected))
		}
	}

	t.Run("empty", f("", nil, nil))
	t.Run("count", f("count", nil, nil))
	t.Run("min", f("min", nil, &QueryAgg{field, "min", nil}))
	t.Run("max", f("max", nil, &QueryAgg{field, "max", nil}))
	t.Run("sum", f("sum", nil, &QueryAgg{field, "sum", nil}))
	t.Run("avg", f("avg", nil, &QueryAgg{field, "avg", nil}))
	t.Run("junk", f("junk", nil, &QueryAgg{field, "junk", nil}))
	chainedAgg := &QueryAgg{"foo", "bar", nil}
	t.Run("chained", f("chained", chainedAgg, &QueryAgg{field, "chained", chainedAgg}))
}

func TestGenerateDescriptionFunction(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		g := NewWithT(t)

		fn := GenerateDescriptionFunction("foo ${bar} ${bar.baz.foo} abc")
		fmt.Printf(fn)

		g.Expect(fn).ShouldNot(Equal(""))
		// There is not much to test here since it is Java code. This is tested on a live Elastic elsewhere.
		g.Expect(fn).Should(ContainSubstring("resolve("))
	})
	t.Run("broken", func(t *testing.T) {
		g := NewWithT(t)

		fn := GenerateDescriptionFunction("foo ${bar} ${bar.baz.foo")
		fmt.Printf(fn)

		g.Expect(fn).Should(ContainSubstring(`${bar.baz.foo"`))
	})
}

func TestActionTransform(t *testing.T) {
	f := func(description string, severity int) func(*testing.T) {
		return func(t *testing.T) {

			g := NewWithT(t)

			transform := ActionTransform(v3.GlobalAlert{
				Spec: libcalicov3.GlobalAlertSpec{
					Description: description,
					Severity:    severity,
				},
			})

			fmt.Println(transform.Source)
			g.Expect(transform.Params["description"]).Should(Equal(description))
			g.Expect(transform.Params["severity"]).Should(Equal(severity))
			g.Expect(transform.Params["type"]).Should(Equal(AlertEventType))
		}
	}
	t.Run("funny description", f(`description ${foo} "bar"`, 100))
	t.Run("low severity", f("test", 5))
}
