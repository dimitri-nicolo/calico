// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package collector

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/felix/config"
)

var _ = Describe("L7 logs aggregation tests", func() {
	Context("While translating config params into L7 aggregation kind", func() {
		It("Should return the default aggregation kind if nothing is set", func() {
			cfg := &config.Config{}
			aggKind := getL7AggregationKindFromConfigParams(cfg)
			expectedKind := DefaultL7AggregationKind()
			// Path parts will be 0 because nothing has been set for this test
			expectedKind.NumURLPathParts = 0
			expectedKind.URLCharLimit = 0
			Expect(aggKind).To(Equal(expectedKind))
		})

		It("Should accurately modify http header levels", func() {
			cfg := &config.Config{
				L7LogsFileAggregationHTTPHeaderInfo: "IncludeL7HTTPHeaderInfo",
			}
			aggKind := getL7AggregationKindFromConfigParams(cfg)
			expectedKind := DefaultL7AggregationKind()
			expectedKind.HTTPHeader = L7HTTPHeaderInfo
			// Path parts will be 0 because nothing has been set for this test
			expectedKind.NumURLPathParts = 0
			expectedKind.URLCharLimit = 0
			Expect(aggKind).To(Equal(expectedKind))
		})

		It("Should accurately modify method levels", func() {
			cfg := &config.Config{
				L7LogsFileAggregationHTTPMethod: "ExcludeL7HTTPMethod",
			}
			aggKind := getL7AggregationKindFromConfigParams(cfg)
			expectedKind := DefaultL7AggregationKind()
			expectedKind.HTTPMethod = L7HTTPMethodNone
			// Path parts will be 0 because nothing has been set for this test
			expectedKind.NumURLPathParts = 0
			expectedKind.URLCharLimit = 0
			Expect(aggKind).To(Equal(expectedKind))
		})

		It("Should accurately modify service levels", func() {
			cfg := &config.Config{
				L7LogsFileAggregationServiceInfo: "ExcludeL7ServiceInfo",
			}
			aggKind := getL7AggregationKindFromConfigParams(cfg)
			expectedKind := DefaultL7AggregationKind()
			expectedKind.Service = L7ServiceInfoNone
			// Path parts will be 0 because nothing has been set for this test
			expectedKind.NumURLPathParts = 0
			expectedKind.URLCharLimit = 0
			Expect(aggKind).To(Equal(expectedKind))
		})

		It("Should accurately modify destination levels", func() {
			cfg := &config.Config{
				L7LogsFileAggregationDestinationInfo: "ExcludeL7DestinationInfo",
			}
			aggKind := getL7AggregationKindFromConfigParams(cfg)
			expectedKind := DefaultL7AggregationKind()
			expectedKind.Destination = L7DestinationInfoNone
			// Path parts will be 0 because nothing has been set for this test
			expectedKind.NumURLPathParts = 0
			expectedKind.URLCharLimit = 0
			Expect(aggKind).To(Equal(expectedKind))
		})

		It("Should accurately modify source levels", func() {
			cfg := &config.Config{
				L7LogsFileAggregationSourceInfo: "ExcludeL7SourceInfo",
			}
			aggKind := getL7AggregationKindFromConfigParams(cfg)
			expectedKind := DefaultL7AggregationKind()
			expectedKind.Source = L7SourceInfoNone
			// Path parts will be 0 because nothing has been set for this test
			expectedKind.NumURLPathParts = 0
			expectedKind.URLCharLimit = 0
			Expect(aggKind).To(Equal(expectedKind))
		})

		It("Should accurately modify response code levels", func() {
			cfg := &config.Config{
				L7LogsFileAggregationResponseCode: "ExcludeL7ResponseCode",
			}
			aggKind := getL7AggregationKindFromConfigParams(cfg)
			expectedKind := DefaultL7AggregationKind()
			expectedKind.ResponseCode = L7ResponseCodeNone
			// Path parts will be 0 because nothing has been set for this test
			expectedKind.NumURLPathParts = 0
			expectedKind.URLCharLimit = 0
			Expect(aggKind).To(Equal(expectedKind))
		})

		It("Should accurately modify URL levels", func() {
			cfg := &config.Config{
				L7LogsFileAggregationTrimURL: "ExcludeL7URL",
			}
			aggKind := getL7AggregationKindFromConfigParams(cfg)
			expectedKind := DefaultL7AggregationKind()
			expectedKind.TrimURL = L7URLNone
			// Path parts will be 0 because nothing has been set for this test
			expectedKind.NumURLPathParts = 0
			expectedKind.URLCharLimit = 0
			Expect(aggKind).To(Equal(expectedKind))

			cfg = &config.Config{
				L7LogsFileAggregationTrimURL: "TrimURLQueryAndPath",
			}
			aggKind = getL7AggregationKindFromConfigParams(cfg)
			expectedKind = DefaultL7AggregationKind()
			expectedKind.TrimURL = L7BaseURL
			// Path parts will be 0 because nothing has been set for this test
			expectedKind.NumURLPathParts = 0
			expectedKind.URLCharLimit = 0
			Expect(aggKind).To(Equal(expectedKind))

			cfg = &config.Config{
				L7LogsFileAggregationTrimURL: "IncludeL7FullURL",
			}
			aggKind = getL7AggregationKindFromConfigParams(cfg)
			expectedKind = DefaultL7AggregationKind()
			expectedKind.TrimURL = L7FullURL
			// Path parts will be 0 because nothing has been set for this test
			expectedKind.NumURLPathParts = 0
			expectedKind.URLCharLimit = 0
			Expect(aggKind).To(Equal(expectedKind))
		})

		It("Should accurately modify URL path part setting", func() {
			cfg := &config.Config{
				L7LogsFileAggregationNumURLPath: 2,
			}
			aggKind := getL7AggregationKindFromConfigParams(cfg)
			expectedKind := DefaultL7AggregationKind()
			expectedKind.NumURLPathParts = 2
			expectedKind.URLCharLimit = 0
			Expect(aggKind).To(Equal(expectedKind))
		})

		It("Should accurately modify URL path part setting", func() {
			cfg := &config.Config{
				L7LogsFileAggregationURLCharLimit: 2,
			}
			aggKind := getL7AggregationKindFromConfigParams(cfg)
			expectedKind := DefaultL7AggregationKind()
			expectedKind.NumURLPathParts = 0
			expectedKind.URLCharLimit = 2
			Expect(aggKind).To(Equal(expectedKind))
		})
	})
})
