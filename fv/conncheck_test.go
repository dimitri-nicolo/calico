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

// +build fvtests

package fv_test

import (
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo"

	"github.com/projectcalico/felix/fv/connectivity"

	"github.com/projectcalico/libcalico-go/lib/apiconfig"

	"github.com/projectcalico/felix/fv/infrastructure"
	"github.com/projectcalico/felix/fv/workload"
)

// These tests verify that test-connection and test-workload work properly across all the different protocols.
var _ = describeConnCheckTests("tcp")
var _ = describeConnCheckTests("sctp")
var _ = describeConnCheckTests("udp")
var _ = describeConnCheckTests("udp-recvmsg")
var _ = describeConnCheckTests("udp-noconn")

func describeConnCheckTests(protocol string) bool {
	return infrastructure.DatastoreDescribe("Connectivity sanity checks: "+protocol,
		[]apiconfig.DatastoreType{apiconfig.EtcdV3, apiconfig.Kubernetes},
		func(getInfra infrastructure.InfraFactory) {

			var (
				infra   infrastructure.DatastoreInfra
				felixes []*infrastructure.Felix
				hostW   [2]*workload.Workload
				cc      *connectivity.Checker
			)

			BeforeEach(func() {
				infra = getInfra()
				felixes, _ = infrastructure.StartNNodeTopology(2, infrastructure.DefaultTopologyOptions(), infra)

				// Create host-networked "workloads", one on each "host".
				for ii := range felixes {
					// Workload doesn't understand the extra connectivity types that test-connection tries.
					wlProto := protocol
					if strings.Contains(protocol, "-") {
						wlProto = strings.Split(protocol, "-")[0]
					}
					hostW[ii] = workload.Run(felixes[ii], fmt.Sprintf("host%d", ii), "", felixes[ii].IP, "8055", wlProto)
				}

				cc = &connectivity.Checker{}
				cc.Protocol = protocol
			})

			AfterEach(func() {
				for _, wl := range hostW {
					wl.Stop()
				}
				for _, felix := range felixes {
					felix.Stop()
				}

				if CurrentGinkgoTestDescription().Failed {
					infra.DumpErrorData()
				}
				infra.Stop()
			})

			It("should have host-to-host", func() {
				cc.ExpectSome(felixes[0], hostW[1])
				cc.CheckConnectivity()
			})
		})
}
