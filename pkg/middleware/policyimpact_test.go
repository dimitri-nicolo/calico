package middleware_test

import (
	"bytes"
	"encoding/json"
	"io/ioutil"

	"github.com/tigera/es-proxy/pkg/middleware"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"net/http"
)

var _ = Describe("The PolicyImpactRequestProcessor properly modifies requests", func() {

	DescribeTable("extracts parameters",
		func(originalReqBody string, expectedModifiedBody string, expectedParams *middleware.PolicyImpactParams) {

			//build the request
			b := []byte(originalReqBody)
			req, err := http.NewRequest(http.MethodPost, "", bytes.NewBuffer(b))
			Expect(err).NotTo(HaveOccurred())

			//pass it to the function under test
			p, err := middleware.PolicyImpactRequestProcessor(req)
			Expect(err).NotTo(HaveOccurred())

			//read the request body after the call
			ob, err := ioutil.ReadAll(req.Body)
			Expect(err).NotTo(HaveOccurred())
			obs := string(ob)

			//check the final request body
			Expect(obs).To(MatchJSON(expectedModifiedBody))

			//check expected params
			if expectedParams == nil {
				Expect(p).To(BeNil())
			} else {
				pjson := pipParamsToJson(*p)
				ejson := pipParamsToJson(*expectedParams)
				Expect(pjson).To(MatchJSON(ejson))
			}

		},
		Entry("Not a pip request", nullESQuery, nullESQuery, nil),
		Entry("Pip request", pipPostRequestBody, nullESQuery, jsonToPipParams(pipParams)),
	)

})

func pipParamsToJson(pipParams middleware.PolicyImpactParams) string {
	RegisterFailHandler(Fail)
	b, err := json.Marshal(pipParams)
	Expect(err).ToNot(HaveOccurred())
	return string(b)
}

func jsonToPipParams(j string) *middleware.PolicyImpactParams {
	RegisterFailHandler(Fail)
	var data middleware.PolicyImpactParams
	err := json.Unmarshal([]byte(j), &data)
	Expect(err).ToNot(HaveOccurred())
	return &data
}

var nullESQuery = `{"query":{"bool":{"must":[{"match_all":{}}],"must_not":[],"should":[]}},"from":0,"size":10,"sort":[],"aggs":{}}`

var pipPostRequestBody = `{"query":{"bool":{"must":[{"match_all":{}}],"must_not":[],"should":[]}},"from":0,"size":10,"sort":[],"aggs":{},
"policyActions":[{"policy":{
			"metadata":{
				"name":"p-name",
				"generateName":"p-gen-name",
				"namespace":"p-name-space",
				"selfLink":"p-self-link",
				"resourceVersion":"p-res-ver"
			},
			"spec":{
				"tier":"default",
				"order":1,
				"selector":"a|bogus|selector|string"
			}
		}
		,"action":"create"}]

}`

var pipParams = `{"policyActions":[{"policy":{
	"metadata":{
		"name":"p-name",
		"generateName":"p-gen-name",
		"namespace":"p-name-space",
		"selfLink":"p-self-link",
		"resourceVersion":"p-res-ver"
	},
	"spec":{
		"tier":"default",
		"order":1,
		"selector":"a|bogus|selector|string"
	}
}
,"action":"create"}]}`
