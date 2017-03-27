// Copyright (c) 2016 Tigera, Inc. All rights reserved.

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
package autodetection_test

import (
	"github.com/projectcalico/calicoctl/calico_node/startup/autodetection"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Filtered enumeration tests", func() {

	Describe("No filters", func() {
		Context("Get interface and address", func() {

			iface, addr, err := autodetection.FilteredEnumeration(nil, nil, 4)
			It("should have enumerated at least one IP address", func() {
				Expect(err).To(BeNil())
				Expect(iface).ToNot(BeNil())
				Expect(addr).ToNot(BeNil())
			})
		})
	})
})
