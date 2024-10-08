// Copyright (c) 2024 Tigera, Inc. All rights reserved.
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

package ut

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/projectcalico/calico/felix/bpf/stats"
	"github.com/projectcalico/calico/felix/bpf/tc"
	"github.com/projectcalico/calico/felix/bpf/utils"
)

func TestTcpStatsProgramCleanup(t *testing.T) {
	RegisterTestingT(t)

	bpffs, err := utils.MaybeMountBPFfs()
	Expect(err).NotTo(HaveOccurred())
	Expect(bpffs).To(Equal("/sys/fs/bpf"))
	t.Run("tcp stats cleanup", func(t *testing.T) {
		RegisterTestingT(t)
		vethName, veth := createVeth()
		defer deleteLink(veth)
		err = stats.AttachTcpStatsBpfProgram(vethName, "debug", 0)
		Expect(err).NotTo(HaveOccurred())

		getTcpProgId := func() int {
			cmd := exec.Command("bpftool", "prog", "list", "-j")
			out, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred())
			var progs []struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
				Maps []int  `json:"map_ids"`
			}

			err = json.Unmarshal(out, &progs)
			Expect(err).NotTo(HaveOccurred())

			for _, p := range progs {
				if strings.Contains(p.Name, "calico_tcp") {
					return p.ID
				}
			}
			return 0
		}
		Expect(getTcpProgId()).NotTo(Equal(0))
		tc.CleanUpTcpStatsPrograms()
		Expect(getTcpProgId()).To(Equal(0))
	})

}
