// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package state

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("SendWithTimeout", func() {
	Context("timing out", func() {
		It("times out if the sender channel is blocked", func() {
			ch := make(chan SendInterface)
			defer close(ch)

			val := SendWithTimeout(ch, struct{}{}, 1*time.Second)
			Expect(val).Should(BeAssignableToTypeOf(&ErrChannelWriteTimeout{}))
		})

		It("times out if the receiver channel is blocked", func() {
			ch := make(chan SendInterface)
			defer close(ch)

			go func() {
				// Read whats sent but do nothing with the receiver channel to force the read timeout
				<-ch
			}()

			val := SendWithTimeout(ch, struct{}{}, 1*time.Second)
			Expect(val).Should(BeAssignableToTypeOf(&ErrChannelReadTimeout{}))
		})
	})
})
