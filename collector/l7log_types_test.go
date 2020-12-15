// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package collector

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("L7 log type tests", func() {
	Describe("L7Spec", func() {
		Context("Merge", func() {
			It("Merges correctly", func() {
				a := L7Spec{
					Duration:      36,
					DurationMax:   14,
					BytesReceived: 45,
					BytesSent:     68,
					Count:         3,
				}
				b := L7Spec{
					Duration:      16,
					DurationMax:   16,
					BytesReceived: 64,
					BytesSent:     32,
					Count:         1,
				}

				a.Merge(b)
				Expect(a.Duration).Should(Equal(52))
				Expect(a.DurationMax).Should(Equal(16))
				Expect(a.BytesReceived).Should(Equal(109))
				Expect(a.BytesSent).Should(Equal(100))
				Expect(a.Count).Should(Equal(4))
			})
		})
	})

	Describe("L7Data Tests", func() {
		meta := L7Meta{
			SrcNameAggr:  "client-*",
			SrcNamespace: "test-ns",
			SrcType:      FlowLogEndpointTypeWep,
			DstNameAggr:  "server-*",
			DstNamespace: "test-ns",
			DstType:      FlowLogEndpointTypeWep,
			ServiceNames: "svc1,svc2",
			ResponseCode: 200,
			Method:       "POST",
			Domain:       "www.server.com",
			Path:         "/test/path",
			UserAgent:    "firefox",
			Type:         "html/1.1",
		}
		spec := L7Spec{
			Duration:      52,
			DurationMax:   16,
			BytesReceived: 109,
			BytesSent:     100,
			Count:         4,
		}
		data := L7Data{meta, spec}

		It("Should create an appropriate L7 Log", func() {
			now := time.Now()
			end := now.Add(3 * time.Second)
			log := data.ToL7Log(now, end)

			Expect(log.StartTime).To(Equal(now))
			Expect(log.EndTime).To(Equal(end))
			Expect(log.SrcNameAggr).To(Equal(meta.SrcNameAggr))
			Expect(log.SrcNamespace).To(Equal(meta.SrcNamespace))
			Expect(log.SrcType).To(Equal(meta.SrcType))
			Expect(log.DstNameAggr).To(Equal(meta.DstNameAggr))
			Expect(log.DstNamespace).To(Equal(meta.DstNamespace))
			Expect(log.DstType).To(Equal(meta.DstType))
			Expect(log.ResponseCode).To(Equal(meta.ResponseCode))
			Expect(log.Method).To(Equal(meta.Method))
			Expect(log.URL).To(Equal("www.server.com/test/path"))
			Expect(log.UserAgent).To(Equal(meta.UserAgent))
			Expect(log.Type).To(Equal(meta.Type))
			Expect(log.DurationMean).To(Equal(13 * time.Millisecond))
			Expect(log.DurationMax).To(Equal(16 * time.Millisecond))
			Expect(log.BytesIn).To(Equal(109))
			Expect(log.BytesOut).To(Equal(100))
			Expect(log.Count).To(Equal(4))
			Expect(log.Services).To(HaveLen(2))
			Expect(log.Services[0].Name).To(Equal("svc1"))
			Expect(log.Services[1].Name).To(Equal("svc2"))
		})

	})
})
