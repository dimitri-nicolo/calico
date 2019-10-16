// Copyright (c) 2019 Tigera Inc. All rights reserved.

package query

import (
	"encoding/json"
	"testing"

	"github.com/projectcalico/libcalico-go/lib/validator/v3/query"

	. "github.com/onsi/gomega"
)

func TestFlowsConverter(t *testing.T) {
	t.Run("action", genFlowsTest("action", `{"term":{"action":{"value":"value"}}}`))
	t.Run("dest_labels.labels", genFlowsTest("dest_labels.labels", `
		{
		  "nested": {
		    "path": "dest_labels",
		    "query": { "term": { "dest_labels.labels": { "value": "value" } } }
		  }
		}
	`))
	t.Run("source_labels.labels", genFlowsTest("source_labels.labels", `
		{
		  "nested": {
		    "path": "source_labels",
		    "query": { "term": { "source_labels.labels": { "value": "value" } } }
		  }
		}
	`))
	t.Run("policies.all_policies", genFlowsTest("policies.all_policies", `
		{
		  "nested": {
		    "path": "policies",
		    "query": { "term": { "policies.all_policies": { "value": "value" } } }
		  }
		}
	`))
}

func genFlowsTest(key, expected string) func(*testing.T) {
	return func(t *testing.T) {
		g := NewWithT(t)

		atom := &query.Atom{key, query.CmpEqual, "value"}
		c := NewFlowsConverter().(*converter)

		doc := c.atomToElastic(atom)
		actual, err := json.Marshal(&doc)
		g.Expect(err).ShouldNot(HaveOccurred())

		var e interface{}
		err = json.Unmarshal([]byte(expected), &e)
		g.Expect(err).ShouldNot(HaveOccurred())
		normalized, err := json.Marshal(&e)
		g.Expect(err).ShouldNot(HaveOccurred())

		g.Expect(actual).Should(Equal(normalized))
	}
}
