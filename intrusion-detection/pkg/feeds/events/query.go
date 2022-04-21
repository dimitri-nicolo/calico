// Copyright 2019 Tigera Inc. All rights reserved.

package events

import (
	"context"
	"encoding/json"
	"time"

	log "github.com/sirupsen/logrus"

	apiV3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"github.com/projectcalico/calico/intrusion-detection/controller/pkg/db"
	"github.com/projectcalico/calico/intrusion-detection/controller/pkg/elastic"
)

type ipSetQuerier struct {
	elastic.SetQuerier
}

func (i ipSetQuerier) QuerySet(ctx context.Context, feed *apiV3.GlobalThreatFeed) ([]db.SecurityEventInterface, time.Time, string, error) {
	var results []db.SecurityEventInterface
	lastSuccessfulSearch := time.Now()
	iter, ipSetHash, err := i.QueryIPSet(ctx, feed)
	if err != nil {
		return nil, time.Time{}, ipSetHash, err
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
		sEvent := ConvertFlowLog(l, key, hit, feed.Name)
		results = append(results, sEvent)
	}
	log.WithField("num", c).Debug("got events")
	return results, lastSuccessfulSearch, ipSetHash, iter.Err()
}

func NewSuspiciousIP(q elastic.SetQuerier) db.SuspiciousSet {
	return ipSetQuerier{q}
}

type domainNameSetQuerier struct {
	elastic.SetQuerier
}

func (d domainNameSetQuerier) QuerySet(ctx context.Context, feed *apiV3.GlobalThreatFeed) ([]db.SecurityEventInterface, time.Time, string, error) {
	set, err := d.GetDomainNameSet(ctx, feed.Name)
	if err != nil {
		return nil, time.Time{}, "", err
	}
	var results []db.SecurityEventInterface
	lastSuccessfulSearch := time.Now()
	iter, domainNameSetHash, err := d.QueryDomainNameSet(ctx, set, feed)
	if err != nil {
		return nil, time.Time{}, domainNameSetHash, err
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
			sEvent := ConvertDNSLog(l, key, hit, domains, feed.Name)
			results = append(results, sEvent)
		}
	}
	log.WithField("num", c).Debug("got events")
	return results, lastSuccessfulSearch, domainNameSetHash, iter.Err()
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
