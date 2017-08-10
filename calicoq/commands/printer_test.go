// Copyright (c) 2017 Tigera, Inc. All rights reserved.

package commands_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	. "github.com/tigera/calicoq/calicoq/commands"
)

var _ = Describe("Test NewWorkloadEndpointPrintFromKey", func() {
	It("Creates a new WorkloadEndpointPrint Object with a WorkloadEndpointKey", func() {
		key := model.WorkloadEndpointKey{
			Hostname:       "testNode",
			OrchestratorID: "testOrchestrator",
			WorkloadID:     "testWorkload",
			EndpointID:     "testEndpoint",
		}

		wepp := NewWorkloadEndpointPrintFromKey(key)
		Expect(wepp.Node).To(Equal("testNode"))
		Expect(wepp.Orchestrator).To(Equal("testOrchestrator"))
		Expect(wepp.Workload).To(Equal("testWorkload"))
		Expect(wepp.Name).To(Equal("testEndpoint"))
	})

	It("Creates a new WorkloadEndpointPrint Object with a HostEndpointKey", func() {
		key := model.HostEndpointKey{
			EndpointID: "testEndpoint",
		}

		wepp := NewWorkloadEndpointPrintFromKey(key)
		Expect(wepp.Name).To(Equal("testEndpoint"))
	})

	It("Creates an empty WorkloadEndpointPrint Object if a different Key is given", func() {
		key := model.PolicyKey{
			Name: "testPolicy",
		}

		wepp := NewWorkloadEndpointPrintFromKey(key)
		Expect(wepp.Node).To(Equal(""))
		Expect(wepp.Orchestrator).To(Equal(""))
		Expect(wepp.Workload).To(Equal(""))
		Expect(wepp.Name).To(Equal(""))
	})
})

var _ = Describe("Test NewWorkloadEndpointPrintFromNameString", func() {
	It("Creates a new WorkloadEndpointPrint Object with a valid name string", func() {
		nameString := "Workload endpoint testNode/testOrchestrator/testWorkload/testName"
		wepp := NewWorkloadEndpointPrintFromNameString(nameString)
		Expect(wepp.Node).To(Equal("testNode"))
		Expect(wepp.Orchestrator).To(Equal("testOrchestrator"))
		Expect(wepp.Workload).To(Equal("testWorkload"))
		Expect(wepp.Name).To(Equal("testName"))
	})

	It("Creates an empty WorkloadEndpointPrint Object for invalid name strings", func() {
		tooManyWords := "Workload endpoint stuff testNode/testOrchestrator/testWorkload/testName"
		wepp := NewWorkloadEndpointPrintFromNameString(tooManyWords)
		Expect(wepp.Node).To(Equal(""))
		Expect(wepp.Orchestrator).To(Equal(""))
		Expect(wepp.Workload).To(Equal(""))
		Expect(wepp.Name).To(Equal(""))

		wrongType := "Policy endpoint testNode/testOrchestrator/testWorkload/testName"
		wepp = NewWorkloadEndpointPrintFromNameString(wrongType)
		Expect(wepp.Node).To(Equal(""))
		Expect(wepp.Orchestrator).To(Equal(""))
		Expect(wepp.Workload).To(Equal(""))
		Expect(wepp.Name).To(Equal(""))

		notEnoughIdents := "Workload endpoint testNode/testOrchestrator/testWorkload"
		wepp = NewWorkloadEndpointPrintFromNameString(notEnoughIdents)
		Expect(wepp.Node).To(Equal(""))
		Expect(wepp.Orchestrator).To(Equal(""))
		Expect(wepp.Workload).To(Equal(""))
		Expect(wepp.Name).To(Equal(""))
	})
})

var _ = Describe("Test NewRulePrintFromMatchString", func() {
	It("Creates a RulePrint Object with a valid match string", func() {
		matchString := "Policy \"testPolicy\" testDirection rule 1 testSelectorType match; selector \"testSelector\""
		rp := NewRulePrintFromMatchString(matchString)
		Expect(rp.PolicyName).To(Equal("testPolicy"))
		Expect(rp.Direction).To(Equal("testDirection"))
		Expect(rp.SelectorType).To(Equal("testSelectorType"))
		Expect(rp.Selector).To(Equal("testSelector"))
		Expect(rp.Order).To(Equal(1))
	})

	It("Creates an empty RulePrint Object for invalid match strings", func() {
		formatWrong := "Policy \"testPolicy\" testDirection rule 1 testSelectorType match something; selector \"testSelector\""
		rp := NewRulePrintFromMatchString(formatWrong)
		Expect(rp.PolicyName).To(Equal(""))
		Expect(rp.Direction).To(Equal(""))
		Expect(rp.SelectorType).To(Equal(""))
		Expect(rp.Selector).To(Equal(""))
		Expect(rp.Order).To(Equal(0))
	})
})

var _ = Describe("Test NewRulePrintFromSelectorString", func() {
	It("Creates a RulePrint Object with a valid selector string", func() {
		selString := "testDirection rule 1 testSelectorType match; selector \"testSelector\""
		rp := NewRulePrintFromSelectorString(selString)
		Expect(rp.Direction).To(Equal("testDirection"))
		Expect(rp.SelectorType).To(Equal("testSelectorType"))
		Expect(rp.Selector).To(Equal("testSelector"))
		Expect(rp.Order).To(Equal(1))
	})

	It("Creates an empty RulePrint Object with an invalid selector string", func() {
		hasPrefix := APPLICABLE_ENDPOINTS + " testDirection rule 1 testSelectorType match; selector testSelector"
		rp := NewRulePrintFromSelectorString(hasPrefix)
		Expect(rp.Direction).To(Equal(""))
		Expect(rp.SelectorType).To(Equal(""))
		Expect(rp.Selector).To(Equal(""))
		Expect(rp.Order).To(Equal(0))

		formatWrong := "testDirection ruleNum 1 testSelectorType match; selector testSelector"
		rp = NewRulePrintFromSelectorString(formatWrong)
		Expect(rp.Direction).To(Equal(""))
		Expect(rp.SelectorType).To(Equal(""))
		Expect(rp.Selector).To(Equal(""))
		Expect(rp.Order).To(Equal(0))
	})
})
