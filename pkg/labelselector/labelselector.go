// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package labelselector

import (
	log "github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/projectcalico/felix/labelindex"
	"github.com/projectcalico/libcalico-go/lib/selector"

	"github.com/tigera/compliance/pkg/resources"
)

// This is just a wrapper around the Felix InheritIndex helper, but uses ResourceID selector and label identifiers and
// provides automatic fanout based on registered listeners.
//
// This helper manages the links between selectors and labels. Callers register selectors and labels associated with a
// specific resource and thie helper calls with match start and stop events between the linked and unlinked selector and
// labels.

type Interface interface {
	RegisterCallbacks(kinds []schema.GroupVersionKind, started MatchStarted, stopped MatchStopped)

	UpdateLabels(res resources.ResourceID, labels map[string]string, parentIDs []string)
	DeleteLabels(res resources.ResourceID)
	UpdateParentLabels(id string, labels map[string]string)
	DeleteParentLabels(id string)
	UpdateSelector(res resources.ResourceID, selector string)
	DeleteSelector(res resources.ResourceID)
}

type MatchStarted func(selector, labels resources.ResourceID)
type MatchStopped func(selector, labels resources.ResourceID)

func NewLabelSelection() Interface {
	ls := &labelSelection{}
	ls.index = labelindex.NewInheritIndex(ls.onMatchStarted, ls.onMatchStopped)
	return ls
}

type labelSelection struct {
	// InheritIndex helper.  This is used to track correlations between endpoints and
	// registered selectors.
	index *labelindex.InheritIndex

	// Callbacks.
	cbs []callbacksWithKind
}

type callbacksWithKind struct {
	started MatchStarted
	stopped MatchStopped
	kind    schema.GroupVersionKind
}

func (ls *labelSelection) RegisterCallbacks(kinds []schema.GroupVersionKind, started MatchStarted, stopped MatchStopped) {
	for _, kind := range kinds {
		ls.cbs = append(ls.cbs, callbacksWithKind{
			started: started,
			stopped: stopped,
			kind:    kind,
		})
	}
}

func (ls *labelSelection) UpdateLabels(res resources.ResourceID, labels map[string]string, parentIDs []string) {
	ls.index.UpdateLabels(res, labels, parentIDs)
}

func (ls *labelSelection) DeleteLabels(res resources.ResourceID) {
	ls.index.DeleteLabels(res)
}

func (ls *labelSelection) UpdateParentLabels(parentID string, labels map[string]string) {
	ls.index.UpdateParentLabels(parentID, labels)
}

func (ls *labelSelection) DeleteParentLabels(parentID string) {
	ls.index.DeleteParentLabels(parentID)
}

func (ls *labelSelection) UpdateSelector(res resources.ResourceID, sel string) {
	parsedSel, err := selector.Parse(sel)
	if err != nil {
		// The selector is bad, remove the associated resource from the helper.
		log.WithError(err).Errorf("Bad selector found in config, removing from cache: %s", sel)
		ls.index.DeleteSelector(res)
		return
	}
	ls.index.UpdateSelector(res, parsedSel)
}

func (ls *labelSelection) DeleteSelector(res resources.ResourceID) {
	ls.index.DeleteSelector(res)
}

// onMatchStarted is called from the InheritIndex helper when a selector-endpoint match has
// started.
func (c *labelSelection) onMatchStarted(selId, labelsId interface{}) {
	selRes := selId.(resources.ResourceID)
	labelsRes := labelsId.(resources.ResourceID)

	for i := range c.cbs {
		if c.cbs[i].kind == selRes.GroupVersionKind || c.cbs[i].kind == labelsRes.GroupVersionKind {
			c.cbs[i].started(selRes, labelsRes)
		}
	}
}

// onMatchStopped is called from the InheritIndex helper when a selector-endpoint match has
// stopped.
func (c *labelSelection) onMatchStopped(selId, labelsId interface{}) {
	selRes := selId.(resources.ResourceID)
	labelsRes := labelsId.(resources.ResourceID)

	for i := range c.cbs {
		if c.cbs[i].kind == selRes.GroupVersionKind || c.cbs[i].kind == labelsRes.GroupVersionKind {
			c.cbs[i].stopped(selRes, labelsRes)
		}
	}
}
