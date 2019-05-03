// Copyright (c) 2016-2017 Tigera, Inc. All rights reserved.
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

package hashutils_test

import (
	. "github.com/tigera/compliance/pkg/hashutils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Id", func() {
	It("should return the suffix if short enough", func() {
		Expect(GetLengthLimitedName("felix", 10)).To(Equal("felix"))
	})
	It("should return a shortened hashed name when too long", func() {
		name := GetLengthLimitedName("1234567891123456789112345678910001234567891012345678911234567891123456789100012345678910", 50)
		Expect(name).To(HaveLen(50))
		Expect(name).To(Equal("123456789112345678911234567891000123456789-q6fu3r9"))
	})
})
