package policycalc

import (
	log "github.com/sirupsen/logrus"

	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/net"
	"github.com/projectcalico/libcalico-go/lib/numorstring"
)

type MatchType byte

const (
	MatchTypeUnknown MatchType = iota
	MatchTypeTrue
	MatchTypeFalse
	MatchTypeUncertain
)

func (m MatchType) String() string {
	switch m {
	case MatchTypeTrue:
		return "MatchTrue"
	case MatchTypeFalse:
		return "MatchFalse"
	case MatchTypeUncertain:
		return "MatchUncertain"
	default:
		return "-"
	}
}

type FlowMatcher func(*Flow) MatchType
type EndpointMatcher func(*FlowEndpointData) MatchType

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

	// Return a closure that invokes the matcher and negates the response.
	return func(flow *Flow) MatchType {
		mt := fm(flow)
		switch mt {
		case MatchTypeTrue:
			mt = MatchTypeFalse
		case MatchTypeFalse:
			mt = MatchTypeTrue
		}
		log.Debugf("Invert match %s -> %s", mt, mt)
		return mt
	}
}

// IPVersion matcher
func (m *MatcherFactory) IPVersion(version *int) FlowMatcher {
	if version == nil {
		return nil
	}

	// Create a closure that checks the IP version against the version of the IPs in the flow.
	return func(flow *Flow) MatchType {
		// Do the best match we can, by checking actual IPs if present.
		if flow.IPVersion == nil {
			log.Debugf("IPVersion: %s (unknown)", MatchTypeUncertain)
			return MatchTypeUncertain
		}

		if *flow.IPVersion == *version {
			log.Debugf("IPVersion: %s (version matches %d)", MatchTypeTrue, *version)
			return MatchTypeTrue
		}
		log.Debugf("IPVersion: %s (version %d != %d)", MatchTypeFalse, *flow.IPVersion, *version)
		return MatchTypeFalse
	}
}

// ICMP matcher
func (m *MatcherFactory) ICMP(icmp *v3.ICMPFields) FlowMatcher {
	if icmp == nil || (icmp.Code == nil && icmp.Type == nil) {
		// We have no actual ICMP match parameters so don't return a matcher.
		return nil
	}

	// Flows will not contain ICMP data, the best we can do is check the protocol and set match type to false if not
	// ICMP.
	return func(flow *Flow) MatchType {
		if flow.Proto == nil {
			log.Debugf("ICMP: %s (protocol unknown)", MatchTypeUncertain)
			return MatchTypeUncertain
		}
		switch *flow.Proto {
		case ProtoICMP, ProtoICMPv6:
			log.Debugf("ICMP: %s (ICMP parameters unknown)", MatchTypeUncertain)
			return MatchTypeUncertain
		default:
			log.Debugf("ICMP: %s (protocol is not ICMP or ICMPv6)", MatchTypeFalse)
			return MatchTypeFalse
		}
	}
}

// HTTP matcher
func (m *MatcherFactory) HTTP(http *v3.HTTPMatch) FlowMatcher {
	if http == nil || (len(http.Methods) == 0 && len(http.Paths) == 0) {
		// We have no actual HTTP match parameters so don't return a matcher.
		return nil
	}

	// Flows will not contain HTTP data.
	return func(flow *Flow) MatchType {
		log.Debugf("HTTP: %s (unknown)", MatchTypeUncertain)
		return MatchTypeUncertain
	}
}

// Protocol matcher
func (m *MatcherFactory) Protocol(p *numorstring.Protocol) FlowMatcher {
	protocol := GetProtocolNumber(p)
	if protocol == nil {
		return nil
	}

	return func(flow *Flow) MatchType {
		if flow.Proto == nil {
			log.Debugf("Protocol: %s (unknown)", MatchTypeUncertain)
			return MatchTypeUncertain
		}
		if *flow.Proto == *protocol {
			log.Debugf("Protocol: %s (protocol matches %d)", MatchTypeTrue, *protocol)
			return MatchTypeTrue
		}
		log.Debugf("Protocol: %s (protocol %d != %d)", MatchTypeFalse, *flow.Proto, *protocol)
		return MatchTypeFalse
	}
}

// Src creates a FlowMatcher from an EndpointMatcher - it will invoke the EndpointMatcher using the flow source.
func (m *MatcherFactory) Src(em EndpointMatcher) FlowMatcher {
	if em == nil {
		return nil
	}

	return func(flow *Flow) MatchType {
		log.Debug("Source match")
		return em(&flow.Source)
	}
}

// Dst creates a FlowMatcher from an EndpointMatcher - it will invoke the EndpointMatcher using the flow dest.
func (m *MatcherFactory) Dst(em EndpointMatcher) FlowMatcher {
	if em == nil {
		return nil
	}

	return func(flow *Flow) MatchType {
		log.Debug("Destination match")
		return em(&flow.Destination)
	}
}

// ServiceAccounts endpoints matcher
func (m *MatcherFactory) ServiceAccounts(sa *v3.ServiceAccountMatch) EndpointMatcher {
	if sa == nil {
		return nil
	}
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
	return func(ed *FlowEndpointData) MatchType {
		if ed.IP == nil {
			log.Debugf("Nets: %s (unknown)", MatchTypeUncertain)
			return MatchTypeUncertain
		}
		for i := range nets {
			if cnets[i].Contains(ed.IP.IP) {
				log.Debugf("Nets: %s (IP matches %s)", MatchTypeTrue, ed.IP)
				return MatchTypeTrue
			}
		}
		log.Debugf("Nets: %s (IP does not match %s)", MatchTypeFalse, ed.IP)
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
		return func(ep *FlowEndpointData) MatchType {
			log.Debugf("Ports: %s (named ports cannot be matched)", MatchTypeUncertain)
			return MatchTypeUncertain
		}
	}

	// Create a closure matching on the numerical port values.
	log.Debug("Ports matcher")
	return func(ed *FlowEndpointData) MatchType {
		// If the port is not specified in the endpoint data then return uncertain.
		if ed.Port == nil {
			log.Debugf("Ports: %s (unknown)", MatchTypeUncertain)
			return MatchTypeUncertain
		}

		for _, port := range numerical {
			if port.MinPort <= *ed.Port && port.MaxPort >= *ed.Port {
				log.Debugf("Ports: %s (numerical port %d matched)", MatchTypeUncertain, *ed.Port)
				return MatchTypeTrue
			}
		}

		// We didn't match, return the worst match value. This will be False if all ports were numerical, or Uncertain
		// if one or more ports were named.
		log.Debugf("Ports: %s (numerical port %d not matched)", worstMatch, *ed.Port)
		return worstMatch
	}
}

// Domains Endpoint matcher
func (m *MatcherFactory) Domains(domains []string) EndpointMatcher {
	if len(domains) == 0 {
		return nil
	}
	return func(ed *FlowEndpointData) MatchType {
		log.Debugf("Domains: %s (unknown)", MatchTypeUncertain)
		return MatchTypeUncertain
	}
}

// Selector Endpoint matcher
func (m *MatcherFactory) Selector(sel string) EndpointMatcher {
	if sel == "" {
		return nil
	}
	return m.selectors.GetSelectorEndpointMatcher(sel)
}

// NamespaceSelector Endpoint matcher
func (m *MatcherFactory) NamespaceSelector(sel string) EndpointMatcher {
	if sel == "" {
		return nil
	}
	return m.namespaces.GetNamespaceSelectorEndpointMatcher(sel)
}

// Namespace Endpoint matcher
func (m *MatcherFactory) Namespace(namespace string) EndpointMatcher {
	if namespace == "" {
		return nil
	}

	// Create a closure to match the endpoint namespace against the specified namespace.
	return func(ed *FlowEndpointData) MatchType {
		if ed.Namespace == namespace {
			log.Debugf("Namespace: %s (name matches %s)", MatchTypeTrue, ed.Namespace)
			return MatchTypeTrue
		}
		log.Debugf("Namespace: %s (name %s != %s)", MatchTypeUncertain, ed.Namespace, namespace)
		return MatchTypeFalse
	}
}
