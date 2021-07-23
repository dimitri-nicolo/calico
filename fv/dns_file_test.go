// +build fvtests

// Copyright (c) 2019-2021 Tigera, Inc. All rights reserved.

package fv_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/felix/fv/containers"
	"github.com/projectcalico/felix/fv/infrastructure"
	"github.com/projectcalico/felix/fv/utils"
	"github.com/projectcalico/felix/fv/workload"
	api "github.com/tigera/api/pkg/apis/projectcalico/v3"
	client "github.com/projectcalico/libcalico-go/lib/clientv3"
)

var _ = Describe("_BPF-SAFE_ DNS Policy", func() {

	var (
		etcd   *containers.Container
		felix  *infrastructure.Felix
		w      *workload.Workload
		client client.Interface
		infra  infrastructure.DatastoreInfra
		dnsDir string
	)

	BeforeEach(func() {
		etcd = nil
		felix = nil
		w = nil
		var err error
		dnsDir, err = ioutil.TempDir("", "dnsinfo")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		if w != nil {
			w.Stop()
		}
		if felix != nil {
			felix.Stop()
		}
		if etcd != nil {
			etcd.Stop()
			infra.Stop()
		}
	})

	startWithPersistentFileContent := func(fileContent string) {
		// Populate the DNS info file that Felix will read.
		err := ioutil.WriteFile(path.Join(dnsDir, "dnsinfo.txt"), []byte(fileContent), 0644)
		Expect(err).NotTo(HaveOccurred())

		// Now start etcd and Felix.
		opts := infrastructure.DefaultTopologyOptions()
		opts.ExtraVolumes[dnsDir] = "/dnsinfo"
		opts.ExtraEnvVars["FELIX_DNSCACHEFILE"] = "/dnsinfo/dnsinfo.txt"
		opts.ExtraEnvVars["FELIX_PolicySyncPathPrefix"] = "/var/run/calico"
		felix, etcd, client, infra = infrastructure.StartSingleNodeEtcdTopology(opts)
	}

	Describe("file with 1000 entries", func() {

		It("should read and program those entries", func() {
			fileContent := "1\n"
			for i := 0; i < 1000; i++ {
				fileContent = fileContent + fmt.Sprintf(`{"LHS":"xyz.com","RHS":"10.10.%v.%v","Expiry":"3019-04-16T00:12:13Z","Type":"ip"}
`,
					(i/254)+1,
					(i%254)+1)
			}
			startWithPersistentFileContent(fileContent)

			w = workload.Run(felix, "w0", "default", "10.65.0.10", "8055", "tcp")
			w.Configure(client)

			policy := api.NewGlobalNetworkPolicy()
			policy.Name = "allow-xyz"
			order := float64(20)
			policy.Spec.Order = &order
			policy.Spec.Selector = "all()"
			policy.Spec.Egress = []api.Rule{
				{
					Action:      api.Allow,
					Destination: api.EntityRule{Domains: []string{"xyz.com"}},
				},
			}
			_, err := client.GlobalNetworkPolicies().Create(utils.Ctx, policy, utils.NoOptions)
			Expect(err).NotTo(HaveOccurred())

			if os.Getenv("FELIX_FV_ENABLE_BPF") == "true" {
				Eventually(func() error {
					ipsetsOutput, err := felix.ExecOutput("calico-bpf", "ipsets", "dump")
					if err != nil {
						return err
					}
					numMembers := 0
					for _, line := range strings.Split(ipsetsOutput, "\n") {
						if strings.HasPrefix(line, "IP set ") {
							// New IP set.
							numMembers = 0
						} else if strings.TrimSpace(line) != "" {
							// Member in current set.
							numMembers++
						} else {
							// Empty line => end of IP set.
							if numMembers == 1000 {
								return nil
							}
						}
					}
					return fmt.Errorf("No IP set with 1000 members in:\n[%v]", ipsetsOutput)
				}, "5s", "1s").Should(Succeed())
			} else {
				Eventually(func() int {
					for name, count := range getIPSetCounts(felix.Container) {
						if strings.HasPrefix(name, "cali40d:") {
							return count
						}
					}
					return 0
				}, "5s", "1s").Should(Equal(1000))
			}
		})
	})

	DescribeTable("Persistent file errors",
		func(fileContent string) {
			startWithPersistentFileContent(fileContent)

			// Now stop Felix again.
			felix.Stop()
			felix = nil

			// If Felix failed to cope with reading the persistent file, we'd either see
			// the start up call failing like this:
			//
			//    Container failed before being listed in 'docker ps'
			//
			// or the Stop() call would fail because of the container no longer existing.
		},

		Entry("Empty", ""),
		Entry("Just whitespace", `

`),
		Entry("Unsupported version", "6\n"),
		Entry("Supported version, no mappings", "1\n"),
		Entry("Supported version without newline", "1"),
		Entry("Non-JSOF content", `1
gobble de gook {
`),
		Entry("Truncated prematurely", `1
{"LHS":"xyz.com","RHS":"bob.xyz.com","Expiry":"2019-04-16T12:58:07Z","Type":"name"}
{"LHS":"server-5.xyz.com","RHS":"172.17.0.3","Expiry":"2019-04-16T1`),
		Entry("Extra fields present", `1
{"LHS":"xyz.com","RHS":"bob.xyz.com","Expiry":"2019-04-16T12:58:07Z","Type":"name","Bonus":"hey!"}
{"LHS":"server-5.xyz.com","Bonus":"hey!","RHS":"172.17.0.3","Expiry":"2019-04-16T12:58:07Z","Type":"ip"}
`),
		Entry("Mixed JSON and garbage", `1
{"LHS":"xyz.com","RHS":"bob.xyz.com","Expiry":"2019-04-16T12:58:07Z","Type":"name"}
      garbage
{"LHS":"server-5.xyz.com","RHS":"172.17.0.3","Expiry":"2019-04-16T12:58:07Z","Type":"ip"}
`),
	)
})
