// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package collector

import (
	"fmt"
	"strconv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/felix/config"
)

var _ = Describe("L7 logs aggregation tests", func() {

	var la *l7LogAggregator

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

	Context("With per-node limit of 5", func() {

		BeforeEach(func() {
			la = NewL7LogAggregator().(*l7LogAggregator)
			la.PerNodeLimit(5)
		})

		It("Should only buffer 5 logs", func() {
			for i := 0; i < 15; i++ {
				src := [16]byte{127, 0, 0, byte(i)}
				dst := [16]byte{127, 0, 0, byte(5 + i)}
				err := la.FeedUpdate(
					L7Update{Tuple: MakeTuple(src, dst, i, i, i), Duration: i,
						DurationMax: i, BytesSent: i, BytesReceived: i,
						ResponseCode: "200", Path: fmt.Sprintf("/%s", strconv.Itoa(i)), Count: 1, Type: "tcp"})
				Expect(err).ShouldNot(HaveOccurred())
			}
			Expect(la.l7Store).Should(HaveLen(5))
			// la.Get() will return the 5 stored logs, plus an extra one to say
			// that there were 10 more updates that could not be fully logged.
			emitted := la.Get()
			Expect(emitted).To(HaveLen(6))
			// last log is of type unlogged and contains unlogged messages
			last := emitted[len(emitted)-1]
			Expect(last.Type).To(Equal(L7LogTypeUnLogged))
			Expect(last.Count).To(Equal(10))
		})

		It("Should buffer logs and overflow logs up to 5 logs", func() {
			// Create 3 full logs
			for i := 0; i < 3; i++ {
				src := [16]byte{127, 0, 0, byte(i)}
				dst := [16]byte{127, 0, 0, byte(5 + i)}
				err := la.FeedUpdate(
					L7Update{Tuple: MakeTuple(src, dst, i, i, i), Duration: i,
						DurationMax: i, BytesSent: i, BytesReceived: i,
						ResponseCode: "200", Path: fmt.Sprintf("/%s", strconv.Itoa(i)), Count: 1, Type: "tcp"})
				Expect(err).ShouldNot(HaveOccurred())
			}
			// Create 5 overflow logs
			for i := 0; i < 5; i++ {
				src := [16]byte{127, 0, 0, byte(i)}
				dst := [16]byte{127, 0, 0, byte(5 + i)}
				err := la.FeedUpdate(
					L7Update{Tuple: MakeTuple(src, dst, i, i, i), Count: 1})
				Expect(err).ShouldNot(HaveOccurred())
			}
			Expect(la.l7Store).Should(HaveLen(3))
			Expect(la.l7OverflowStore).Should(HaveLen(5))
			// la.Get() will return the 3 full logs, 2 overflow logs, and 1 extra log
			// to record that there were 3 more updates that could not be fully logged.
			emitted := la.Get()
			Expect(emitted).To(HaveLen(6))
			// last log is of the type unlogged and containers unlogged messages
			last := emitted[len(emitted)-1]
			Expect(last.Type).To(Equal(L7LogTypeUnLogged))
			Expect(last.Count).To(Equal(3))
			// 2 of the logs should be overflow logs and have no Type
			numOverflow := 0
			for _, log := range emitted {
				if log.Type == "" {
					numOverflow++
				}
			}
			Expect(numOverflow).To(Equal(2))
		})

		It("Should buffer 5 logs with full logs taking priority over overflow logs", func() {
			// Create 10 overflow logs
			for i := 0; i < 10; i++ {
				src := [16]byte{127, 0, 0, byte(i)}
				dst := [16]byte{127, 0, 0, byte(5 + i)}
				err := la.FeedUpdate(
					L7Update{Tuple: MakeTuple(src, dst, i, i, i), Count: 1})
				Expect(err).ShouldNot(HaveOccurred())
			}
			// Create 5 full logs
			for i := 0; i < 5; i++ {
				src := [16]byte{127, 0, 0, byte(i)}
				dst := [16]byte{127, 0, 0, byte(5 + i)}
				err := la.FeedUpdate(
					L7Update{Tuple: MakeTuple(src, dst, i, i, i), Duration: i,
						DurationMax: i, BytesSent: i, BytesReceived: i,
						ResponseCode: "200", Path: fmt.Sprintf("/%s", strconv.Itoa(i)), Count: 1, Type: "tcp"})
				Expect(err).ShouldNot(HaveOccurred())
			}
			Expect(la.l7Store).Should(HaveLen(5))
			Expect(la.l7OverflowStore).Should(HaveLen(5))
			// la.Get() will return the 5 full logs and 1 extra log to record
			// that there were 10 more updates that could not be fully logged.
			emitted := la.Get()
			Expect(emitted).To(HaveLen(6))
			// last log is of the type unlogged and containers unlogged messages
			last := emitted[len(emitted)-1]
			Expect(last.Type).To(Equal(L7LogTypeUnLogged))
			Expect(last.Count).To(Equal(10))
			// None of the logs should be overflow logs and have no Type
			numOverflow := 0
			for _, log := range emitted {
				if log.Type == "" {
					numOverflow++
				}
			}
			Expect(numOverflow).To(Equal(0))
		})
	})
})
