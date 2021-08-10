// Copyright (c) 2021 Tigera, Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package timemanager_test

import (
	"time"

	"github.com/projectcalico/felix/nfqueue/timemanager"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("DNSPolicyPacketProcessor", func() {
	When("AddTime is called twice with the same ID within the expire duration", func() {
		It("the second call returns the same time", func() {
			timeManager := timemanager.New(
				timemanager.WithExpireDuration(800*time.Millisecond),
				timemanager.WithExpireTickDuration(200*time.Millisecond),
			)

			timeManager.Start()
			defer timeManager.Stop()

			t1 := timeManager.AddTime("test")
			time.Sleep(400 * time.Millisecond)
			t2 := timeManager.AddTime("test")

			Expect(t1.Equal(t2)).Should(BeTrue())
		})
	})

	When("AddTime is called a second time with the same ID after the expire duration", func() {
		It("the second call returns a newer time", func() {
			timeManager := timemanager.New(
				timemanager.WithExpireDuration(400*time.Millisecond),
				timemanager.WithExpireTickDuration(50*time.Millisecond),
			)

			timeManager.Start()
			defer timeManager.Stop()

			t1 := timeManager.AddTime("test")
			time.Sleep(800 * time.Millisecond)
			t2 := timeManager.AddTime("test")

			Expect(t2.After(t1)).Should(BeTrue())
		})
	})
})
