// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package xrefcache

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/projectcalico/calico/libcalico-go/lib/backend/k8s/conversion"
	"github.com/projectcalico/calico/libcalico-go/lib/selector/parser"
)

var (
	// Fake resource kind used to register an in-scope selector.
	KindInScopeSelection = metav1.TypeMeta{
		Kind:       "in-scope-selector",
		APIVersion: "internal.tigera.io/v1",
	}
	KindsInScopeSelection = []metav1.TypeMeta{KindInScopeSelection}
)

// calculateInScopeEndpointsSelector converts an EndpointsSelection into a single selector string and the appropriate
// in-scope resource ID to use.
func calculateInScopeEndpointsSelector(e *apiv3.EndpointsSelection) (apiv3.ResourceID, string, error) {
	// Start of with the endpoint selector (if specified)
	var updated string
	if e != nil && e.Selector != "" {
		updated = fmt.Sprintf("(%s)", e.Selector)
	}

	// If the namespace selector is specified then include that in our selector, ANDing the selectors together.
	if e != nil && e.Namespaces != nil {
		sel := createSelector(*e.Namespaces, conversion.NamespaceLabelPrefix, apiv3.LabelNamespace)
		if updated != "" {
			updated = fmt.Sprintf("%s && %s", updated, sel)
		} else {
			updated = sel
		}
	}

	if e != nil && e.ServiceAccounts != nil {
		sel := createSelector(*e.ServiceAccounts, conversion.ServiceAccountLabelPrefix, apiv3.LabelServiceAccount)
		if updated != "" {
			updated = fmt.Sprintf("%s && %s", updated, sel)
		} else {
			updated = sel
		}
	}

	if updated == "" {
		updated = "all()"
	}

	// Normalize it now.
	parsedSelector, err := parser.Parse(updated)
	if err != nil {
		log.WithError(err).Errorf("Failed to parse full endpoints selector: %s", updated)
		return apiv3.ResourceID{}, "", err
	}

	return apiv3.ResourceID{
		TypeMeta: KindInScopeSelection,
		Name:     updated,
	}, parsedSelector.String(), nil
}

//TODO(rlb): This code is taken from and modified to be non-SA specific. We should move this back into libcalico-go and
//TODO       ensure we use the same processing elsewhere.

// createSelector creates the selector for the supplied NamesAndLabelsMatch.
func createSelector(nal apiv3.NamesAndLabelsMatch, labelPrefix, nameLabel string) string {
	var updated string
	if nal.Selector != "" {
		updated = parseSelectorAttachPrefix(nal.Selector, labelPrefix)
	} else if len(nal.Names) == 0 {
		// No selector or names, leave selector blank.
		return updated
	}
	//TODO(rlb): The libcalico-go does not have this, but I think it should... check with Spike.
	// There is a selector, but no names. Make sure we include a check for the name label otherwise an all() operator
	// will not select only this resource type.
	if len(nal.Names) == 0 {
		return fmt.Sprintf("(%s) && has(%s)", updated, nameLabel)
	}
	// Convert the list of names to selector
	names := strings.Join(nal.Names, "', '")
	selectors := fmt.Sprintf("%s in { '%s' }", nameLabel, names)

	// Normalize it now
	parsedSelector, err := parser.Parse(selectors)
	if err != nil {
		log.WithError(err).Errorf("Failed to parse names: %s", selectors)
		return ""
	}
	selectors = parsedSelector.String()

	// A list of names is AND'd with the selectors.
	if updated != "" {
		selectors = fmt.Sprintf("(%s) && (%s)", updated, selectors)
	}
	log.Debugf("SA Selector is: %s", selectors)
	return selectors
}

// parseSelectorAttachPrefix takes a v3 selector and returns the appropriate v1 representation
// by prefixing the keys with the given prefix.
// If prefix is `pcns.` then the selector changes from `k == 'v'` to `pcns.k == 'v'`.
func parseSelectorAttachPrefix(s, prefix string) string {
	parsedSelector, err := parser.Parse(s)
	if err != nil {
		log.WithError(err).Errorf("Failed to parse selector: %s (for prefix) %s", s, prefix)
		return ""
	}
	parsedSelector.AcceptVisitor(parser.PrefixVisitor{Prefix: prefix})
	updated := parsedSelector.String()
	log.WithFields(log.Fields{"original": s, "updated": updated}).Debug("Updated selector")
	return updated
}
