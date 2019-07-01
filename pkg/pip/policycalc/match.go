package policycalc

import (
	"fmt"
	"reflect"

	log "github.com/sirupsen/logrus"

	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/net"
	"github.com/projectcalico/libcalico-go/lib/numorstring"
)

type MatchType byte

const (
	// Use bit-wise values for these match types - it makes it easy to negate a response by xor-ing with the
	// true and false flags.
	MatchTypeTrue MatchType = 1 << iota
	MatchTypeFalse
	MatchTypeUncertain
)

const (
	MatchTypeNone MatchType = 0
)

func (m MatchType) String() string {
	var str string
	switch m {
	case MatchTypeTrue:
		str = "True"
	case MatchTypeFalse:
		str = "False"
	case MatchTypeUncertain:
		str = "Uncertain"
	default:
		str = "None"
	}
	return fmt.Sprintf("%d(%s)", m, str)
}

type FlowMatcher func(*Flow) MatchType
type EndpointMatcher func(*FlowEndpointData) MatchType

var (
	// "Marker" functions used to short-circuit some of the processing when rules or policies contain matches that
	// we can never be performed due to limited available data.
	//
	// Note that it is not possible to compare functions in golang, but it is possible to use reflect to get the
	// value of a function. There are some subtleties regarding closures returning the same function value, but since
	// both of these "marker" functions are declared separately that does not matter here.
	FlowMatcherUncertain     FlowMatcher     = func(_ *Flow) MatchType { return MatchTypeUncertain }
	EndpointMatcherUncertain EndpointMatcher = func(_ *FlowEndpointData) MatchType { return MatchTypeUncertain }
)

// NewMatcherFactory creates a new MatcherFactory
func NewMatcherFactory(cfg *Config, namespaces *NamespaceHandler, selectors *EndpointSelectorHandler) *MatcherFactory {
	return &MatcherFactory{
		cfg:        cfg,
		namespaces: namespaces,
		selectors:  selectors,
	}
}

// MatcherFactory is used to create Flow and Endpoint matchers for use in the compiled policies.
type MatcherFactory struct {
	cfg        *Config
	namespaces *NamespaceHandler
	selectors  *EndpointSelectorHandler
}

// Not returns a negated flow matcher.
func (m *MatcherFactory) Not(fm FlowMatcher) FlowMatcher {
	if fm == nil {
		return nil
	}
	if reflect.ValueOf(fm) == reflect.ValueOf(FlowMatcherUncertain) {
		// The flow matcher is the static "uncertain flow matcher, so the negated version is the same.
		return FlowMatcherUncertain
	}

	// Return a closure that invokes the matcher and negates the response.
	log.Debug("Negated matcher")
	return func(flow *Flow) MatchType {
		return fm(flow) ^ (MatchTypeTrue | MatchTypeFalse)
	}
}

// IPVersion matcher
func (m *MatcherFactory) IPVersion(version *int) FlowMatcher {
	if version == nil {
		return nil
	}

	// Create a closure that checks the IP version against the version of the IPs in the flow.
	log.Debug("IPVersion matcher")
	return func(flow *Flow) MatchType {
		// Do the best match we can, by checking actual IPs if present.
		if flow.IPVersion == nil {
			return MatchTypeUncertain
		}

		if *flow.IPVersion == *version {
			return MatchTypeTrue
		}
		return MatchTypeFalse
	}
}

// ICMP matcher
func (m *MatcherFactory) ICMP(icmp *v3.ICMPFields) FlowMatcher {
	if icmp == nil || (icmp.Code == nil && icmp.Type == nil) {
		// We have no actual ICMP match parameters so don't return a matcher.
		return nil
	}

	// Flows will not contain ICMP data.
	log.Debug("ICMP matcher - always uncertain")
	return FlowMatcherUncertain
}

// HTTP matcher
func (m *MatcherFactory) HTTP(http *v3.HTTPMatch) FlowMatcher {
	if http == nil || (len(http.Methods) == 0 && len(http.Paths) == 0) {
		// We have no actual HTTP match parameters so don't return a matcher.
		return nil
	}

	// Flows will not contain HTTP data.
	log.Debug("HTTP matcher - always uncertain")
	return FlowMatcherUncertain
}

// Protocol matcher
func (m *MatcherFactory) Protocol(p *numorstring.Protocol) FlowMatcher {
	protocol := GetProtocolNumber(p)
	if protocol == nil {
		return nil
	}

	log.Debug("Protocol matcher")
	return func(flow *Flow) MatchType {
		if flow.Proto == nil {
			log.Debug("Protocol uncertain")
			return MatchTypeUncertain
		}
		if *flow.Proto == *protocol {
			log.Debug("Protocol matched")
			return MatchTypeTrue
		}
		log.Debug("Protocol did not match")
		return MatchTypeFalse
	}
}

// Src creates a FlowMatcher from an EndpointMatcher - it will invoke the EndpointMatcher using the flow source.
func (m *MatcherFactory) Src(em EndpointMatcher) FlowMatcher {
	if em == nil {
		return nil
	}

	if reflect.ValueOf(em) == reflect.ValueOf(EndpointMatcherUncertain) {
		// The endpoint matcher is the static "uncertain" matcher, so the flow matcher should also be uncertain.
		log.Debug("Src matcher - always uncertain")
		return FlowMatcherUncertain
	}

	log.Debug("Src matcher")
	return func(flow *Flow) MatchType {
		return em(&flow.Source)
	}
}

// Dst creates a FlowMatcher from an EndpointMatcher - it will invoke the EndpointMatcher using the flow dest.
func (m *MatcherFactory) Dst(em EndpointMatcher) FlowMatcher {
	if em == nil {
		return nil
	}

	if reflect.ValueOf(em) == reflect.ValueOf(EndpointMatcherUncertain) {
		// The endpoint matcher is the static "uncertain" matcher, so the flow matcher should also be uncertain.
		log.Debug("Dst matcher - always uncertain")
		return FlowMatcherUncertain
	}

	log.Debug("Dst matcher")
	return func(flow *Flow) MatchType {
		return em(&flow.Destination)
	}
}

// ServiceAccounts endpoints matcher
func (m *MatcherFactory) ServiceAccounts(sa *v3.ServiceAccountMatch) EndpointMatcher {
	if sa == nil {
		return nil
	}
	log.Debug("Namespace selector matcher")
	return m.namespaces.GetServiceAccountEndpointMatchers(sa)
}

// Nets Endpoint matcher
func (m *MatcherFactory) Nets(nets []string) EndpointMatcher {
	if len(nets) == 0 {
		return nil
	}

	cnets := make([]net.IPNet, len(nets))
	for i := range nets {
		_, cidr, err := net.ParseCIDROrIP(nets[i])
		if err != nil {
			return nil
		}
		cnets[i] = *cidr
	}

	// Create a closure matching on the nets.
	log.Debug("Nets matcher")
	return func(ed *FlowEndpointData) MatchType {
		if ed.IP == nil {
			return MatchTypeUncertain
		}
		for i := range nets {
			if cnets[i].Contains(ed.IP.IP) {
				return MatchTypeTrue
			}
		}
		return MatchTypeFalse
	}
}

// Ports Endpoint matcher
func (m *MatcherFactory) Ports(ports []numorstring.Port) EndpointMatcher {
	if len(ports) == 0 {
		return nil
	}

	// If there are any named ports in the configuration then remove them - in this case the worst case match will be
	// "uncertain" rather than False, because we cannot rule out a match against the named port.
	numerical := make([]numorstring.Port, 0)
	worstMatch := MatchTypeFalse
	for _, p := range ports {
		if p.PortName != "" {
			worstMatch = MatchTypeUncertain
			continue
		}
		numerical = append(numerical, p)
	}

	// If all of the ports were named then just return the EndpointMatcherUncertain marker - this provides some minor
	// shortcuts in the processing.
	if len(numerical) == 0 {
		log.Debug("Ports matcher with only named ports - always uncertain")
		return EndpointMatcherUncertain
	}

	// Create a closure matching on the numerical port values.
	log.Debug("Ports matcher")
	return func(ed *FlowEndpointData) MatchType {
		// If the port is not specified in the endpoint data then return uncertain.
		if ed.Port == nil {
			return MatchTypeUncertain
		}

		for _, port := range numerical {
			if port.MinPort <= *ed.Port && port.MaxPort >= *ed.Port {
				return MatchTypeTrue
			}
		}

		// We didn't match, return the worst match value. This will be False if all ports were numerical, or Uncertain
		// if one or more ports were named.
		return worstMatch
	}
}

// Domains Endpoint matcher
func (m *MatcherFactory) Domains(domains []string) EndpointMatcher {
	if len(domains) == 0 {
		return nil
	}
	log.Debug("Domains matcher - always uncertain")
	return func(ed *FlowEndpointData) MatchType {
		return MatchTypeUncertain
	}
}

// Selector Endpoint matcher
func (m *MatcherFactory) Selector(sel string) EndpointMatcher {
	if sel == "" {
		return nil
	}
	log.Debug("Endpoint selector matcher")
	return m.selectors.GetSelectorEndpointMatcher(sel)
}

// NamespaceSelector Endpoint matcher
func (m *MatcherFactory) NamespaceSelector(sel string) EndpointMatcher {
	if sel == "" {
		return nil
	}
	log.Debug("Namespace selector matcher")
	return m.namespaces.GetNamespaceSelectorEndpointMatcher(sel)
}

// Namespace Endpoint matcher
func (m *MatcherFactory) Namespace(namespace string) EndpointMatcher {
	if namespace == "" {
		return nil
	}

	// Create a closure to match the endpoint namespace against the specified namespace.
	log.Debug("Namespace matcher")
	return func(ed *FlowEndpointData) MatchType {
		if ed.Namespace == namespace {
			return MatchTypeTrue
		}
		return MatchTypeFalse
	}
}
