package policycalc

import (
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/tigera/es-proxy/pkg/pip/flow"

	v1 "k8s.io/api/core/v1"

	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/selector"
)

// NewNamespaceHandler creates a new NamespaceHandler.
func NewNamespaceHandler(n []*v1.Namespace, sa []*v1.ServiceAccount) *NamespaceHandler {
	nh := &NamespaceHandler{
		namespaces:             make(map[string]*namespaceData),
		selectorMatchers:       make(map[string]EndpointMatcher),
		serviceAccountMatchers: make(map[string]EndpointMatcher),
	}
	for i := range n {
		nh.setNamespaceLabels(
			n[i].Name, n[i].Labels,
		)
	}
	for i := range sa {
		nh.setServiceAccountLabels(
			sa[i].Namespace, sa[i].Name, sa[i].Labels,
		)
	}
	return nh
}

// NamespaceHandler is used for handling namespace selector matches. The handler should be configured from the resource
// data, and once programmed can provide compiled namespace selector matchers.
type NamespaceHandler struct {
	namespaces             map[string]*namespaceData
	selectorMatchers       map[string]EndpointMatcher
	serviceAccountMatchers map[string]EndpointMatcher
}

// GetNamespaceSelectorEndpointMatcher returns a namespace selector EndpointMatcher. The matcher pre-compiles the list
// of matching namespaces.
func (n *NamespaceHandler) GetNamespaceSelectorEndpointMatcher(selStr string) EndpointMatcher {
	// Use the previously cached namespace selector matcher.
	if m, ok := n.selectorMatchers[selStr]; ok {
		log.WithField("selector", selStr).Debug("Returning cached namespace selector")
		return m
	}

	// We don't have one, parse the selector string and create the Selector matcher.
	parsedSel, err := selector.Parse(selStr)
	if err != nil {
		// The selector is bad so we don't add it.
		log.WithError(err).Errorf("Bad selector found in config: %s", selStr)
		return nil
	}

	// Construct a slice of namespaces whose labels match the selector.
	namespaces := make([]string, 0)
	for name, ns := range n.namespaces {
		if parsedSel.Evaluate(ns.labels) {
			log.WithField("selector", selStr).Debugf("Selector matches namespace %s", name)
			namespaces = append(namespaces, name)
		}
	}

	// Create a closure to perform the match.
	matcher := func(ep *FlowEndpointData) MatchType {
		// If the Endpoint namespace is one of the matched selectors then this matches.
		for i := range namespaces {
			if namespaces[i] == ep.Namespace {
				return MatchTypeTrue
			}
		}
		return MatchTypeFalse
	}

	// Cache the matcher for re-use.
	n.selectorMatchers[selStr] = matcher
	return matcher
}

// GetServiceAccountEndpointMatchers returns a service account EndpointMatcher. The matcher pre-compiles the list of
// matching service accounts.
func (n *NamespaceHandler) GetServiceAccountEndpointMatchers(sa *v3.ServiceAccountMatch) EndpointMatcher {
	if sa == nil || (len(sa.Names) == 0 && sa.Selector == "") {
		return nil
	}

	// Use the previously cached namespace selector matcher.
	key := sa.Selector + "/" + strings.Join(sa.Names, "/")
	if m, ok := n.serviceAccountMatchers[key]; ok {
		log.WithField("match", sa).Debug("Returning cached service account matcher")
		return m
	}

	// Track which service accounts in each namespace match the selector.
	saNames := sa.Names
	var saNamesPerNamespace map[string][]string

	var parsedSel selector.Selector
	if sa.Selector != "" {
		// We have a selector, so initialize the names per namespace map.
		saNamesPerNamespace = make(map[string][]string, 0)

		var err error
		parsedSel, err = selector.Parse(sa.Selector)
		if err != nil {
			// The selector is bad so we don't add it.
			log.WithError(err).Errorf("Bad selector found in config: %s", sa.Selector)
			return nil
		}

		// Construct a slice of service accounts whose labels match the selector.
		for name, ns := range n.namespaces {
			if s := ns.getServiceAccounts(parsedSel); s != nil {
				saNamesPerNamespace[name] = s
			}
		}
	}

	// Create a closure to perform the match.
	matcher := func(ep *FlowEndpointData) MatchType {
		if ep.Type != flow.EndpointTypeWep {
			return MatchTypeFalse
		}

		if saNamesPerNamespace != nil && len(saNamesPerNamespace) == 0 {
			// If we have a selector match, but the selector doesn't select any validly configured service account
			// then this is a no match - this trumps the case where the endpoint service account is not known.
			return MatchTypeFalse
		}

		if ep.ServiceAccount == nil {
			// The service account value is not available, so the match type is uncertain.
			return MatchTypeUncertain
		}

		matched := len(saNames) == 0
		for _, n := range saNames {
			if n == *ep.ServiceAccount {
				matched = true
				break
			}
		}
		if !matched {
			return MatchTypeFalse
		}

		if saNamesPerNamespace == nil {
			// No selector specified, so at this point we must match.
			return MatchTypeTrue
		}

		// Check the matching service account names for the endpoints namespace. As soon as we find a match we can exit.
		for _, n := range saNamesPerNamespace[ep.Namespace] {
			if n == *ep.ServiceAccount {
				return MatchTypeTrue
			}
		}

		// No matching service account name found for the endpoints namespace.
		return MatchTypeFalse
	}

	// Cache the matcher for re-use.
	n.serviceAccountMatchers[key] = matcher
	return matcher
}

// namespaceData encapsulates the namespace labels and the labels for all service accounts in the namespace.
type namespaceData struct {
	labels               map[string]string
	serviceAccountLabels map[string]map[string]string
}

// getServiceAccounts gets the service accounts for this namespace that match the selector.
func (n *namespaceData) getServiceAccounts(sel selector.Selector) (out []string) {
	for n, l := range n.serviceAccountLabels {
		if sel.Evaluate(l) {
			out = append(out, n)
		}
	}
	return
}

// setNamespaceLabels sets the labels for a specific Namespace.
func (n *NamespaceHandler) setNamespaceLabels(name string, labels map[string]string) {
	n.get(name).labels = labels
}

// setServiceAccountLabels sets the labels for a specific ServiceAccount
func (n *NamespaceHandler) setServiceAccountLabels(namespace, name string, labels map[string]string) {
	n.get(namespace).serviceAccountLabels[name] = labels
}

// get returns the cached namespace data for the specified namespace, creating the entry if it does not exist.
func (n *NamespaceHandler) get(name string) *namespaceData {
	ns := n.namespaces[name]
	if ns == nil {
		log.WithField("namespace", name).Debug("Creating new namespace entry")
		ns = &namespaceData{
			serviceAccountLabels: make(map[string]map[string]string),
		}
		n.namespaces[name] = ns
	}
	return ns
}
