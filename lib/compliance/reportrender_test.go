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
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	api "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/compliance"
)

var _ = Describe("ReportTemplate Renderer", func() {
	It("inventory-summary report rendering", func() {
		tmpl := `startTime,endTime,endpointSelector,namespaceSelector,serviceAccountSelectors,endpointsNumInScope,endpointsNumIngressProtected,endpointsNumEgressProtected,endpointsNumIngressFromInternet,endpointsNumEgressToInternet,endpointsNumIngressFromOtherNamespace,endpointsNumEgressToOtherNamespace,endpointsNumEnvoyEnabled
{{ .StartTime }},{{ .EndTime }},{{ .ReportSpec.EndpointsSelection.EndpointSelector }},{{ .ReportSpec.EndpointsSelection.Namespaces.Selector }},{{ .ReportSpec.EndpointsSelection.ServiceAccounts.Selector }},{{ .EndpointsNumTotal }},{{ .EndpointsNumIngressProtected }},{{ .EndpointsNumEgressProtected }},{{ .EndpointsNumIngressFromInternet }},{{ .EndpointsNumEgressToInternet }},{{ .EndpointsNumIngressFromOtherNamespace }},{{ .EndpointsNumEgressToOtherNamespace }},{{ .EndpointsNumEnvoyEnabled }}`
		rendered := `startTime,endTime,endpointSelector,namespaceSelector,serviceAccountSelectors,endpointsNumInScope,endpointsNumIngressProtected,endpointsNumEgressProtected,endpointsNumIngressFromInternet,endpointsNumEgressToInternet,endpointsNumIngressFromOtherNamespace,endpointsNumEgressToOtherNamespace,endpointsNumEnvoyEnabled
2019-04-01 00:00:00 +0000 UTC,2019-04-01 10:00:00 +0000 UTC,lbl == 'lbl-val',endpoint-namespace-selector,serviceaccount-selector,1,10,100,1000,9000,900,90,9`

		matches, err := compliance.RenderTemplate(tmpl, compliance.ReportDataSample)
		Expect(err).ToNot(HaveOccurred())
		Expect(matches).To(Equal(rendered))
	})

	It("inventory-endpoints report rendering", func() {
		tmpl := `name,namespace,ingressProtected,egressProtected,envoyEnabled,appliedPolicies,services
{{ range .Endpoints -}}
  {{ .ID.Name }},{{ .ID.Namespace }},{{ .IngressProtected }},{{ .EgressProtected }},{{ .EnvoyEnabled }},{{ join .AppliedPolicies ";" }},{{ join .Services ";" }}
{{- end }}`
		rendered := `name,namespace,ingressProtected,egressProtected,envoyEnabled,appliedPolicies,services
sample-res,sample-ns,false,true,false,sample-kind(sample-ns/sample-res);sample-kind(sample-ns/sample-res),sample-kind(sample-ns/sample-res);sample-kind(sample-ns/sample-res)`

		matches, err := compliance.RenderTemplate(tmpl, compliance.ReportDataSample)
		Expect(err).ToNot(HaveOccurred())
		Expect(matches).To(Equal(rendered))
	})

	It("inventory-endpoints report rendering with | separator", func() {
		tmpl := `name,namespace,ingressProtected,egressProtected,envoyEnabled,appliedPolicies,services
{{ range .Endpoints -}}
  {{ .ID.Name }},{{ .ID.Namespace }},{{ .IngressProtected }},{{ .EgressProtected }},{{ .EnvoyEnabled }},{{ join .AppliedPolicies "|" }},{{ join .Services "|" }}
{{- end }}`
		rendered := `name,namespace,ingressProtected,egressProtected,envoyEnabled,appliedPolicies,services
sample-res,sample-ns,false,true,false,sample-kind(sample-ns/sample-res)|sample-kind(sample-ns/sample-res),sample-kind(sample-ns/sample-res)|sample-kind(sample-ns/sample-res)`

		matches, err := compliance.RenderTemplate(tmpl, compliance.ReportDataSample)
		Expect(err).ToNot(HaveOccurred())
		Expect(matches).To(Equal(rendered))
	})

	It("inventory-endpoints report rendering multiple items", func() {
		const endpointsCount = 100

		tmpl := `name,namespace,ingressProtected,egressProtected,envoyEnabled,appliedPolicies,services
{{ range .Endpoints -}}
  {{ .ID.Name }},{{ .ID.Namespace }},{{ .IngressProtected }},{{ .EgressProtected }},{{ .EnvoyEnabled }},{{ join .AppliedPolicies ";" }},{{ join .Services ";" }}
{{ end }}`
		rendered := `sample-res,sample-ns,false,true,false,sample-kind(sample-ns/sample-res);sample-kind(sample-ns/sample-res);sample-kind(sample-ns/sample-res);sample-kind(sample-ns/sample-res);sample-kind(sample-ns/sample-res);sample-kind(sample-ns/sample-res);sample-kind(sample-ns/sample-res);sample-kind(sample-ns/sample-res);sample-kind(sample-ns/sample-res);sample-kind(sample-ns/sample-res),sample-kind(sample-ns/sample-res);sample-kind(sample-ns/sample-res);sample-kind(sample-ns/sample-res);sample-kind(sample-ns/sample-res);sample-kind(sample-ns/sample-res);sample-kind(sample-ns/sample-res);sample-kind(sample-ns/sample-res);sample-kind(sample-ns/sample-res);sample-kind(sample-ns/sample-res);sample-kind(sample-ns/sample-res)`

		// Add entries to resource-list
		aer := compliance.EndpointSample
		for i := 1; i < 10-1; i++ {
			aer.AppliedPolicies = append(aer.AppliedPolicies, compliance.ResourceIdSample)
			aer.Services = append(aer.Services, compliance.ResourceIdSample)
		}
		// Add entries to endpoint-list
		endpoints := []api.EndpointsReportEndpoint{}
		for i := 0; i < endpointsCount; i++ {
			endpoints = append(endpoints, aer)
		}
		// Populate multiple endpoints with multiple resource entries as test data.
		ard := compliance.ReportDataSample
		ard.Endpoints = endpoints

		matches, err := compliance.RenderTemplate(tmpl, ard)
		Expect(err).ToNot(HaveOccurred())

		matches = strings.TrimSpace(matches) // remove last \n
		endpointList := strings.Split(matches, "\n")
		Expect(len(endpointList)).To(Equal(endpointsCount + 1)) // + Header
		Expect(endpointList[endpointsCount]).To(Equal(rendered))

		// Cap maximum entries
		capped_tmpl := `{{ range .Endpoints -}} {{ join .AppliedPolicies ";" 3 }} {{ end }}`

		matches, err = compliance.RenderTemplate(capped_tmpl, ard)
		Expect(err).ToNot(HaveOccurred())
		matches = strings.TrimSpace(matches) // remove last \n
		endpointList = strings.Split(matches, " ")
		resourceList := strings.Split(endpointList[0], ";")
		Expect(len(resourceList)).To(Equal(3))
	})

	It("inventory-endpoints report using ResourceID", func() {
		const k8sNetNamespace = "networking.k8s.io/v1"
		tmpl := "{{ range .Endpoints -}} {{ .ID }} {{- end }}"
		rendered := "sample-kind(sample-ns/sample-res)"
		renderedWithAPIVer := fmt.Sprintf("sample-kind.%s(sample-ns/sample-res)", k8sNetNamespace)

		matches, err := compliance.RenderTemplate(tmpl, compliance.ReportDataSample)
		Expect(err).ToNot(HaveOccurred())
		Expect(matches).To(Equal(rendered))

		resId := compliance.ResourceIdSample
		resId.APIVersion = k8sNetNamespace
		endpoint := compliance.EndpointSample
		endpoint.ID = resId
		rds := compliance.ReportDataSample
		rds.Endpoints = []api.EndpointsReportEndpoint{endpoint}

		matches, err = compliance.RenderTemplate(tmpl, rds)
		Expect(err).ToNot(HaveOccurred())
		Expect(matches).To(Equal(renderedWithAPIVer))
	})

	It("inventory-endpoints report failing with invalid argument", func() {
		// Wrong number of arguments
		tmpl := `{{ range .Endpoints -}} {{ join .AppliedPolicies }} {{ end }}`
		_, err := compliance.RenderTemplate(tmpl, compliance.ReportDataSample)
		Expect(err).To(HaveOccurred())

		// Invalid argument (not a slice)
		no_slice_tmpl := `{{ join .EndpointsNumTotal ";" }}`
		_, err = compliance.RenderTemplate(no_slice_tmpl, compliance.ReportDataSample)
		Expect(err).To(HaveOccurred())

		// Invalid max-entries argument
		invalid_capped_tmpl := `{{ range .Endpoints -}} {{ join .AppliedPolicies ";" "1" }} {{ end }}`
		_, err = compliance.RenderTemplate(invalid_capped_tmpl, compliance.ReportDataSample)
		Expect(err).To(HaveOccurred())
	})
})
