// Copyright (c) 2019 Tigera Inc. All rights reserved.

package query

import (
	"encoding/json"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/projectcalico/libcalico-go/lib/validator/v3/query"
)

func TestAuditConverter(t *testing.T) {
	g := NewWithT(t)

	atom := &query.Atom{"kind", query.CmpEqual, "v"}
	c := NewAuditConverter().(*converter)

	doc := c.atomToElastic(atom)
	actual, err := json.Marshal(&doc)
	g.Expect(err).ShouldNot(HaveOccurred())

	g.Expect(actual).Should(Equal([]byte(`{"term":{"kind":{"value":"v"}}}`)))
}
