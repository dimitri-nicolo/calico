// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package winfv_test

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/tigera/windows-networking/pkg/testutils"

	. "github.com/projectcalico/felix/fv/winfv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var allowDnsPolicy string = `
apiVersion: crd.projectcalico.org/v1
kind: NetworkPolicy
metadata:
  name: allow-dns
  namespace: demo
spec:
  order: 1
  selector: app == 'porter'
  types:
  - Egress
  egress:
  - action: Allow
    protocol: UDP
    destination:
      ports:
      - 53
`

var allowDomainPolicy string = `
apiVersion: crd.projectcalico.org/v1
kind: NetworkPolicy
metadata:
  name: allow-domain
  namespace: demo
spec:
  order: 1
  selector: app == 'porter'
  types:
  - Egress
  egress:
  - action: Allow
    destination:
      domains:
      - "gobyexample.com"
      - "*.google.com"
`

var _ = Describe("Windows DNS policy test", func() {
	var (
		fv       *WinFV
		err      error
		porterIP string
		dnsMap   []JsonMappingV1
	)

	Context("Check DNS policy", func() {
		BeforeEach(func() {
			fv, err = NewWinFV("c:\\CalicoWindows",
				"c:\\TigeraCalico\\flowlogs",
				"c:\\TigeraCalico\\felix-dns-cache.txt")
			Expect(err).NotTo(HaveOccurred())

			config := map[string]interface{}{
				"WindowsDNSExtraTTL":   "10",
				"DNSCacheSaveInterval": "10",
			}

			err := fv.AddConfigItems(config)
			Expect(err).NotTo(HaveOccurred())

			fv.RestartFelix()

			porterIP = testutils.InfraPodIP("porter", "demo")
			Expect(porterIP).NotTo(BeEmpty())

			testutils.KubectlApply("allow-dns.yaml", allowDnsPolicy)

			curlWithError("www.google.com")
			curlWithError("gobyexample.com")
		})

		AfterEach(func() {
			testutils.KubectlDelete("allow-domain.yaml")
			testutils.KubectlDelete("allow-dns.yaml")
		})

		// The test code does not check or depend on this, but currently:
		// www.google.com has one ip per DNS request.
		// gobyexample.com has 4 ips per DNS request.
		getDomainIPs := func(domain string) []string {
			result := []string{}
			for _, m := range dnsMap {
				if m.LHS == domain {
					result = append(result, m.RHS)
				}
			}
			return result
		}
		It("should get expected DNS policy", func() {
			// Apply DNS policy
			testutils.KubectlApply("allow-domain.yaml", allowDomainPolicy)

			// curl in Powershell will retransmit SYN packet if there are drops.
			// So we should see packets to go through after the DNS policy is in place.
			t1 := time.Now()
			curl("www.google.com")
			t2 := time.Now()
			curl("gobyexample.com")

			log.Printf("-----\nhttp www.google.com took %v seconds \n-----", t2.Sub(t1).Seconds())

			// Normally it would take around 5 seconds for the first http command to go through.
			// Add two seconds for execution and logging etc.
			Expect(t2.Before(t1.Add(7 * time.Second))).To(BeTrue())
			Expect(strings.Contains(getEndpointInfo(porterIP), "allow-domain")).To(BeTrue())

			displayDNS()
			// Sleep 15 seconds, 5 seconds more than DNS flush interval.
			time.Sleep(15 * time.Second)

			// Get IPs from DNS cache file
			dnsMap, err = fv.ReadDnsCacheFile()
			Expect(err).NotTo(HaveOccurred())
			log.Printf("dns map %v", dnsMap)

			googleIP := getDomainIPs("www.google.com")
			log.Printf("google ip %s", googleIP)

			goexampleIPs := getDomainIPs("gobyexample.com")
			log.Printf("gobyexample ip %v", goexampleIPs)

			Expect(len(googleIP)).NotTo(BeZero())
			Expect(len(goexampleIPs)).NotTo(BeZero())

			// Sleep further 45s (totally more than 60 seconds) so DNS TTL (30s) plus Extra TTL (10s) expires.
			time.Sleep(45 * time.Second)
			Expect(strings.Contains(getEndpointInfo(porterIP), "allow-domain")).To(BeFalse())
		})
	})

	Context("Check cleanup of the resources", func() {
		BeforeEach(func() {
			testutils.Powershell("Stop-Service -Name CalicoFelix")
		})

		It("should cleanup etw sessions", func() {
			output := testutils.Powershell("logman query -ets")
			log.Printf("-----\n%s\n-----", output)

			Expect(strings.Contains(output, "tigera")).To(BeFalse())
			Expect(strings.Contains(output, "PktMon")).To(BeFalse())
		})

		AfterEach(func() {
			testutils.Powershell("Start-Service -Name CalicoFelix")
		})
	})
})

func curl(target string) {
	cmd := fmt.Sprintf(`c:\k\kubectl.exe --kubeconfig=c:\k\config exec -t porter -n demo -- powershell.exe "curl %s -UseBasicParsing -TimeoutSec 10"`,
		target)
	output := testutils.Powershell(cmd)
	log.Printf("-----\n%s\n-----", output)
	Expect(strings.Contains(output, "200")).To(BeTrue())
}

func curlWithError(target string) {
	cmd := fmt.Sprintf(`c:\k\kubectl.exe --kubeconfig=c:\k\config exec -t porter -n demo -- powershell.exe "curl %s -UseBasicParsing -TimeoutSec 10"`,
		target)
	testutils.PowershellWithError(cmd)
}

func displayDNS() {
	cmd := `c:\k\kubectl.exe --kubeconfig=c:\k\config exec -t porter -n demo -- powershell.exe "ipconfig /displaydns"`
	output := testutils.Powershell(cmd)
	log.Printf("-----\n%s\n-----", output)
}

func getEndpointInfo(target string) string {
	cmd := fmt.Sprintf(` Get-HnsEndpoint | where  IpAddress -EQ %s | ConvertTo-Json -Depth 5`, target)
	output := testutils.Powershell(cmd)
	log.Printf("-----\n%s\n-----", output)
	return output
}
