// +build fvtests

// Copyright (c) 2019 Tigera, Inc. All rights reserved.
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
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/felix/fv/containers"
	"github.com/projectcalico/felix/fv/infrastructure"
	"github.com/projectcalico/felix/fv/workload"
	client "github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/set"
)

type mapping struct {
	lhs, rhs string
}

var _ = Describe("DNS Policy", func() {

	var (
		scapy  *containers.Container
		etcd   *containers.Container
		felix  *infrastructure.Felix
		client client.Interface
		w      [1]*workload.Workload
		dnsDir string
	)

	BeforeEach(func() {
		opts := infrastructure.DefaultTopologyOptions()
		var err error
		dnsDir, err = ioutil.TempDir("", "dnsinfo")
		Expect(err).NotTo(HaveOccurred())

		// Start scapy first, so we can get its IP and configure Felix to trust it.
		scapy = containers.Run("scapy",
			containers.RunOpts{AutoRemove: true, WithStdinPipe: true},
			"-i", "--privileged", "scapy")
		scapy.WaitUntilRunning()

		// Now start etcd and Felix, with Felix trusting scapy's IP.
		opts.ExtraVolumes[dnsDir] = "/dnsinfo"
		opts.ExtraEnvVars["FELIX_DOMAININFOSTORE"] = "/dnsinfo/dnsinfo.txt"
		opts.ExtraEnvVars["FELIX_DOMAININFOSAVEINTERVAL"] = "1"
		opts.ExtraEnvVars["FELIX_DOMAININFOTRUSTEDSERVERS"] = scapy.IP
		felix, etcd, client = infrastructure.StartSingleNodeEtcdTopology(opts)
		infrastructure.CreateDefaultProfile(client, "default", map[string]string{"default": ""}, "")

		// Create a workload, using that profile.
		for ii := range w {
			iiStr := strconv.Itoa(ii)
			w[ii] = workload.Run(felix, "w"+iiStr, "default", "10.65.0.1"+iiStr, "8055", "tcp")
			w[ii].Configure(client)
		}
	})

	// Stop etcd and workloads, collecting some state if anything failed.
	AfterEach(func() {
		if CurrentGinkgoTestDescription().Failed {
			felix.Exec("ipset", "list")
			felix.Exec("iptables-save", "-c")
			felix.Exec("ip", "r")
			felix.Exec("conntrack", "-L")
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

	fileHasMappings := func(mappings []mapping) func() bool {
		mset := set.FromArray(mappings)
		return func() bool {
			f, err := os.Open(path.Join(dnsDir, "dnsinfo.txt"))
			if err == nil {
				scanner := bufio.NewScanner(f)
				for scanner.Scan() {
					line := scanner.Text()
					mset.Iter(func(item interface{}) error {
						m := item.(mapping)
						if strings.Contains(line, "\""+m.lhs+"\"") && strings.Contains(line, "\""+m.rhs+"\"") {
							return set.RemoveItem
						}
						return nil
					})
					if mset.Len() == 0 {
						return true
					}
				}
			}
			return false
		}
	}

	fileHasMapping := func(lname, rname string) func() bool {
		return fileHasMappings([]mapping{{lhs: lname, rhs: rname}})
	}

	DescribeTable("DNS response processing",
		func(dnsSpec string, check func() bool) {
			// Establish conntrack state, in Felix, as though the workload just sent a
			// DNS request to scapy.
			felix.Exec("conntrack", "-I", "-s", w[0].IP, "-d", scapy.IP, "-p", "UDP", "-t", "10", "--sport", "53", "--dport", "53")

			// Now drive scapy.
			go func() {
				defer scapy.Stdin.Close()
				io.WriteString(scapy.Stdin,
					fmt.Sprintf("conf.route.add(host='%v',gw='%v')\n", w[0].IP, felix.IP))
				io.WriteString(scapy.Stdin,
					fmt.Sprintf("send(IP(dst='%v')/UDP(sport=53)/%v)\n", w[0].IP, dnsSpec))
			}()

			// Run the check function.
			Eventually(check, "5s", "1s").Should(BeTrue())
		},

		Entry("A record",
			"DNS(qr=1,qdcount=1,ancount=1,qd=DNSQR(qname='bankofsteve.com',qtype='A'),an=(DNSRR(rrname='bankofsteve.com',type='A',ttl=36000,rdata='192.168.56.1')))",
			fileHasMapping("bankofsteve.com", "192.168.56.1"),
		),
		Entry("AAAA record",
			"DNS(qr=1,qdcount=1,ancount=1,qd=DNSQR(qname='bankofsteve.com',qtype='AAAA'),an=(DNSRR(rrname='bankofsteve.com',type='AAAA',ttl=36000,rdata='fdf5:8944::3')))",
			fileHasMapping("bankofsteve.com", "fdf5:8944::3"),
		),
		Entry("CNAME record",
			"DNS(qr=1,qdcount=1,ancount=1,qd=DNSQR(qname='bankofsteve.com',qtype='CNAME'),an=(DNSRR(rrname='bankofsteve.com',type='CNAME',ttl=36000,rdata='my.home.server')))",
			fileHasMapping("bankofsteve.com", "my.home.server"),
		),
		Entry("3 A records",
			"DNS(qr=1,qdcount=1,ancount=3,qd=DNSQR(qname='microsoft.com',qtype='A'),an=("+
				"DNSRR(rrname='microsoft.com',type='A',ttl=24,rdata='19.16.5.102')/"+
				"DNSRR(rrname='microsoft.com',type='A',ttl=36,rdata='10.146.25.132')/"+
				"DNSRR(rrname='microsoft.com',type='A',ttl=48,rdata='35.5.5.199')"+
				"))",
			fileHasMappings([]mapping{
				{lhs: "microsoft.com", rhs: "19.16.5.102"},
				{lhs: "microsoft.com", rhs: "10.146.25.132"},
				{lhs: "microsoft.com", rhs: "35.5.5.199"},
			}),
		),
	)
})
