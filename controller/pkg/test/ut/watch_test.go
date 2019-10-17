// Copyright (c) 2019 Tigera Inc. All rights reserved.

package ut

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	oElastic "github.com/olivere/elastic/v7"

	. "github.com/onsi/gomega"
	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/yalp/jsonpath"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	alertElastic "github.com/tigera/intrusion-detection/controller/pkg/alert/elastic"
	"github.com/tigera/intrusion-detection/controller/pkg/elastic"
)

var Debug = os.Getenv("TEST_WATCH_DEBUG") == "yes"

func clearWatches(g *WithT, ctx context.Context) {
	metas, err := uut.ListWatches(ctx)
	g.Expect(err).ShouldNot(HaveOccurred())
	for _, m := range metas {
		err := uut.DeleteWatch(ctx, m)
		g.Expect(err).ShouldNot(HaveOccurred())
	}
}

func TestXPackWatch(t *testing.T) {
	g := NewWithT(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	defer clearWatches(g, ctx)

	name := "test"
	err := uut.PutWatch(ctx, name, &elastic.PutWatchBody{
		Trigger: elastic.Trigger{
			Schedule: elastic.Schedule{
				Interval: &elastic.Interval{
					Duration: time.Minute,
				},
			},
		},
		Input: &elastic.Input{
			Simple: &elastic.Simple{"foo": "bar"},
		},
		Actions: nil,
	})
	g.Expect(err).ShouldNot(HaveOccurred())

	m, err := uut.ListWatches(ctx)
	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(m).Should(HaveLen(1))
	g.Expect(m[0].Name).Should(Equal(name))

	err = uut.DeleteWatch(ctx, m[0])
	g.Expect(err).ShouldNot(HaveOccurred())

	m, err = uut.ListWatches(ctx)
	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(m).Should(HaveLen(0))
}

func TestWatch(t *testing.T) {
	g := NewWithT(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for _, dataSet := range []string{"audit", "dns", "flows"} {
		// Install the mapping
		template := mustGetString(fmt.Sprintf("test_files/%s_template.json", dataSet))
		_, err := elasticClient.IndexPutTemplate(fmt.Sprintf("%s_logs", dataSet)).BodyString(template).Do(ctx)
		g.Expect(err).ToNot(HaveOccurred())

		var index string
		switch dataSet {
		case "audit":
			index = fmt.Sprintf("tigera_secure_ee_audit_kube.cluster.%s", strings.ToLower(t.Name()))
			oldIndex := elastic.AuditIndex
			elastic.AuditIndex = index
			defer func() {
				_, err := elasticClient.DeleteIndex(elastic.AuditIndex).Do(ctx)
				elastic.AuditIndex = oldIndex
				g.Expect(err).ShouldNot(HaveOccurred())
			}()
		case "dns":
			index = fmt.Sprintf("tigera_secure_ee_dns.cluster.%s", strings.ToLower(t.Name()))
			oldIndex := elastic.DNSLogIndex
			elastic.DNSLogIndex = index
			defer func() {
				_, err := elasticClient.DeleteIndex(elastic.DNSLogIndex).Do(ctx)
				elastic.DNSLogIndex = oldIndex
				g.Expect(err).ShouldNot(HaveOccurred())
			}()
		case "flows":
			index = fmt.Sprintf("tigera_secure_ee_flows.cluster.%s", strings.ToLower(t.Name()))
			oldIndex := elastic.FlowLogIndex
			elastic.FlowLogIndex = index
			defer func() {
				_, err := elasticClient.DeleteIndex(elastic.FlowLogIndex).Do(ctx)
				elastic.FlowLogIndex = oldIndex
				g.Expect(err).ShouldNot(HaveOccurred())
			}()
		default:
			panic("bad test")
		}

		// Index some logs
		i := elasticClient.Index().Index(index)

		var logs []interface{}
		b, err := ioutil.ReadFile(fmt.Sprintf("test_files/watch_%s_data.json", dataSet))
		err = json.Unmarshal(b, &logs)

		var logIds []string
		for _, l := range logs {
			// Force the time to be inside the interval
			//l.StartTime.Time = time.Now().Add(-time.Minute * 2)
			resp, err := i.BodyJson(l).Do(ctx)
			g.Expect(err).ToNot(HaveOccurred())
			logIds = append(logIds, resp.Id)
		}

		// Refresh the index
		_, err = elasticClient.Refresh(index).Do(ctx)
		g.Expect(err).ShouldNot(HaveOccurred())
	}

	f := func(alert v3.GlobalAlert, numExpected int) func(*testing.T) {
		return func(t *testing.T) {
			g := NewWithT(t)

			nameMd5 := md5.Sum([]byte(t.Name()))
			encodedName := hex.EncodeToString(nameMd5[:])

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Set up a temporary index for events
			oldEventIndex := elastic.EventIndex
			elastic.EventIndex = fmt.Sprintf(elastic.EventIndexPattern, encodedName)
			defer func() {
				_, err := elasticClient.DeleteIndex(elastic.EventIndex).Do(ctx)
				elastic.EventIndex = oldEventIndex
				g.Expect(err).ToNot(HaveOccurred())
			}()

			// Install the event mapping
			_, err := elasticClient.CreateIndex(elastic.EventIndex).Body(elastic.EventMapping).Do(ctx)
			g.Expect(err).ToNot(HaveOccurred())

			body, err := alertElastic.Watch(alert)
			g.Expect(err).ShouldNot(HaveOccurred())

			if Debug {
				b, _ := json.MarshalIndent(map[string]interface{}{"watch": body}, "", "  ")
				fmt.Println(string(b))
			}

			res, err := uut.TestWatch(ctx, body)
			g.Expect(err).ShouldNot(HaveOccurred())

			if Debug {
				b, _ := json.MarshalIndent(res, "", "  ")
				fmt.Println(string(b))
			}

			g.Expect(res.Messages).Should(ConsistOf())
			g.Expect(res.State).Should(Equal("executed"))
			g.Expect(res.Status.ExecutionState).Should(Equal("executed"))
			g.Expect(res.Status.Actions).Should(HaveKey(alertElastic.IndexActionName))
			g.Expect(res.Status.Actions[alertElastic.IndexActionName]["last_execution"].(map[string]interface{})["successful"]).Should(BeTrue())

			response, err := jsonpath.Read(res.Result, "$.actions[0].index.response")
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(response).Should(HaveLen(numExpected))

			_, err = elasticClient.Refresh(elastic.EventIndex).Do(ctx)
			g.Expect(err).ShouldNot(HaveOccurred())

			s, err := elasticClient.Search(elastic.EventIndex).Do(ctx)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(s.Hits.TotalHits.Value).Should(BeNumerically("==", numExpected))
		}
	}

	lookback := v1.Duration{time.Hour * 24 * 365 * 10}
	t.Run("dns", f(v3.GlobalAlert{Spec: v3.GlobalAlertSpec{
		Description: "dns",
		Severity:    100,
		Lookback:    lookback,
		DataSet:     "dns",
	}}, 200))
	t.Run("dns.count.agg0", f(v3.GlobalAlert{
		Spec: v3.GlobalAlertSpec{
			Description: "${count} distinct responses",
			Severity:    100,
			DataSet:     "dns",
			Lookback:    lookback,
			Metric:      "count",
			Condition:   "gte",
			Threshold:   0,
		},
	}, 1))
	t.Run("dns.count.agg1", f(v3.GlobalAlert{
		Spec: v3.GlobalAlertSpec{
			Description: "query for ${qname} yields ${rrsets.rdata}",
			Severity:    100,
			DataSet:     "dns",
			Lookback:    lookback,
			AggregateBy: []string{"qname"},
			Metric:      "count",
			Condition:   "gte",
			Threshold:   0,
		},
	}, 10))
	t.Run("dns.count.agg2", f(v3.GlobalAlert{
		Spec: v3.GlobalAlertSpec{
			Description: "query for ${qname} yields ${rrsets.rdata}",
			Severity:    100,
			DataSet:     "dns",
			Lookback:    lookback,
			AggregateBy: []string{"qname", "qtype"},
			Metric:      "count",
			Condition:   "gte",
			Threshold:   0,
		},
	}, 20))
	t.Run("dns.sum.agg0", f(v3.GlobalAlert{
		Spec: v3.GlobalAlertSpec{
			Description: "query for ${qname} yields ${rrsets.rdata}",
			Severity:    100,
			DataSet:     "dns",
			Lookback:    lookback,
			Field:       "count",
			Metric:      "sum",
			Condition:   "gte",
			Threshold:   0,
		},
	}, 1))
	t.Run("dns.sum.agg1", f(v3.GlobalAlert{
		Spec: v3.GlobalAlertSpec{
			Description: "query for ${qname} yields ${rrsets.rdata}",
			Severity:    100,
			DataSet:     "dns",
			Lookback:    lookback,
			AggregateBy: []string{"qname"},
			Field:       "count",
			Metric:      "sum",
			Condition:   "gte",
			Threshold:   0,
		},
	}, 10))
	t.Run("dns.sum.agg2", f(v3.GlobalAlert{
		Spec: v3.GlobalAlertSpec{
			Description: "query for ${qname} yields ${rrsets.rdata}",
			Severity:    100,
			DataSet:     "dns",
			Lookback:    lookback,
			AggregateBy: []string{"qname", "qtype"},
			Field:       "count",
			Metric:      "sum",
			Condition:   "gte",
			Threshold:   0,
		},
	}, 20))
	t.Run("dns.query[0]", f(v3.GlobalAlert{
		Spec: v3.GlobalAlertSpec{
			Description: "query",
			Severity:    100,
			Lookback:    lookback,
			DataSet:     "dns",
			Query:       "qtype = A",
		},
	}, 91))
	t.Run("dns.query[1]", f(v3.GlobalAlert{
		Spec: v3.GlobalAlertSpec{
			Description: "query",
			Severity:    100,
			Lookback:    lookback,
			DataSet:     "dns",
			Query:       "qtype = A AND rcode = NoError",
		},
	}, 10))
	t.Run("dns.query[2]", f(v3.GlobalAlert{
		Spec: v3.GlobalAlertSpec{
			Description: "query",
			Severity:    100,
			Lookback:    lookback,
			DataSet:     "dns",
			Query:       "qtype = A AND rcode != NoError",
		},
	}, 81))
	t.Run("dns.query[2].agg.sum", f(v3.GlobalAlert{
		Spec: v3.GlobalAlertSpec{
			Description: "query",
			Severity:    100,
			Lookback:    lookback,
			DataSet:     "dns",
			Query:       "qtype = A AND rcode != NoError",
			AggregateBy: []string{"qname"},
			Field:       "count",
			Metric:      "sum",
			Condition:   "gt",
			Threshold:   0,
		},
	}, 10))
	t.Run("flows", f(v3.GlobalAlert{
		Spec: v3.GlobalAlertSpec{
			Description: "flows",
			Severity:    100,
			Lookback:    lookback,
			DataSet:     "flows",
		},
	}, 200))
	t.Run("audit", f(v3.GlobalAlert{
		Spec: v3.GlobalAlertSpec{
			Description: "audit",
			Severity:    100,
			Lookback:    lookback,
			DataSet:     "audit",
		},
	}, 200))
}

func TestActionTransform(t *testing.T) {
	g := NewWithT(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	transform := alertElastic.ActionTransform(v3.GlobalAlert{
		Spec: v3.GlobalAlertSpec{
			Description: "foo ${foo}",
			Severity:    100,
		},
	})

	body := elastic.PutWatchBody{
		Trigger: elastic.Trigger{
			Schedule: elastic.Schedule{
				Interval: &elastic.Interval{time.Second},
			},
		},
		Input: &elastic.Input{
			Simple: &elastic.Simple{
				"foo": "bar",
			},
		},
		Transform: &elastic.Transform{
			Script: elastic.Script{
				Source: "[ ctx.payload ]",
			},
		},
		Actions: map[string]elastic.Action{
			"index": {
				Transform: transform,
				Index: &elastic.IndexAction{
					Index: "foo",
				},
			},
		},
	}

	res, err := elasticClient.XPackWatchExecute().BodyJson(map[string]interface{}{
		"watch": body,
	}).Do(ctx)
	g.Expect(err).ShouldNot(HaveOccurred())

	b, _ := json.MarshalIndent(res.WatchRecord, "", "  ")
	fmt.Println(string(b))

	g.Expect(res.WatchRecord.State).Should(Equal("executed"))
	g.Expect(jsonpath.Read(res.WatchRecord.Status.Actions["index"], "$.last_execution.successful")).Should(Equal(true))
	g.Expect(jsonpath.Read(res.WatchRecord.Result, "$.actions[0].transform.status")).Should(Equal("success"))
	g.Expect(jsonpath.Read(res.WatchRecord.Result, "$.actions[0].transform.payload._doc[0].description")).Should(Equal("foo bar"))
}

func TestGenerateDescriptionFunction(t *testing.T) {
	f := func(description, data, expected string) func(*testing.T) {
		return func(t *testing.T) {
			g := NewWithT(t)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			res, err := elasticClient.PerformRequest(ctx, oElastic.PerformRequestOptions{
				Method: "POST",
				Path:   "/_scripts/painless/_execute",
				Body: map[string]elastic.Script{
					"script": {
						Language: "painless",
						Source: alertElastic.ResolveCode + alertElastic.GenerateDescriptionFunction(description) +
							`description(params.data)`,
						Params: map[string]interface{}{
							"data": json.RawMessage(data),
						},
					},
				},
			})
			g.Expect(err).ShouldNot(HaveOccurred())

			var result map[string]string
			err = json.Unmarshal(res.Body, &result)
			g.Expect(err).ShouldNot(HaveOccurred())

			g.Expect(result["result"]).Should(Equal(expected))
		}
	}

	t.Run("blank", f("", `{}`, ""))
	t.Run("no variables", f("abc", `{}`, "abc"))
	t.Run("broken variable", f("abc ${def", `{}`, "abc ${def"))
	t.Run("variable missing braces", f("abc $def", `{"def": "123"}`, "abc $def"))
	t.Run("missing variable", f("abc ${123} def", `{"def": "123"}`, "abc null def"))
	t.Run("variable at end", f("abc ${def}", `{"def": "123"}`, "abc 123"))
	t.Run("variable at beginning", f("${def} abc", `{"def": "123"}`, "123 abc"))
	t.Run("variable in middle", f("abc ${def} ghi", `{"def": "123"}`, "abc 123 ghi"))
	t.Run("nested variable", f("${abc.def}", `{"abc":{"def": "123"}}`, "123"))
	t.Run("nested array variable", f("${abc.def}", `{"abc":[{"def": "123"}]}`, "[123]"))
	t.Run("nested array variable 2 deep", f("${abc.def.ghi}", `{"abc":[{"def": {"ghi":"123"}}]}`, "[123]"))
	t.Run("nested 2 deep array variable", f("${abc.def.ghi}", `{"abc":{"def": [{"ghi":"123"}]}}`, "[123]"))
	t.Run("dots in variable name", f("${abc.def.ghi}", `{"abc":{"def.ghi": "123"}}`, "123"))
	t.Run("overrun variable name", f("${abc.def.ghi}", `{"abc":{"def": "123"}}`, "null"))
	t.Run("trailing dot on variable name", f("${abc.def.}", `{"abc":{"def": "123"}}`, "null"))
	t.Run("trailing dots on variable name", f("${abc.def..}", `{"abc":{"def": "123"}}`, "null"))
}
