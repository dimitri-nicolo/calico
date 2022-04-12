// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package v1_test

import (
	"encoding/json"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/projectcalico/calico/lma/pkg/apis/v1"
)

var _ = Describe("Unmarshaling works correctly", func() {
	It("Errors with no from field", func() {
		var tr TimeRange
		d := "{\"to\":\"now\"}"

		err := json.Unmarshal([]byte(d), &tr)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("Request body contains an invalid value for the time range: missing `from` field"))
	})

	It("Errors with no to field", func() {
		var tr TimeRange
		d := "{\"from\":\"now\"}"

		err := json.Unmarshal([]byte(d), &tr)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("Request body contains an invalid value for the time range: missing `to` field"))
	})

	It("Parses relative times", func() {
		var tr TimeRange
		d := "{\"from\":\"now-15m\",\"to\":\"now\"}"

		err := json.Unmarshal([]byte(d), &tr)
		Expect(err).NotTo(HaveOccurred())
		Expect(tr.Now).NotTo(BeNil())
		Expect(tr.To).To(Equal(*tr.Now))
		Expect(tr.Duration()).To(Equal(15 * time.Minute))
	})

	It("Parses actual times", func() {
		var tr TimeRange
		d := "{\"from\":\"2021-05-30T21:23:10Z\", \"to\":\"2021-05-30T21:24:10Z\"}"

		err := json.Unmarshal([]byte(d), &tr)
		Expect(err).NotTo(HaveOccurred())
		Expect(tr.Now).To(BeNil())
		Expect(tr.Duration()).To(Equal(time.Minute))
	})

	It("Errors with mixed formats", func() {
		var tr TimeRange
		d := "{\"from\":\"2021-05-30T21:23:10Z\", \"to\":\"now\"}"

		err := json.Unmarshal([]byte(d), &tr)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("Request body contains an invalid time range: values must either both be explicit times or both be relative to now"))
	})

	It("Errors with reversed relative times", func() {
		var tr TimeRange
		d := "{\"from\":\"now\", \"to\":\"now-15m\"}"

		err := json.Unmarshal([]byte(d), &tr)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("Request body contains an invalid time range: from (now) is after to (now-15m)"))
	})

	It("Errors with reversed actual times", func() {
		var tr TimeRange
		d := "{\"from\":\"2021-05-30T21:23:10Z\", \"to\":\"2021-05-30T21:22:10Z\"}"

		err := json.Unmarshal([]byte(d), &tr)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("Request body contains an invalid time range: from (2021-05-30T21:23:10Z) is after to (2021-05-30T21:22:10Z)"))
	})

	It("Errors with bad time in from", func() {
		var tr TimeRange
		d := "{\"from\":\"now-X\", \"to\":\"now-15m\"}"

		err := json.Unmarshal([]byte(d), &tr)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("Request body contains an invalid value for the time range 'from' field: now-X"))
	})

	It("Errors with bad time in to", func() {
		var tr TimeRange
		d := "{\"from\":\"now-15m\", \"to\":\"now-X\"}"

		err := json.Unmarshal([]byte(d), &tr)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("Request body contains an invalid value for the time range 'to' field: now-X"))
	})
})
