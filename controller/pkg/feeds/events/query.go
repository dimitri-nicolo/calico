// Copyright 2019 Tigera Inc. All rights reserved.

package events

import (
	"context"
	"encoding/json"

	log "github.com/sirupsen/logrus"

	"github.com/tigera/intrusion-detection/controller/pkg/db"
	"github.com/tigera/intrusion-detection/controller/pkg/elastic"
)

type ipSetQuerier struct {
	elastic.SetQuerier
}

func (i ipSetQuerier) QuerySet(ctx context.Context, name string) ([]db.SecurityEventInterface, error) {
	var results []db.SecurityEventInterface
	iter, err := i.QueryIPSet(ctx, name)
	if err != nil {
		return nil, err
	}
	c := 0
	for iter.Next() {
		c++
		key, hit := iter.Value()
		var l FlowLogJSONOutput
		err := json.Unmarshal(hit.Source, &l)
		if err != nil {
			log.WithError(err).WithField("raw", hit.Source).Error("could not unmarshal")
			continue
		}
		sEvent := ConvertFlowLog(l, key, hit, name)
		results = append(results, sEvent)
	}
	log.WithField("num", c).Debug("got events")
	return results, iter.Err()
}

func NewSuspiciousIP(q elastic.SetQuerier) db.SuspiciousSet {
	return ipSetQuerier{q}
}

type domainNameSetQuerier struct {
	elastic.SetQuerier
}

func (d domainNameSetQuerier) QuerySet(ctx context.Context, name string) ([]db.SecurityEventInterface, error) {
	set, err := d.GetDomainNameSet(ctx, name)
	if err != nil {
		return nil, err
	}
	var results []db.SecurityEventInterface
	iter, err := d.QueryDomainNameSet(ctx, name, set)
	if err != nil {
		return nil, err
	}
	// Hash the domain name set for use in conversion
	domains := make(map[string]struct{})
	for _, dn := range set {
		domains[dn] = struct{}{}
	}

	c := 0
	filt := newDNSFilter()
	for iter.Next() {
		c++
		key, hit := iter.Value()
		if filt.pass(hit.Index, hit.Id) {
			var l DNSLog
			err := json.Unmarshal(hit.Source, &l)
			if err != nil {
				log.WithError(err).WithField("raw", hit.Source).Error("could not unmarshal")
				continue
			}
			sEvent := ConvertDNSLog(l, key, hit, domains, name)
			results = append(results, sEvent)
		}
	}
	log.WithField("num", c).Debug("got events")
	return results, iter.Err()
}

func NewSuspiciousDomainNameSet(q elastic.SetQuerier) db.SuspiciousSet {
	return domainNameSetQuerier{q}
}

type indexAndID struct {
	index string
	id    string
}

// DNS logs contain the domain queried for as well as the results, and a suspicious name might appear in multiple
// locations in a single DNS query event. We don't want to create multiple security events for a single DNS query,
// so just take the first one, which we track using a hashmap.
type dnsFilter struct {
	seen map[indexAndID]struct{}
}

// pass returns true only if this is the first log we've seen with a particular index/id.
func (d *dnsFilter) pass(index, id string) bool {
	key := indexAndID{index, id}
	_, ok := d.seen[key]
	d.seen[key] = struct{}{}
	return !ok
}

func newDNSFilter() *dnsFilter {
	return &dnsFilter{seen: make(map[indexAndID]struct{})}
}
