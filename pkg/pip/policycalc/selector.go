package policycalc

import (
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/libcalico-go/lib/selector"
)

// NewEndpointSelectorHandler creates a new EndpointSelectorHandler used for enumerating Endpoint selectors and
// temporarily caching the result.
func NewEndpointSelectorHandler() *EndpointSelectorHandler {
	return &EndpointSelectorHandler{
		selectorMatchers: make(map[string]EndpointMatcher),
	}
}

// EndpointSelectorHandler is used for enumerating Endpoint selectors and temporarily caching the result.
type EndpointSelectorHandler struct {
	selectorMatchers map[string]EndpointMatcher
}

// InitFlowDataSelectorResults initializes the cached selector results in the flow data.
func (c *EndpointSelectorHandler) CreateSelectorCache() []MatchType {
	return make([]MatchType, len(c.selectorMatchers))
}

// GetSelectorEndpointMatcher returns an Endpoint matcher function for a specific Endpoint selector.
func (c *EndpointSelectorHandler) GetSelectorEndpointMatcher(selStr string) EndpointMatcher {
	if m, ok := c.selectorMatchers[selStr]; ok {
		return m
	}

	// We don't have one, parse the selector string and create the Selector matcher.
	parsedSel, err := selector.Parse(selStr)
	if err != nil {
		// The selector is bad so we don't add it to the label helper.
		log.WithError(err).Errorf("Bad selector found in config: %s", selStr)
		return nil
	}

	// Create a closure to perform the selector matching and the caching.
	cacheIdx := len(c.selectorMatchers)
	matcher := func(_ *Flow, ep *FlowEndpointData) MatchType {
		if !ep.isCalicoEndpoint() {
			log.Debugf("Endpoint selector: %s (not Calico Endpoint)", MatchTypeFalse)
			return MatchTypeFalse
		}
		val := ep.cachedSelectorResults[cacheIdx]
		if val == MatchTypeUnknown {
			if parsedSel.EvaluateLabels(ep) {
				val = MatchTypeTrue
			} else {
				val = MatchTypeFalse
			}
			ep.cachedSelectorResults[cacheIdx] = val
		}
		log.Debugf("Endpoint selector: %s = %s", selStr, val)

		return val
	}

	c.selectorMatchers[selStr] = matcher

	return matcher
}
