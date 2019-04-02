// +build fvtests

// Copyright (c) 2017-2019 Tigera, Inc. All rights reserved.
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

package fv_test

import (
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"

	. "github.com/onsi/ginkgo"

	"github.com/projectcalico/felix/fv/containers"
	"github.com/projectcalico/felix/fv/infrastructure"
	"github.com/projectcalico/felix/fv/workload"
	client "github.com/projectcalico/libcalico-go/lib/clientv3"
)

var _ = Describe("DNS Policy", func() {

	var (
		etcd   *containers.Container
		felix  *infrastructure.Felix
		client client.Interface
		w      [1]*workload.Workload
	)

	BeforeEach(func() {
		opts := infrastructure.DefaultTopologyOptions()
		felix, etcd, client = infrastructure.StartSingleNodeEtcdTopology(opts)
		infrastructure.CreateDefaultProfile(client)

		// Create a workload, using that profile.
		for ii := range w {
			iiStr := strconv.Itoa(ii)
			w[ii] = workload.Run(felix, "w"+iiStr, "default", "10.65.0.1"+iiStr, "8055", "tcp")
			w[ii].Configure(client)
		}

		// Allow workloads to connect out to the Internet.
		felix.Exec(
			"iptables", "-w", "-t", "nat",
			"-A", "POSTROUTING",
			"-o", "eth0",
			"-j", "MASQUERADE",
		)
	})

	// Stop etcd and workloads, collecting some state if anything failed.
	AfterEach(func() {
		if CurrentGinkgoTestDescription().Failed {
			felix.Exec("iptables-save", "-c")
			felix.Exec("ip", "r")
		}

		for ii := range w {
			w[ii].Stop()
		}
		felix.Stop()

		if CurrentGinkgoTestDescription().Failed {
			etcd.Exec("etcdctl", "ls", "--recursive", "/")
		}
		etcd.Stop()
	})

	Context("Connectivity to tigera.io", func() {
		It("can be established by default", func() {
			for i := 0; i < 3; i++ {
				out, err := w[0].ExecOutput("nslookup", "microsoft.com", "8.8.8.8")
				log.WithError(err).Infof("nslookup said:\n%v", out)
				time.Sleep(3 * time.Second)
			}
			felix.Exec("iptables-save", "-c")
		})

		It("cannot be established when specified", func() {
			for i := 0; i < 3; i++ {
				out, err := w[0].ExecOutput("nslookup", "microsoft.com", "8.8.8.8")
				log.WithError(err).Infof("nslookup said:\n%v", out)
				time.Sleep(3 * time.Second)
			}
		})

	})
})
