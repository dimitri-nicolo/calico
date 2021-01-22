// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package collector

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/tigera/l7-collector/pkg/config"
)

var (
	httpDestinationLog = "{\"downstream_remote_address\":\"192.168.138.208:34368\",\"connection_id\":0,\"type\":\"HTTP/1.1\",\"upstream_local_address\":\"192.168.35.210:58580\",\"duration\":3,\"downstream_local_address\":\"192.168.35.210:80\",\"user_agent\":\"curl/7.68.0\",\"response_code\":200,\"start_time\":\"2020-11-24T22:24:29.237Z\",\"request_id\":\"e23c0019-36b7-4142-8e86-39d15b00e965\",\"upstream_host\":\"192.168.35.210:80\",\"bytes_received\":0,\"request_path\":\"/ip\",\"hostname\":\"httpbin-584c76bfcb-74jx4\",\"downstream_direct_remote_address\":\"192.168.138.208:34368\",\"reporter\":\"destination\",\"bytes_sent\":33,\"request_method\":\"GET\"}"
	httpSourceLog      = "{\"duration\":6,\"downstream_local_address\":\"10.105.83.191:80\",\"user_agent\":\"curl/7.68.0\",\"response_code\":200,\"start_time\":\"2020-11-24T22:24:29.238Z\",\"request_id\":\"e23c0019-36b7-4142-8e86-39d15b00e965\",\"upstream_host\":\"10.105.83.191:80\",\"bytes_received\":0,\"request_path\":\"/ip\",\"hostname\":\"ubuntu-76895788d9-tkkr4\",\"downstream_direct_remote_address\":\"192.168.138.208:34366\",\"reporter\":\"source\",\"bytes_sent\":33,\"request_method\":\"GET\",\"connection_id\":0,\"downstream_remote_address\":\"192.168.138.208:34366\",\"type\":\"HTTP/1.1\",\"upstream_local_address\":\"192.168.138.208:34368\"}"
	tcpDestinationLog  = "{\"reporter\":\"destination\",\"bytes_sent\":7,\"request_method\":null,\"downstream_remote_address\":\"192.168.138.208:46330\",\"connection_id\":0,\"type\":\"tcp\",\"upstream_local_address\":\"192.168.45.171:34674\",\"duration\":2,\"downstream_local_address\":\"192.168.45.171:6379\",\"response_code\":0,\"user_agent\":null,\"start_time\":\"2020-11-24T22:33:32.279Z\",\"request_id\":null,\"upstream_host\":\"192.168.45.171:6379\",\"request_path\":null,\"bytes_received\":14,\"hostname\":\"redis-6d765dd54b-4695l\",\"downstream_direct_remote_address\":\"192.168.138.208:46330\"}"
	badFormatLog       = "{\"request_path\":null,\"bytes_received\":14,\"hostname\":\"ubuntu-76895788d9-tkkr4\",\"downstream_direct_remote_address\":\"192.168.138.208:46328\",\"reporter\":\"source\",\"bytes_sent\":7,\"request_method\":null,\"downstream_remote_address\":\"192.168.138.208:46328\",\"connection_id\":4,\"protocol\":\"tcp\",\"upstream_local_address\":\"192.168.138.208:46330\",\"duration\":5,\"downstream_local_address\":\"10.102.217.45:6379\",\"response_code\":0,\"user_agent\":null,\"start_time\":\"2020-11-24T22:33:32.274Z\",\"request_id\":null,\"upstream_host\":\"10.102.217.45:6379"
	httpIPv6Log        = "{\"downstream_remote_address\":\"[2001:db8:a0b:12f0::1]:56080\",\"connection_id\":0,\"type\":\"HTTP/1.1\",\"upstream_local_address\":\"192.168.35.210:58580\",\"duration\":3,\"downstream_local_address\":\"192.168.35.210:80\",\"user_agent\":\"curl/7.68.0\",\"response_code\":200,\"start_time\":\"2020-11-24T22:24:29.237Z\",\"request_id\":\"e23c0019-36b7-4142-8e86-39d15b00e965\",\"upstream_host\":\"192.168.35.210:80\",\"bytes_received\":0,\"request_path\":\"/ip\",\"hostname\":\"httpbin-584c76bfcb-74jx4\",\"downstream_direct_remote_address\":\"192.168.138.208:34368\",\"reporter\":\"destination\",\"bytes_sent\":33,\"request_method\":\"GET\"}"
)

var _ = Describe("Envoy Log Collector ParseRawLogs test", func() {
	// Can use an empty config since the config is not used in ParseRawLogs
	c := EnvoyCollectorNew(&config.Config{})

	Context("With a log with HTTP destination json format", func() {
		It("should return the expected EnvoyLog", func() {
			log, err := c.ParseRawLogs(httpDestinationLog)
			Expect(err).To(BeNil())
			Expect(log.SrcIp).To(Equal("192.168.138.208"))
			Expect(log.DstIp).To(Equal("192.168.35.210"))
			Expect(log.SrcPort).To(Equal(int32(34368)))
			Expect(log.DstPort).To(Equal(int32(80)))
		})
	})
	Context("With a log with TCP destination json format", func() {
		It("should return the expected EnvoyLog", func() {
			log, err := c.ParseRawLogs(tcpDestinationLog)
			Expect(err).To(BeNil())
			Expect(log.SrcIp).To(Equal("192.168.138.208"))
			Expect(log.DstIp).To(Equal("192.168.45.171"))
			Expect(log.SrcPort).To(Equal(int32(46330)))
			Expect(log.DstPort).To(Equal(int32(6379)))
		})
	})
	Context("With a log with no closing brace for the information json", func() {
		It("should return an error", func() {
			_, err := c.ParseRawLogs(badFormatLog)
			Expect(err).NotTo(BeNil())
		})
	})
	Context("With a log with IPv6 IP address format", func() {
		It("should return the expected EnvoyLog", func() {
			log, err := c.ParseRawLogs(httpIPv6Log)
			Expect(err).To(BeNil())
			Expect(log.SrcIp).To(Equal("2001:db8:a0b:12f0::1"))
			Expect(log.DstIp).To(Equal("192.168.35.210"))
			Expect(log.SrcPort).To(Equal(int32(56080)))
			Expect(log.DstPort).To(Equal(int32(80)))
		})
	})
	Context("With a log which is not a destination log", func() {
		It("should return empty EnvoyLog", func() {
			_, err := c.ParseRawLogs(httpSourceLog)
			Expect(err).NotTo(BeNil())
		})
	})
})
