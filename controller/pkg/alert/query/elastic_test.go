// Copyright 2019 Tigera Inc. All rights reserved.

package query

import (
	"encoding/json"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/projectcalico/libcalico-go/lib/validator/v3/query"
)

func TestQueryToElastic(t *testing.T) {
	// empty
	t.Run("empty", genElasticTest("", `{"match_all": {}}`))

	// Atoms and Comparators
	t.Run("equal", genElasticTest("a=b", `{"term":{"a":{"value":"b"}}}`))
	t.Run("not equal", genElasticTest("a!=b", `{"bool":{"must_not":{"term":{"a":{"value":"b"}}}}}`))
	t.Run("gt", genElasticTest("a>b", `{"range":{"a":{"gt":"b"}}}`))
	t.Run("gte", genElasticTest("a>=b", `{"range":{"a":{"gte":"b"}}}`))
	t.Run("lt", genElasticTest("a<b", `{"range":{"a":{"lt":"b"}}}`))
	t.Run("lte", genElasticTest("a<=b", `{"range":{"a":{"lte":"b"}}}`))

	// basic operations
	t.Run("AND", genElasticTest("a=b AND c=d", `
		{
		  "bool": {
			"must": [
			  { "term": { "a": { "value": "b" } } },
			  { "term": { "c": { "value": "d" } } }
			]
		  }
		}
	`))
	t.Run("OR", genElasticTest("a=b OR c=d", `
		{
		  "bool": {
		    "should": [
		      { "term": { "a": { "value": "b" } } },
		      { "term": { "c": { "value": "d" } } }
		    ]
		  }
		}
	`))
	t.Run("NOT", genElasticTest("NOT a=b", `
		{
		  "bool": {
		    "must_not": { "term": { "a": { "value": "b" } } }
		  }
		}
	`))

	// brackets
	t.Run("(x AND y) OR (a AND b)", genElasticTest("(a=b AND b=c) OR (d=e AND e=f)", `
		{
		  "bool": {
		    "should": [
		      { "bool": {
		        "must": [
		          { "term": { "a": { "value": "b" } } },
		          { "term": { "b": { "value": "c" } } }
		        ]
		      } },
		      { "bool": {
		        "must": [
		          { "term": { "d": { "value": "e" } } },
		          { "term": { "e": { "value": "f" } } }
		        ]
		      } }
		    ]
		  }
		}
	`))
	t.Run("(x OR y) AND (a OR b)", genElasticTest("(a=b OR b=c) AND (d=e OR e=f)", `
		{
		  "bool": {
		    "must": [
		      { "bool": {
		        "should": [
		          { "term": { "a": { "value": "b" } } },
		          { "term": { "b": { "value": "c" } } }
		        ]
		      } },
		      { "bool": {
		        "should": [
		          { "term": { "d": { "value": "e" } } },
		          { "term": { "e": { "value": "f" } } }
		        ]
		      } }
		    ]
		  }
		}
	`))

	// complex NOT operations
	t.Run("NOT (x AND y)", genElasticTest("NOT (a=b AND b=c)", `
		{
		  "bool": {
		    "must_not": {
		      "bool": {
		        "must": [
		          { "term": { "a": { "value": "b" } } },
		          { "term": { "b": { "value": "c" } } }
		        ]
		      }
		    }
		  }
		}
	`))
	t.Run("NOT (x OR y)", genElasticTest("NOT (a=b OR b=c)", `
		{
		  "bool": {
		    "must_not": {
		      "bool": {
		        "should": [
		          { "term": { "a": { "value": "b" } } },
		          { "term": { "b": { "value": "c" } } }
		        ]
		      }
		    }
		  }
		}
	`))
	t.Run("x AND NOT y", genElasticTest("a=b AND NOT b=c", `
		{
		  "bool": {
		    "must": [
		      { "term": { "a": { "value": "b" } } },
		      { "bool": { "must_not": { "term": { "b": { "value": "c" } } } }
		      }
		    ]
		  }
		}
	`))
	t.Run("NOT x AND y", genElasticTest("NOT a=b AND b=c", `
		{
		  "bool": {
		    "must": [
		      { "bool": { "must_not": { "term": { "a": { "value": "b" } } } } },
		      { "term": { "b": { "value": "c" } } }
		    ]
		  }
		}
	`))
	t.Run("x OR NOT y", genElasticTest("a=b OR NOT b=c", `
		{
		  "bool": {
		    "should": [
		      { "term": { "a": { "value": "b" } } },
		      { "bool": { "must_not": { "term": { "b": { "value": "c" } } } }
		      }
		    ]
		  }
		}
	`))
	t.Run("NOT x OR y", genElasticTest("NOT a=b OR b=c", `
		{
		  "bool": {
		    "should": [
		      { "bool": { "must_not": { "term": { "a": { "value": "b" } } } } },
		      { "term": { "b": { "value": "c" } } }
		    ]
		  }
		}
	`))

}

func genElasticTest(input, expected string) func(t *testing.T) {
	return func(t *testing.T) {
		g := NewWithT(t)

		q, err := query.ParseQuery(input)
		g.Expect(err).ShouldNot(HaveOccurred())

		c := &converter{basicAtomToElastic}

		doc := c.Convert(q)
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
