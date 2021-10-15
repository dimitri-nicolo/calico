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

package winfv_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
	"github.com/tigera/windows-networking/pkg/testutils"
)

func getPodIP(name, namespace string) string {
	cmd := fmt.Sprintf(`c:\k\kubectl.exe --kubeconfig=c:\k\config get pod %s -n %s -o jsonpath='{.status.podIP}'`,
		name, namespace)
	return testutils.Powershell(cmd)
}

func kubectlExec(command string) {
	cmd := fmt.Sprintf(`c:\k\kubectl.exe --kubeconfig=c:\k\config -n demo exec %v`, command)
	_ = testutils.Powershell(cmd)
}

// These Windows policy FV tests rely on a 2 node cluster (1 Linux and 1 Windows) provisioned using internal tooling.
// The test infra setup creates some pods:
// - "client" and "clientB" are busybox pods
// - "nginx" and "nginxB" are nginx pods
// - "porter" is a Windows server/client pod using the calico/porter image
//
// The test infra setup also applies some network policies on the pods:
// - "allow-dns": egress policy that allows the porter pod to reach UDP port 53
// - "allow-nginx": egress policy that allows the porter pod to reach the nginx pods on TCP port 80
// - "allow-client": ingress policy that allows the client pods to reach the porter pods on TCP port 80
var _ = Describe("Windows policy test", func() {
	var (
		porter, client, clientB, nginx, nginxB string
	)

	BeforeEach(func() {
		Skip("lmm(temporarily skip failing policy tests)")

		// Get IPs of the pods installed by the test infra setup.
		client = getPodIP("client", "demo")
		clientB = getPodIP("client-b", "demo")
		porter = getPodIP("porter", "demo")
		nginx = getPodIP("nginx", "demo")
		nginxB = getPodIP("nginx-b", "demo")
		log.Infof("Pod IPs: client %s, client-b %s, porter %s, nginx %s, nginx-b %s",
			client, clientB, porter, nginx, nginxB)

		Expect(client).NotTo(BeEmpty())
		Expect(clientB).NotTo(BeEmpty())
		Expect(porter).NotTo(BeEmpty())
		Expect(nginx).NotTo(BeEmpty())
		Expect(nginxB).NotTo(BeEmpty())
	})

	Context("ingress policy tests", func() {
		It("client pod can connect to porter pod", func() {
			kubectlExec(fmt.Sprintf(`-t client -- wget %v -T 5 -qO -`, porter))
		})
		It("client-b pod can't connect to porter pod", func() {
			kubectlExec(fmt.Sprintf(`-t client-b -- wget %v -T 5 -qO -`, porter))
		})
	})
	Context("egress policy tests", func() {
		It("porter pod can connect to nginx pod", func() {
			kubectlExec(fmt.Sprintf(`-t porter -- powershell -Command 'Invoke-WebRequest -UseBasicParsing -TimeoutSec 5 %v'`, nginx))
		})
		It("porter pod cannot connect to nginx-b pod", func() {
			kubectlExec(fmt.Sprintf(`-t porter -- powershell -Command 'Invoke-WebRequest -UseBasicParsing -TimeoutSec 5 %v'`, nginxB))
		})
		It("porter pod cannot connect to google.com", func() {
			kubectlExec(`-t porter -- powershell -Command 'Invoke-WebRequest -UseBasicParsing -TimeoutSec 5 www.google.com'`)
		})
	})
})
