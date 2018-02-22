// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package labelhandler

import (
	"fmt"

	"github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/felix/labelindex"
	"github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/libcalico-go/lib/selector"
	"github.com/tigera/calicoq/web/pkg/querycache/dispatcherv1v3"
)

type LabelHandler interface {
	OnUpdate(update dispatcherv1v3.Update)
	QueryEndpoints(selector string) ([]model.Key, error)
	QuerySelectors(labels map[string]string, profiles []string) ([]SelectorID, error)
	RegisterHandler(callback MatchFn)
}

type SelectorID interface {
	Policy() model.Key
	IsRule() bool
	Index() int
	IsIngress() bool
	IsSource() bool
	IsNegated() bool
}

type MatchType string

const (
	MatchStarted MatchType = "started"
	MatchStopped MatchType = "stopped"
)

type MatchFn func(matchType MatchType, selectorId SelectorID, endpoint model.Key)

func NewLabelHandler() LabelHandler {
	cq := &labelHandler{
		rules: make(map[model.Key]rules, 0),
	}
	cq.index = labelindex.NewInheritIndex(cq.onMatchStarted, cq.onMatchStopped)
	return cq
}

type labelHandler struct {
	// InheritIndex helper.  This is used to track correlations between endpoints and
	// registered selectors.
	index *labelindex.InheritIndex

	// Callbacks.
	callbacks []MatchFn

	// The accumulated matches added during a query.  If there is no query in progress
	// this is nil.
	endpoints []model.Key
	selectors []SelectorID

	// The number of rules in each policy (used so that we can delete selectors when the number
	// of rules decreases.
	rules map[model.Key]rules
}

type rules struct {
	ingress int
	egress  int
}

var zerorules = rules{}

// A queryId is used to identify either a selector or endpoint that have been injected in to
// the InheritIndex helper when running a query. Match callbacks with these identifiers can be
// ignored for our cache, but will be included in the query responses.
type queryId uuid.UUID

// Selector ID used for a policy rule.
type selectorId struct {
	policy model.Key
	flags  ruleFlag
	index  int
}
type ruleFlag byte

const (
	selectorFlagRule ruleFlag = 1 << iota
	selectorFlagIngress
	selectorFlagSrc
	selectorFlagNegated
)

func (s selectorId) Policy() model.Key {
	return s.policy
}
func (s selectorId) IsRule() bool {
	return s.flags&selectorFlagRule != 0
}
func (s selectorId) Index() int {
	return s.index
}
func (s selectorId) IsIngress() bool {
	return s.flags&selectorFlagIngress != 0
}
func (s selectorId) IsSource() bool {
	return s.flags&selectorFlagSrc != 0
}
func (s selectorId) IsNegated() bool {
	return s.flags&selectorFlagNegated != 0
}

// OnUpdate handler
func (c *labelHandler) OnUpdate(update dispatcherv1v3.Update) {
	uv3 := update.UpdateV3
	rk, ok := uv3.Key.(model.ResourceKey)
	if !ok {
		log.WithField("key", uv3.Key).Error("Unexpected resource in event type")
	}
	switch rk.Kind {
	case v3.KindProfile:
		c.onUpdateProfile(update)
	case v3.KindGlobalNetworkPolicy:
		c.onUpdatePolicy(update)
	case v3.KindNetworkPolicy:
		c.onUpdatePolicy(update)
	case v3.KindWorkloadEndpoint:
		c.onUpdateWorkloadEndpoint(update)
	case v3.KindHostEndpoint:
		c.onUpdateHostEndpoint(update)
	default:
		log.WithField("key", uv3.Key).Error("Unexpected resource in event type")
	}
}

func (c *labelHandler) RegisterHandler(callback MatchFn) {
	c.callbacks = append(c.callbacks, callback)
}

// QueryEndpoints returns a list of endpoint keys that match the supplied
// selector.
func (c *labelHandler) QueryEndpoints(selectorExpression string) ([]model.Key, error) {
	// Start by adding the query selector to the required list of selectors.
	selectorId := queryId(uuid.NewV4())
	if err := c.registerSelector(selectorId, selectorExpression); err != nil {
		return nil, err
	}

	// The register selector call will result in synchronous matchStarted callbacks to update our
	// endpoint matches. Thus our endpoints slice should now have the results we need.  All of
	// the updates will be for this specific selector since we only run a single query at a time.
	results := c.endpoints
	c.endpoints = nil

	// Remove the query selector so that we are no longer tracking it.
	c.unregisterSelector(selectorId)
	return results, nil
}

// QuerySelectors returns a list of SelectorIDs that match the supplied
// selector.
//TODO (rlb):  I think the handling of rules is wrong now that I think about it.  The rules
// will need to be handled by an active rules calculator, otherwise we won't include the rules
// from the profile.
func (c *labelHandler) QuerySelectors(labels map[string]string, profiles []string) ([]SelectorID, error) {
	// Add a fake endpoint with the requested labels and profiles.
	endpointId := queryId(uuid.NewV4())
	c.index.UpdateLabels(endpointId, labels, profiles)

	// The addition of the endpoint will result in synchronous callbacks to update our matches.  Thus
	// our match map should now have the results we need.  All of the updates will be for this specific
	// endpoint.
	results := c.selectors
	c.selectors = nil

	// Remove the fake endpoint so that we are no longer tracking it.
	c.index.DeleteLabels(endpointId)
	return results, nil
}

// onUpdateWorkloadEndpoints is called when the syncer has an update for a WorkloadEndpoint.
// This updates the InheritIndex helper and tracks global counts.
func (c *labelHandler) onUpdateWorkloadEndpoint(update dispatcherv1v3.Update) {
	uv1 := update.UpdateV1
	uv3 := update.UpdateV3
	key := uv3.Key.(model.ResourceKey)
	if uv3.UpdateType == api.UpdateTypeKVDeleted {
		c.index.DeleteLabels(key)
		return
	}
	value := uv1.Value.(*model.WorkloadEndpoint)
	c.index.UpdateLabels(uv3.Key, value.Labels, value.ProfileIDs)
}

// onUpdateHostEndpoints is called when the syncer has an update for a WorkloadEndpoint.
// This updates the InheritIndex helper and tracks global counts.
func (c *labelHandler) onUpdateHostEndpoint(update dispatcherv1v3.Update) {
	uv3 := update.UpdateV3
	key := uv3.Key.(model.ResourceKey)
	if uv3.UpdateType == api.UpdateTypeKVDeleted {
		c.index.DeleteLabels(key)
		return
	}
	value := uv3.Value.(*v3.HostEndpoint)
	c.index.UpdateLabels(uv3.Key, value.GetObjectMeta().GetLabels(), value.Spec.Profiles)
}

// onUpdateProfile is called when the syncer has an update for a Profile.
// This updates the InheritIndex helper tQ1o in turn update any endpoint labels that are
// inherited from the profile.
func (c *labelHandler) onUpdateProfile(update dispatcherv1v3.Update) {
	uv3 := update.UpdateV3
	key := uv3.Key.(model.ResourceKey)
	if uv3.UpdateType == api.UpdateTypeKVDeleted {
		c.index.DeleteParentLabels(key.Name)
		return
	}

	value := uv3.Value.(*v3.Profile)
	c.index.UpdateParentLabels(key.Name, value.Spec.LabelsToApply)
}

// onUpdatePolicy is called when the syncer has an update for a Policy.
// This is used to register/unregister match updates from the InheritIndex helper for the
// policy selector so that we can track total endpoint counts for each policy.
func (c *labelHandler) onUpdatePolicy(update dispatcherv1v3.Update) {
	uv3 := update.UpdateV3
	uv1 := update.UpdateV1

	key := uv3.Key.(model.ResourceKey)
	rulesNow, exists := c.rules[key]

	if uv3.UpdateType == api.UpdateTypeKVDeleted {
		c.onDeletePolicy(key, rulesNow)
		return
	}

	// Create the selectors in advance, so we can handle errors in the selectors prior to making
	// any updates.
	var err error
	policyV1 := uv1.Value.(*model.Policy)
	rulesAfter := rules{
		egress:  len(policyV1.OutboundRules),
		ingress: len(policyV1.InboundRules),
	}
	selectorStrings := map[SelectorID]string{
		selectorId{policy: key}: policyV1.Selector,
	}
	for i := 0; i < rulesAfter.egress; i++ {
		selectorStrings[selectorId{
			policy: key,
			index:  i,
			flags:  selectorFlagRule | selectorFlagSrc | selectorFlagNegated,
		}] = policyV1.OutboundRules[i].NotSrcSelector
		selectorStrings[selectorId{
			policy: key,
			index:  i,
			flags:  selectorFlagRule | selectorFlagSrc,
		}] = policyV1.OutboundRules[i].SrcSelector
		selectorStrings[selectorId{
			policy: key,
			index:  i,
			flags:  selectorFlagRule | selectorFlagNegated,
		}] = policyV1.OutboundRules[i].NotDstSelector
		selectorStrings[selectorId{
			policy: key,
			index:  i,
			flags:  selectorFlagRule,
		}] = policyV1.OutboundRules[i].DstSelector
	}
	for i := 0; i < rulesAfter.ingress; i++ {
		selectorStrings[selectorId{
			policy: key,
			index:  i,
			flags:  selectorFlagRule | selectorFlagIngress | selectorFlagSrc | selectorFlagNegated,
		}] = policyV1.InboundRules[i].NotSrcSelector
		selectorStrings[selectorId{
			policy: key,
			index:  i,
			flags:  selectorFlagRule | selectorFlagIngress | selectorFlagSrc,
		}] = policyV1.InboundRules[i].SrcSelector
		selectorStrings[selectorId{
			policy: key,
			index:  i,
			flags:  selectorFlagRule | selectorFlagIngress | selectorFlagNegated,
		}] = policyV1.InboundRules[i].NotDstSelector
		selectorStrings[selectorId{
			policy: key,
			index:  i,
			flags:  selectorFlagRule | selectorFlagIngress,
		}] = policyV1.InboundRules[i].DstSelector
	}

	selectors := make(map[SelectorID]selector.Selector, len(selectorStrings))
	for k, v := range selectorStrings {
		selectors[k], err = selector.Parse(v)
		if err != nil {
			// We have found a bad policy selector in our cache, so we'd better remove it. Send
			// in a delete update.
			log.WithError(err).Error("Bad policy selector found in config - removing policy from cache")
			c.onDeletePolicy(key, rulesNow)
			return
		}
	}

	// Start by deleting any of the selectors associated with rules that are no
	// longer valid.
	if exists {
		c.onDeletePolicyRules(key, rulesNow, rulesAfter)
	}
	for k, v := range selectors {
		c.index.UpdateSelector(k, v)
	}
	c.rules[key] = rulesAfter
}

// onUpdatePolicy is called when the syncer has an update for a Policy.
// This is used to register/unregister match updates from the InheritIndex helper for the
// policy selector so that we can track total endpoint counts for each policy.
func (c *labelHandler) onDeletePolicy(key model.ResourceKey, rules rules) {
	// Unregister the main policy selector and the selector for each rule in that policy.
	c.unregisterSelector(selectorId{
		policy: key,
	})
	c.onDeletePolicyRules(key, rules, zerorules)
	delete(c.rules, key)
}

// onUpdatePolicy is called when the syncer has an update for a Policy.
// This is used to register/unregister match updates from the InheritIndex helper for the
// policy selector so that we can track total endpoint counts for each policy.
func (c *labelHandler) onDeletePolicyRules(key model.ResourceKey, rulesNow, rulesAfter rules) {
	for i := rulesAfter.egress; i < rulesNow.egress; i++ {
		c.index.DeleteSelector(selectorId{
			policy: key,
			index:  i,
			flags:  selectorFlagRule | selectorFlagSrc | selectorFlagNegated,
		})
		c.index.DeleteSelector(selectorId{
			policy: key,
			index:  i,
			flags:  selectorFlagRule | selectorFlagSrc,
		})
		c.index.DeleteSelector(selectorId{
			policy: key,
			index:  i,
			flags:  selectorFlagRule | selectorFlagNegated,
		})
		c.index.DeleteSelector(selectorId{
			policy: key,
			index:  i,
			flags:  selectorFlagRule,
		})
	}
	for i := rulesAfter.ingress; i < rulesNow.ingress; i++ {
		c.index.DeleteSelector(selectorId{
			policy: key,
			index:  i,
			flags:  selectorFlagRule | selectorFlagIngress | selectorFlagSrc | selectorFlagNegated,
		})
		c.index.DeleteSelector(selectorId{
			policy: key,
			index:  i,
			flags:  selectorFlagRule | selectorFlagIngress | selectorFlagSrc,
		})
		c.index.DeleteSelector(selectorId{
			policy: key,
			index:  i,
			flags:  selectorFlagRule | selectorFlagIngress | selectorFlagNegated,
		})
		c.index.DeleteSelector(selectorId{
			policy: key,
			index:  i,
			flags:  selectorFlagRule | selectorFlagIngress,
		})
	}
	delete(c.rules, key)
}

// registerSelector registers a selector with the InheritIndex helper.
func (c *labelHandler) registerSelector(selectorId interface{}, selectorExpression string) error {
	parsedSel, err := selector.Parse(selectorExpression)
	if err != nil {
		fmt.Printf("Invalid selector: %#v. %v.\n", selectorExpression, err)
		return err
	}

	c.index.UpdateSelector(selectorId, parsedSel)
	return nil
}

// unregisterSelector unregisters a selector with the InheritIndex helper.
func (c *labelHandler) unregisterSelector(selectorId interface{}) {
	c.index.DeleteSelector(selectorId)
}

// onMatchStarted is called from the InheritIndex helper when a selector-endpoint match has
// started.
func (c *labelHandler) onMatchStarted(selId, epId interface{}) {
	switch s := selId.(type) {
	case queryId:
		c.endpoints = append(c.endpoints, epId.(model.Key))
	case SelectorID:
		switch e := epId.(type) {
		case model.Key:
			for _, cb := range c.callbacks {
				cb(MatchStarted, s, e)
			}
		case queryId:
			c.selectors = append(c.selectors, s)
		default:
			log.WithFields(log.Fields{
				"selId": selId,
				"epId":  epId,
			}).Fatal("Unhandled endpoint type in onMatchStarted event")
		}
	default:
		log.WithFields(log.Fields{
			"selId": selId,
			"epId":  epId,
		}).Fatal("Unhandled selector type in onMatchStarted event")
	}
}

// onMatchStopped is called from the InheritIndex helper when a selector-endpoint match has
// stopped.
func (c *labelHandler) onMatchStopped(selId, epId interface{}) {
	switch s := selId.(type) {
	case queryId:
		// noop required - this occurs when the query is deleted.
	case SelectorID:
		switch e := epId.(type) {
		case model.Key:
			for _, cb := range c.callbacks {
				cb(MatchStopped, s, e)
			}
		case queryId:
			// noop required - this occurs when the query is deleted.
		default:
			log.WithFields(log.Fields{
				"selId": selId,
				"epId":  epId,
			}).Fatal("Unhandled endpoint type in onMatchStopped event")
		}
	default:
		log.WithFields(log.Fields{
			"selId": selId,
			"epId":  epId,
		}).Fatal("Unhandled selector type in onMatchStopped event")
	}
}
