package xrefcache

import (
	"net"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/felix/labelindex"

	"github.com/projectcalico/libcalico-go/lib/selector"

	"github.com/tigera/compliance/pkg/resources"
	"github.com/tigera/compliance/pkg/syncer"
)

// This file implements an IPManager. This acts as a bridge between the endpoints that are configured with one or more
// IP addresses and other resources that reference these IP addresses. This may also be used as a cache to determine the
// owner of an IP address.

// Callbacks for match start/stop between an IP address owner and an IP address client.
type IPMatchStarted func(owner, client resources.ResourceID)
type IPMatchStopped func(owner, client resources.ResourceID)

// IPManager interface.
type IPManager interface {
	RegisterCallbacks(onMatchStarted IPMatchStarted, onMatchStopped IPMatchStopped)
	SetOwnerIPs(owner resources.ResourceID, ips []string)
	SetClientIPs(client resources.ResourceID, ips []string)
}

// NewNetworkPolicyRuleSelectorManager creates a new NetworkPolicyRuleSelectorManager.
func NewIPManager(onUpdate func(update syncer.Update)) IPManager {
	ipm := &ipManager{
		ownersByIP: make(map[resources.ResourceID]resources.Set),
		ipsByOwner: make(map[resources.ResourceID]resources.Set),
		usersByIP:  make(map[resources.ResourceID]resources.Set),
		ipsByUser:  make(map[resources.ResourceID]resources.Set),
	}
	ipm.index = labelindex.NewInheritIndex(ipm.matchStarted, ipm.matchStopped)
	return ipm
}

// networkPolicyRuleSelectorManager implements the NetworkPolicyRuleSelectorManager interface.
type ipManager struct {
	// Felix index handler.
	index *labelindex.InheritIndex

	// Registered match stopped/started events.
	onMatchStarted []IPMatchStarted
	onMatchStopped []IPMatchStopped

	ownersByIP map[resources.ResourceID]resources.Set
	ipsByOwner map[resources.ResourceID]resources.Set
	usersByIP  map[resources.ResourceID]resources.Set
	ipsByUser  map[resources.ResourceID]resources.Set
}

// RegisterCallbacks registers match start/stop callbacks with this manager.
func (m *ipManager) RegisterCallbacks(onMatchStarted IPMatchStarted, onMatchStopped IPMatchStopped) {
	m.onMatchStarted = append(m.onMatchStarted, onMatchStarted)
	m.onMatchStopped = append(m.onMatchStopped, onMatchStopped)
}

// SetPolicyRuleSelectors sets the rule selectors that need to be tracked by a policy resource.
func (m *ipManager) SetOwnerIPs(owner resources.ResourceID, ips []string) {
	if len(ips) == 0 {
		m.index.DeleteLabels(owner)
		return
	}

	l := map[string]string{}
	for _, ipStr := range ips {
		ipStr := normalizeIP(ipStr)
		l[ipStr] = ""
	}
	m.index.UpdateLabels(owner, l, nil)
}

// SetPolicyRuleSelectors sets the rule selectors that need to be tracked by a policy resource.
func (m *ipManager) SetClientIPs(client resources.ResourceID, ips []string) {
	if len(ips) == 0 {
		m.index.DeleteSelector(client)
	}
	sels := []string{}
	for _, ip := range ips {
		sels = append(sels, "has("+normalizeIP(ip)+")")
	}
	sel := strings.Join(sels, " && ")
	parsedSel, err := selector.Parse(sel)
	if err != nil {
		// The selector is bad, remove the associated resource from the helper.
		log.WithError(err).Errorf("Bad selector constructed for IP management, removing from cache: %s", sel)
		m.index.DeleteSelector(client)
		return
	}
	m.index.UpdateSelector(client, parsedSel)
}

// onMatchStarted is called from the InheritIndex helper when a selector-endpoint match has
// started.
func (c *ipManager) matchStarted(client, owner interface{}) {
	ownerId := owner.(resources.ResourceID)
	clientId := client.(resources.ResourceID)
	for i := range c.onMatchStarted {
		c.onMatchStarted[i](ownerId, clientId)
	}
}

// onMatchStopped is called from the InheritIndex helper when a selector-endpoint match has
// stopped.
func (c *ipManager) matchStopped(client, owner interface{}) {
	ownerId := owner.(resources.ResourceID)
	clientId := client.(resources.ResourceID)
	for i := range c.onMatchStopped {
		c.onMatchStopped[i](ownerId, clientId)
	}
}

// normalizeIP ensures the string definition of an IP is always the same irrespective of what format it was provided.
func normalizeIP(ip string) string {
	return net.ParseIP(ip).To16().String()
}
