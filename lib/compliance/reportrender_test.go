// Copyright (c) 2019 Tigera, Inc. All rights reserved.

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

package compliance_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/libcalico-go/lib/compliance"
)

var _ = Describe("ReportTemplate Renderer", func() {
	It("inventory-summary report rendering", func() {
		tmpl := `startTime,endTime,endpointSelector,namespaceSelector,serviceAccountSelectors,endpointsNumInScope,endpointsNumIngressProtected,endpointsNumEgressProtected,endpointsNumIngressFromInternet,endpointsNumEgressToInternet,endpointsNumIngressFromOtherNamespace,endpointsNumEgressToOtherNamespace,endpointsNumEnvoyEnabled
{{ .StartTime }},{{ .EndTime }},{{ .ReportSpec.EndpointsSelection.EndpointSelector }},{{ .ReportSpec.EndpointsSelection.Namespaces.Selector }},{{ .ReportSpec.EndpointsSelection.ServiceAccounts.Selector }},{{ .EndpointsNumTotal }},{{ .EndpointsNumIngressProtected }},{{ .EndpointsNumEgressProtected }},{{ .EndpointsNumIngressFromInternet }},{{ .EndpointsNumEgressToInternet }},{{ .EndpointsNumIngressFromOtherNamespace }},{{ .EndpointsNumEgressToOtherNamespace }},{{ .EndpointsNumEnvoyEnabled }}`
		rendered := `startTime,endTime,endpointSelector,namespaceSelector,serviceAccountSelectors,endpointsNumInScope,endpointsNumIngressProtected,endpointsNumEgressProtected,endpointsNumIngressFromInternet,endpointsNumEgressToInternet,endpointsNumIngressFromOtherNamespace,endpointsNumEgressToOtherNamespace,endpointsNumEnvoyEnabled
2019-04-01 00:00:00 +0000 UTC,2019-04-01 10:00:00 +0000 UTC,lbl == 'lbl-val',sample-sel,sample-sel,10,10,10,10,10,10,10,10`

		matches, err := compliance.RenderTemplate(tmpl, compliance.ReportDataSample)
		Expect(err).ToNot(HaveOccurred())
		Expect(matches).To(Equal(rendered))
	})

	It("inventory-endpoints report rendering", func() {
		tmpl := `name,namespace,ingressProtected,egressProtected,envoyEnabled,appliedPolicies,services
{{ range .Endpoints -}}
  {{ .ID.Name }},{{ .ID.Namespace }},{{ .IngressProtected }},{{ .EgressProtected }},{{ .EnvoyEnabled }},{{ joinResources .AppliedPolicies ";" }},{{ joinResources .Services ";" }}
{{- end }}`
		rendered := `name,namespace,ingressProtected,egressProtected,envoyEnabled,appliedPolicies,services
sample-res,sample-ns,false,true,false,sample-kind(sample-ns/sample-res);sample-kind(sample-ns/sample-res),sample-kind(sample-ns/sample-res);sample-kind(sample-ns/sample-res)`

		matches, err := compliance.RenderTemplate(tmpl, compliance.ReportDataSample)
		Expect(err).ToNot(HaveOccurred())
		Expect(matches).To(Equal(rendered))
	})
})
