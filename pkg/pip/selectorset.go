package pip

import (
	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/selector"
)

type selectorSet struct {
	selectors map[string]selector.Selector
}

func NewSelectorSet(npcs []NetworkPolicyChange) selectorSet {
	var ss = selectorSet{}
	for _, npc := range npcs {
		sel, err := selector.Parse(npc.NetworkPolicy.Spec.Selector)
		if err != nil {
			// todo: don't panic
			panic("invalid selector")
		}
		ss.selectors[npc.NetworkPolicy.Spec.Selector] = sel
	}
	return ss
}

func (s *selectorSet) anySelectorSelects(ep *v3.WorkloadEndpoint) bool {
	for _, sel := range s.selectors {
		if sel.Evaluate(ep.Labels) {
			return true
		}
	}
	return false
}
