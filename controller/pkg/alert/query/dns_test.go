// Copyright (c) 2019 Tigera Inc. All rights reserved.

package query

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/projectcalico/libcalico-go/lib/validator/v3/query"

	. "github.com/onsi/gomega"
)

func TestDNSConverter(t *testing.T) {
	t.Run("type", genDNSTest("type", `{"term":{"type":{"value":"value"}}}`))

	for _, key := range []string{
		"servers.name",
		"servers.name_aggr",
		"servers.namespace",
		"servers.ip",
		"rrsets.name",
		"rrsets.type",
		"rrsets.class",
		"rrsets.rdata",
	} {
		t.Run(key, genDNSTest(key, fmt.Sprintf(`
		{
		  "nested": {
		    "path": %q,
		    "query": { "term": { %q: { "value": "value" } } }
		  }
		}
	`, key[:strings.Index(key, ".")], key)))
	}

	t.Run("servers.labels", genDNSTest("servers.labels.a.b.c", `
		{
		  "nested": {
		    "path": "servers",
			"query": { "term": { "servers.labels.a.b.c": { "value": "value" } } }
          }
		}
	`))

	t.Run("client_labels", genDNSTest("client_labels.a.b.c", `
		{ "term": { "client_labels.a.b.c": { "value": "value" } } }
	`))
}

func genDNSTest(key, expected string) func(*testing.T) {
	return func(t *testing.T) {
		g := NewWithT(t)

		atom := &query.Atom{key, query.CmpEqual, "value"}
		c := NewDNSConverter().(*converter)

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
