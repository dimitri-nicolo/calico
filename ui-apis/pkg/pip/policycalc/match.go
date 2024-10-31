package policycalc

import (
	log "github.com/sirupsen/logrus"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"github.com/tigera/api/pkg/lib/numorstring"

	"github.com/projectcalico/calico/libcalico-go/lib/net"
	"github.com/projectcalico/calico/lma/pkg/api"
	pipcfg "github.com/projectcalico/calico/ui-apis/pkg/pip/config"
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

type FlowMatcher func(*api.Flow, *flowCache) MatchType
type EndpointMatcher func(*api.Flow, *api.FlowEndpointData, *flowCache, *endpointCache) MatchType

// NewMatcherFactory creates a new MatcherFactory
func NewMatcherFactory(
	cfg *pipcfg.Config,
	namespaces *NamespaceHandler,
	selectors *EndpointSelectorHandler,
) *MatcherFactory {
	return &MatcherFactory{
		cfg:        cfg,
		namespaces: namespaces,
		selectors:  selectors,
	}
}

// MatcherFactory is used to create Flow and Endpoint matchers for use in the compiled policies.
type MatcherFactory struct {
	cfg        *pipcfg.Config
	namespaces *NamespaceHandler
	selectors  *EndpointSelectorHandler
}

// Not returns a negated flow matcher.
func (m *MatcherFactory) Not(fm FlowMatcher) FlowMatcher {
	if fm == nil {
		return nil
	}

	// Return a closure that invokes the matcher and negates the response.
	return func(flow *api.Flow, flowCache *flowCache) MatchType {
		mt := fm(flow, flowCache)
		switch mt {
		case MatchTypeTrue:
			mt = MatchTypeFalse
			log.Debugf("Invert match %s -> %s", MatchTypeTrue, mt)
		case MatchTypeFalse:
			mt = MatchTypeTrue
			log.Debugf("Invert match %s -> %s", MatchTypeFalse, mt)
		}

		return mt
	}
}

// IPVersion matcher
func (m *MatcherFactory) IPVersion(version *int) FlowMatcher {
	if version == nil {
		return nil
	}

	// Create a closure that checks the IP version against the version of the IPs in the flow.
	return func(flow *api.Flow, flowCache *flowCache) MatchType {
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
	return func(flow *api.Flow, flowCache *flowCache) MatchType {
		if flow.Proto == nil {
			log.Debugf("ICMP: %s (protocol unknown)", MatchTypeUncertain)
			return MatchTypeUncertain
		}
		switch *flow.Proto {
		case api.ProtoICMP, api.ProtoICMPv6:
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
	return func(flow *api.Flow, flowCache *flowCache) MatchType {
		log.Debugf("HTTP: %s (unknown)", MatchTypeUncertain)
		return MatchTypeUncertain
	}
}

// Protocol matcher
func (m *MatcherFactory) Protocol(p *numorstring.Protocol) FlowMatcher {
	protocol := api.GetProtocolNumber(p)
	if protocol == nil {
		return nil
	}

	return func(flow *api.Flow, flowCache *flowCache) MatchType {
		if flow.Proto == nil {
			log.Debugf("Proto: %s (unknown)", MatchTypeUncertain)
			return MatchTypeUncertain
		}
		if *flow.Proto == *protocol {
			log.Debugf("Proto: %s (protocol matches %d)", MatchTypeTrue, *protocol)
			return MatchTypeTrue
		}
		log.Debugf("Proto: %s (protocol %d != %d)", MatchTypeFalse, *flow.Proto, *protocol)
		return MatchTypeFalse
	}
}

// Src creates a FlowMatcher from an EndpointMatcher - it will invoke the EndpointMatcher using the flow source.
func (m *MatcherFactory) Src(em EndpointMatcher) FlowMatcher {
	if em == nil {
		return nil
	}

	return func(flow *api.Flow, flowCache *flowCache) MatchType {
		log.Debug("Source match")
		return em(flow, &flow.Source, flowCache, &flowCache.source)
	}
}

// Dst creates a FlowMatcher from an EndpointMatcher - it will invoke the EndpointMatcher using the flow dest.
func (m *MatcherFactory) Dst(em EndpointMatcher) FlowMatcher {
	if em == nil {
		return nil
	}

	return func(flow *api.Flow, flowCache *flowCache) MatchType {
		log.Debug("Destination match")
		return em(flow, &flow.Destination, flowCache, &flowCache.destination)
	}
}

// CalicoEndpointSelector endpoints matcher
func (m *MatcherFactory) CalicoEndpointSelector() EndpointMatcher {
	return func(_ *api.Flow, ed *api.FlowEndpointData, _ *flowCache, _ *endpointCache) MatchType {
		if ed.IsCalicoManagedEndpoint() {
			return MatchTypeTrue
		}
		return MatchTypeFalse
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
	return func(_ *api.Flow, ed *api.FlowEndpointData, _ *flowCache, _ *endpointCache) MatchType {
		if len(ed.IPs) == 0 {
			// Endpoint IP is unknown. If this is a Calico endpoint then we either have a negative match or an uncertain
			// match depending on configuration.
			if ed.IsCalicoManagedEndpoint() && !m.cfg.CalicoEndpointNetMatchAlways {
				log.Debugf("Nets: %s (unknown, but assume Calico endpoint matchers use label selectors only)", MatchTypeFalse)
				return MatchTypeFalse
			}
			log.Debugf("Nets: %s (unknown)", MatchTypeUncertain)
			return MatchTypeUncertain
		}

		for i := range nets {
			matchedIPs := 0
			for _, ip := range ed.IPs {
				if cnets[i].Contains(ip.IP) {
					log.Debugf("Nets: (ip matches %s)", ip)
					matchedIPs++
				} else {
					log.Debugf("Nets: (ip not matches %s)", ip)
				}
			}

			if matchedIPs == len(ed.IPs) {
				log.Debugf("Nets: %s (All IPs match)", MatchTypeTrue)
				return MatchTypeTrue
			}
		}

		log.Debugf("Nets: %s (IP does not match %s)", MatchTypeFalse, ed.IPs)
		return MatchTypeFalse
	}
}

func (m *MatcherFactory) NotNets(nets []string) EndpointMatcher {
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
	return func(_ *api.Flow, ed *api.FlowEndpointData, _ *flowCache, _ *endpointCache) MatchType {
		if len(ed.IPs) == 0 {
			// Endpoint IP is unknown. If this is a Calico endpoint then we either have a negative match or an uncertain
			// match depending on configuration.
			if ed.IsCalicoManagedEndpoint() && !m.cfg.CalicoEndpointNetMatchAlways {
				log.Debugf("Nets: %s (unknown, but assume Calico endpoint matchers use label selectors only)", MatchTypeFalse)
				return MatchTypeTrue
			}
			log.Debugf("Nets: %s (unknown)", MatchTypeUncertain)
			return MatchTypeUncertain
		}

		for i := range cnets {
			matchedIPs := 0
			for _, ip := range ed.IPs {
				if !cnets[i].Contains(ip.IP) {
					log.Debugf("Nets: (ip not matches %s)", ip)
					matchedIPs++
				} else {
					log.Debugf("Nets: (ip matches %s)", ip)
				}
			}

			if matchedIPs == len(ed.IPs) {
				log.Debugf("Nets: %s (All IPs do not match)", MatchTypeTrue)
				return MatchTypeTrue
			}
		}

		log.Debugf("Nets: %s (IP does not match %s)", MatchTypeFalse, ed.IPs)
		return MatchTypeFalse
	}
}

// Ports Endpoint matcher
func (m *MatcherFactory) Ports(ports []numorstring.Port) EndpointMatcher {
	if len(ports) == 0 {
		return nil
	}

	// Create a closure matching on the numerical port values.
	log.Debug("Ports matcher")
	return func(f *api.Flow, ed *api.FlowEndpointData, _ *flowCache, _ *endpointCache) MatchType {
		// If the port is not specified in the endpoint data then return uncertain.
		if ed.Port == nil {
			log.Debugf("Ports: %s (unknown)", MatchTypeUncertain)
			return MatchTypeUncertain
		}

		worstMatch := MatchTypeFalse
		for _, port := range ports {
			if port.PortName != "" {
				// Check against a named port.
				if ed.NamedPorts == nil {
					// We don't have named port information for this endpoint. In that case we cannot perform a
					// definitive match - in this case the worst possible match is uncertain rather than false.
					log.Debugf("Ports: (worst case %s) named port unknown for endpoint", MatchTypeUncertain)
					worstMatch = MatchTypeUncertain
				}
				// We have named ports, so we can attempt a lookup.
				for _, np := range ed.NamedPorts {
					if np.Name == port.PortName {
						if f.Proto == nil {
							// We found a port with a matching name, but we don't have protocol information to perform
							// a definitive match - in this case the worst possible match is uncertain rather than
							// false.
							log.Debugf("Ports: (worst case %s) named port protocol is not known", MatchTypeUncertain)
							worstMatch = MatchTypeUncertain
						} else {

							if *f.Proto == np.Protocol && np.Port == *ed.Port {
								// We found a port with a matching name and protocol and the corresponding port matches the
								// port in the flow.
								log.Debugf("Ports: %s (named port, protocol match; numerical port matches)", MatchTypeTrue)
								return MatchTypeTrue
							}
						}
					}
				}
			} else {
				// Check against a numerical port.
				if port.MinPort <= *ed.Port && port.MaxPort >= *ed.Port {
					log.Debugf("Ports: %s (numerical port %d matched)", MatchTypeTrue, *ed.Port)
					return MatchTypeTrue
				}
			}
		}

		// We didn't match, return the worst match value. This will be False if all ports were numerical, or Uncertain
		// if one or more ports were named and we do not have named port information.
		log.Debugf("Ports: %s (numerical port %d not matched)", worstMatch, *ed.Port)
		return worstMatch
	}
}

// Domains Endpoint matcher
func (m *MatcherFactory) Domains(domains []string) EndpointMatcher {
	if len(domains) == 0 {
		return nil
	}
	return func(_ *api.Flow, ed *api.FlowEndpointData, _ *flowCache, _ *endpointCache) MatchType {
		log.Debugf("Domains: %s (unknown)", MatchTypeUncertain)
		return MatchTypeUncertain
	}
}

// Service matcher
func (m *MatcherFactory) ServiceSelector(match *v3.ServiceMatch) EndpointMatcher {
	if match == nil || match.Name == "" || match.Namespace == "" {
		return nil
	}
	return func(_ *api.Flow, ed *api.FlowEndpointData, _ *flowCache, _ *endpointCache) MatchType {
		log.Debugf("Service matcher: %s (unknown)", MatchTypeUncertain)
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
	return func(_ *api.Flow, ed *api.FlowEndpointData, _ *flowCache, _ *endpointCache) MatchType {
		if ed.Namespace == namespace {
			log.Debugf("Namespace: %s (name matches %s)", MatchTypeTrue, ed.Namespace)
			return MatchTypeTrue
		}
		log.Debugf("Namespace: %s (name %s != %s)", MatchTypeFalse, ed.Namespace, namespace)
		return MatchTypeFalse
	}
}
