// Copyright (c) 2020-2021 Tigera, Inc. All rights reserved.

package capture_test

import (
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/felix/capture"
)

var _ = Describe("PacketCapture Storage Tests", func() {
	var baseDir string

	BeforeEach(func() {
		var err error

		baseDir, err = ioutil.TempDir("/tmp", "pcap-tests")
		Expect(err).NotTo(HaveOccurred())

	})

	AfterEach(func() {
		var err = os.RemoveAll(baseDir)
		Expect(err).NotTo(HaveOccurred())
	})

	It("Cannot call stop with start", func() {
		var err error
		var activeCaptures capture.ActiveCaptures
		activeCaptures, err = capture.NewActiveCaptures(capture.Config{RotationSeconds: 1, Directory: baseDir}, make(chan interface{}))
		Expect(err).NotTo(HaveOccurred())
		err, _ = activeCaptures.Remove(capture.Key{CaptureName: "any"})
		Expect(err).To(HaveOccurred())
	})

	It("Cannot call start multiple times for the same capture", func() {
		var err error
		var activeCaptures capture.ActiveCaptures
		activeCaptures, err = capture.NewActiveCaptures(capture.Config{RotationSeconds: 1, Directory: baseDir}, make(chan interface{}))
		err = activeCaptures.Add(capture.Key{CaptureName: "any"}, "eth0")
		Expect(err).NotTo(HaveOccurred())
		err = activeCaptures.Add(capture.Key{CaptureName: "any"}, "eth0")
		Expect(err).To(HaveOccurred())
	})
})
