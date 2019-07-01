package policycalc

import (
	"reflect"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Selector handler tests", func() {
	It("handles endpoint selector and results caching", func() {
		sh := NewEndpointSelectorHandler()

		By("creating an endpoints selector")
		m1 := sh.GetSelectorEndpointMatcher("all()")
		Expect(m1).NotTo(BeNil())

		By("creating the same endpoints selector and checking the cache size")
		m2 := sh.GetSelectorEndpointMatcher("all()")
		Expect(m2).NotTo(BeNil())
		Expect(reflect.ValueOf(m2)).To(Equal(reflect.ValueOf(m1)))
		Expect(sh.selectorMatchers).To(HaveLen(1))

		By("checking a different selector returns a different matcher function")
		m3 := sh.GetSelectorEndpointMatcher("vegetable == 'turnip'")
		Expect(m3).NotTo(BeNil())
		Expect(reflect.ValueOf(m3)).NotTo(Equal(reflect.ValueOf(m1)))
		Expect(sh.selectorMatchers).To(HaveLen(2))

		By("matching endpoint against the two selectors (both successfully)")
		ed := &FlowEndpointData{
			Type: EndpointTypeNs,
			Labels: map[string]string{
				"vegetable": "turnip",
			},
			cachedSelectorResults: sh.CreateSelectorCache(),
		}

		Expect(ed.cachedSelectorResults).To(HaveLen(2))
		Expect(ed.cachedSelectorResults[0]).To(Equal(MatchTypeNone))
		Expect(ed.cachedSelectorResults[1]).To(Equal(MatchTypeNone))

		Expect(m1(ed)).To(Equal(MatchTypeTrue))
		Expect(ed.cachedSelectorResults[0]).To(Equal(MatchTypeTrue))
		Expect(ed.cachedSelectorResults[1]).To(Equal(MatchTypeNone))

		Expect(m2(ed)).To(Equal(MatchTypeTrue))
		Expect(ed.cachedSelectorResults[0]).To(Equal(MatchTypeTrue))
		Expect(ed.cachedSelectorResults[1]).To(Equal(MatchTypeNone))

		Expect(m3(ed)).To(Equal(MatchTypeTrue))
		Expect(ed.cachedSelectorResults[0]).To(Equal(MatchTypeTrue))
		Expect(ed.cachedSelectorResults[1]).To(Equal(MatchTypeTrue))

		By("matching endpoint against the two selectors (only one successfully)")
		ed = &FlowEndpointData{
			Type: EndpointTypeHep,
			Labels: map[string]string{
				"vegetable": "parsnip",
			},
			cachedSelectorResults: sh.CreateSelectorCache(),
		}

		Expect(ed.cachedSelectorResults).To(HaveLen(2))
		Expect(ed.cachedSelectorResults[0]).To(Equal(MatchTypeNone))
		Expect(ed.cachedSelectorResults[1]).To(Equal(MatchTypeNone))

		Expect(m1(ed)).To(Equal(MatchTypeTrue))
		Expect(ed.cachedSelectorResults[0]).To(Equal(MatchTypeTrue))
		Expect(ed.cachedSelectorResults[1]).To(Equal(MatchTypeNone))

		Expect(m2(ed)).To(Equal(MatchTypeTrue))
		Expect(ed.cachedSelectorResults[0]).To(Equal(MatchTypeTrue))
		Expect(ed.cachedSelectorResults[1]).To(Equal(MatchTypeNone))

		Expect(m3(ed)).To(Equal(MatchTypeFalse))
		Expect(ed.cachedSelectorResults[0]).To(Equal(MatchTypeTrue))
		Expect(ed.cachedSelectorResults[1]).To(Equal(MatchTypeFalse))
	})
})
