// Copyright (c) 2020 Tigera, Inc. All rights reserved.
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

package ut_test

import (
	"net"
	"os"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/projectcalico/felix/bpf/conntrack"
)

func TestMapEntryDeletion(t *testing.T) {
	RegisterTestingT(t)
	k := conntrack.NewKey(1, net.ParseIP("10.0.0.1"), 51234, net.ParseIP("10.0.0.2"), 8080)
	v := conntrack.Value{}
	for i := range v {
		v[i] = uint8(i)
	}
	err1 := ctMap.Update(k.AsBytes(), v[:])
	err2 := ctMap.Delete(k.AsBytes())
	err3 := ctMap.Delete(k.AsBytes())

	// Defer error checking since the Delete calls do the cleanup for this test...
	Expect(err1).NotTo(HaveOccurred(), "Failed to create map entry")
	Expect(err2).NotTo(HaveOccurred(), "Failed to delete map entry")
	Expect(err3).To(Equal(os.ErrNotExist), "Error from deletion of non-existent entry was incorrect")
}
