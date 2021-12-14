// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package collector

import (
	"net"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("TupleStore stores data for metric updates", func() {
	Context("Stores in and out metric counters", func() {
		It("Aggregates updates from the same connection", func() {
			tupleStore := NewTupleStore()
			var update = muConn1Rule1HTTPReqAllowUpdate
			var key = tupleKey{update.tuple, FlowLogActionAllow, FlowLogReporterDst}

			By("Feeding in two updates containing HTTP request counts and same tuple")
			tupleStore.Store(update)
			tupleStore.Store(update)

			By("Reading the data stored against the tuple and calling a Reset")
			var store = make(map[tupleKey]tupleData)
			tupleStore.IterAndReset(func(key tupleKey, value tupleData) {
				store[key] = value
			})
			data := fetchData(store, key)

			// Expect started connections to be 1
			Expect(data.startedConnections).Should(Equal(1))
			Expect(data.completedConnections).Should(Equal(0))

			// Expect all counters for in metric to equal 2 times the size of update
			checkCounters(data.inMetric, update.inMetric, update.inMetric)

			// Expect all counters for out metric to equal 2 times the size of update
			checkCounters(data.outMetric, update.outMetric, update.outMetric)
		})

		It("Aggregates updates and expired from the same connection", func() {
			tupleStore := NewTupleStore()
			var update = muConn1Rule1AllowUpdate
			var expire = muConn1Rule1AllowExpire
			var key = tupleKey{update.tuple, FlowLogActionAllow, FlowLogReporterDst}

			By("Feeding in an update and an expire")
			tupleStore.Store(update)
			tupleStore.Store(expire)

			By("Reading the data stored against the tuple and calling a Reset")
			var store = make(map[tupleKey]tupleData)
			tupleStore.IterAndReset(func(key tupleKey, value tupleData) {
				store[key] = value
			})
			data := fetchData(store, key)

			// Expect started connections and completed connections to be 1
			Expect(data.startedConnections).Should(Equal(1))
			Expect(data.completedConnections).Should(Equal(1))
			Expect(data.hasExpired).To(Equal(true))

			// Expect all counters for metric to equal the size of the update and expire
			checkCounters(data.inMetric, muConn1Rule1AllowUpdate.inMetric, muConn1Rule1AllowExpire.inMetric)
			checkCounters(data.outMetric, muConn1Rule1AllowUpdate.outMetric, muConn1Rule1AllowExpire.outMetric)
		})

		It("Aggregates original source IPs", func() {
			tupleStore := NewTupleStore()
			var key = tupleKey{muWithOrigSourceIPs.tuple, FlowLogActionAllow, FlowLogReporterDst}

			By("Feeding in an update using the same tuple and public ips: IP1")
			tupleStore.Store(muWithOrigSourceIPs)
			By("Feeding in an update using using the same tuple and public ips: IP1, IP2")
			tupleStore.Store(muWithMultipleOrigSourceIPs)

			By("Reading the data stored against the tuple and calling a Reset")
			var store = make(map[tupleKey]tupleData)
			tupleStore.IterAndReset(func(key tupleKey, value tupleData) {
				store[key] = value
			})
			data := fetchData(store, key)

			// Expect started connections to be 1
			Expect(data.startedConnections).Should(Equal(1))
			Expect(data.completedConnections).Should(Equal(0))

			// expect IP1 and IP2
			var expectedOrigSourceIps = NewBoundedSet(testMaxBoundedSetSize)
			expectedOrigSourceIps.Add(net.ParseIP(publicIP1Str))
			expectedOrigSourceIps.Add(net.ParseIP(publicIP2Str))

			Expect(data.origSourceIPs.ToIPSlice()).To(ConsistOf(expectedOrigSourceIps.ToIPSlice()))
			Expect(data.origSourceIPs.TotalCount()).To(Equal(expectedOrigSourceIps.TotalCount()))
		})

		It("Aggregates labels", func() {
			tupleStore := NewTupleStore()
			var key = tupleKey{muWithEndpointMeta.tuple, FlowLogActionAllow, FlowLogReporterDst}

			By("Feeding in an update using the same tuple and labels test-app = true and k8s-app= true")
			tupleStore.Store(muWithEndpointMeta)
			By("Feeding in an update using using the same tuple and label test-app = true, new-label=true and k8s-app = false")
			tupleStore.Store(muWithEndpointMetaAndDifferentLabels)

			By("Reading the data stored against the tuple and calling a Reset")
			var store = make(map[tupleKey]tupleData)
			tupleStore.IterAndReset(func(key tupleKey, value tupleData) {
				store[key] = value
			})
			data := fetchData(store, key)

			// Expect started connections to be 1
			Expect(data.startedConnections).Should(Equal(1))
			Expect(data.completedConnections).Should(Equal(0))

			// expect test-app = true and new-label = true for source labels
			var expectedSrcLabels = map[string]string{"test-app": "true", "new-label": "true"}
			// expect k8s-app = false for destination labels
			var expectedDstLabels = map[string]string{"k8s-app": "false"}

			Expect(data.srcLabels).To(BeEquivalentTo(expectedSrcLabels))
			Expect(data.dstLabels).To(BeEquivalentTo(expectedDstLabels))
		})
	})

	Context("Calling reset on a store with data", func() {
		var tupleStore *tupleStore
		var key = tupleKey{muWithOrigSourceIPs.tuple, FlowLogActionAllow, FlowLogReporterDst}

		BeforeEach(func() {
			By("Creating a new tuple store")
			tupleStore = NewTupleStore()

			By("Feeding in an update")
			tupleStore.Store(muWithOrigSourceIPs)

			By("Reading the data stored against the tuple and calling a Reset")
			tupleStore.IterAndReset(func(key tupleKey, value tupleData) {
			})
		})

		It("Resets all metrics by calling Reset", func() {
			By("Reading again the data stored against the tuple and calling a Reset")
			var store = make(map[tupleKey]tupleData)
			tupleStore.IterAndReset(func(key tupleKey, value tupleData) {
				store[key] = value
			})
			dataAfterReset := fetchData(store, key)

			// Expect started connection to be zero
			Expect(dataAfterReset.startedConnections).Should(Equal(0))
			Expect(dataAfterReset.completedConnections).Should(Equal(0))

			// Expect all counters for in metric to equal 0
			checkEmptyCounters(dataAfterReset.inMetric)

			// Expect all counters for out metric to equal 0
			checkEmptyCounters(dataAfterReset.outMetric)

			//Expect all original source IPs to be empty
			Expect(dataAfterReset.origSourceIPs.TotalCount()).Should(Equal(0))
			Expect(len(dataAfterReset.origSourceIPs.ToIPSlice())).Should(Equal(0))
		})

		It("Stores an update after a reset", func() {
			By("Feeding in an update again")
			tupleStore.Store(muWithOrigSourceIPs)

			By("Reading again the data stored against the tuple")
			var store = make(map[tupleKey]tupleData)
			tupleStore.IterAndReset(func(key tupleKey, value tupleData) {
				store[key] = value
			})
			dataAfterUpdate := fetchData(store, key)

			// Expect started connection to be zero as the connection has not been purged
			Expect(dataAfterUpdate.startedConnections).Should(Equal(0))
			Expect(dataAfterUpdate.completedConnections).Should(Equal(0))

			// Expect all counters for metric to equal the size of the update
			checkCounters(dataAfterUpdate.inMetric, muWithOrigSourceIPs.inMetric)
			checkCounters(dataAfterUpdate.outMetric, muWithOrigSourceIPs.outMetric)
		})

		It("Purges data an Expire", func() {
			By("Feeding in an expire update")
			tupleStore.Store(muWithOrigSourceIPsExpire)

			By("Reading again the data stored against the tuple")
			var store = make(map[tupleKey]tupleData)
			tupleStore.IterAndReset(func(key tupleKey, value tupleData) {
				store[key] = value
			})
			dataAfterExpire := fetchData(store, key)

			// Expect completed connections to be 1
			Expect(dataAfterExpire.startedConnections).Should(Equal(0))
			Expect(dataAfterExpire.completedConnections).Should(Equal(1))

			// Expect all counters for metric to equal the size of the expire
			checkCounters(dataAfterExpire.inMetric, muWithOrigSourceIPsExpire.inMetric)
			checkCounters(dataAfterExpire.outMetric, muWithOrigSourceIPsExpire.outMetric)

			By("Resetting the tuple store")
			var storeAfterReset = make(map[tupleKey]tupleData)
			tupleStore.IterAndReset(func(key tupleKey, value tupleData) {
				storeAfterReset[key] = value
			})

			// Expect all data to be purged
			Expect(len(storeAfterReset)).Should(Equal(0))
		})
	})

	Context("Perform different actions on an empty storage", func() {
		var tupleStore *tupleStore

		BeforeEach(func() {
			By("Creating a new tuple store")
			tupleStore = NewTupleStore()
		})

		It("Calling fetch on an empty tuple store", func() {
			var store = make(map[tupleKey]tupleData)
			tupleStore.IterAndReset(func(key tupleKey, value tupleData) {
				store[key] = value
			})
			Expect(len(store)).To(Equal(0))
		})
	})
})

func checkEmptyCounters(metric MetricValue) {
	Expect(metric.deltaDeniedHTTPRequests).To(Equal(0))
	Expect(metric.deltaAllowedHTTPRequests).To(Equal(0))
	Expect(metric.deltaPackets).To(Equal(0))
	Expect(metric.deltaBytes).To(Equal(0))
}

func checkCounters(metric MetricValue, expected ...MetricValue) {
	if expected != nil {
		var value = combine(expected)
		Expect(metric.deltaDeniedHTTPRequests).To(Equal(value.deltaDeniedHTTPRequests))
		Expect(metric.deltaAllowedHTTPRequests).To(Equal(value.deltaAllowedHTTPRequests))
		Expect(metric.deltaPackets).To(Equal(value.deltaPackets))
		Expect(metric.deltaBytes).To(Equal(value.deltaBytes))
	}
}

func combine(expected []MetricValue) MetricValue {
	var result MetricValue
	for _, m := range expected {
		result.deltaPackets += m.deltaPackets
		result.deltaBytes += m.deltaBytes
		result.deltaAllowedHTTPRequests += m.deltaAllowedHTTPRequests
		result.deltaDeniedHTTPRequests += m.deltaDeniedHTTPRequests
	}

	return result
}

func fetchData(store map[tupleKey]tupleData, key tupleKey) tupleData {
	data, found := store[key]
	Expect(found).Should(Equal(true))
	Expect(data).NotTo(Equal(nil))

	return data
}
