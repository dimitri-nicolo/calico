package parser

import (
	_ "embed"
	"encoding/json"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//go:embed testdata/simple_testdata.conf
var simple_testdata string

//go:embed testdata/simple_testdata_parsed.json
var simple_parsed []byte

var _ = Describe("Waf Ruleset Parser Test", func() {

	It("Parse simple ruleset testdata File", func() {
		rules, err := Parse(simple_testdata)
		Expect(err).To(BeNil())

		Expect(rules).To(Not(BeNil()))

		jsonData, err := json.MarshalIndent(rules, "", "    ")
		Expect(err).To(BeNil())

		Expect(jsonData).To(Equal(simple_parsed))
	})
})
