// Copyright 2019 Tigera Inc. All rights reserved.

package puller

import (
	"context"
	"github.com/tigera/intrusion-detection/controller/pkg/util"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	calico "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
	core "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/tigera/intrusion-detection/controller/pkg/db"
	"github.com/tigera/intrusion-detection/controller/pkg/feeds/statser"
	"github.com/tigera/intrusion-detection/controller/pkg/feeds/sync/elastic"
)

var (
	nameLabelFmt     = "[a-z0-9]([-a-z0-9]*[a-z0-9])?"
	nameSubdomainFmt = nameLabelFmt + "(\\." + nameLabelFmt + ")*"

	// All resource names must follow the subdomain name format.  Some resources we impose
	// more restrictive naming requirements.
	nameRegex     = regexp.MustCompile("^" + nameSubdomainFmt + "$")
	redundantDots = regexp.MustCompile(`\.\.+`)
)

type dnSetNewlineDelimited struct{}

type dnSetPersistence struct {
	d db.DomainNameSet
	c elastic.DomainNameSetController
}

func (i dnSetNewlineDelimited) parse(r io.Reader, logContext *log.Entry) interface{} {
	var snapshot db.DomainNameSetSpec

	// line handler
	h := func(n int, line string) {
		// TODO (spike) handle international domain names properly
		line = canonicalizeDNSName(line)
		if nameRegex.MatchString(line) {
			snapshot = append(snapshot, line)
		} else {
			logContext.WithFields(log.Fields{
				"line":     line,
				"line_num": n,
			}).Warn("unable to parse domain name")
		}
	}

	parseNewlineDelimited(r, h)

	return snapshot
}

func (i dnSetNewlineDelimited) makeGNS(name string, labels map[string]string, snapshot interface{}) *calico.GlobalNetworkSet {
	panic("not supported")
}

func (i dnSetPersistence) lastModified(ctx context.Context, name string) (time.Time, error) {
	return i.d.GetDomainNameSetModified(ctx, name)
}

func (i dnSetPersistence) add(ctx context.Context, name string, snapshot interface{}, f func(), st statser.Statser) {
	i.c.Add(ctx, name, snapshot.(db.DomainNameSetSpec), f, st)
}

func (i dnSetPersistence) get(ctx context.Context, name string) (interface{}, error) {
	panic("not supported")
}

func NewDomainNameSetHTTPPuller(
	f *calico.GlobalThreatFeed,
	ddb db.DomainNameSet,
	configMapClient core.ConfigMapInterface,
	secretsClient core.SecretInterface,
	client *http.Client,
	e elastic.DomainNameSetController,
) Puller {
	d := dnSetPersistence{d: ddb, c: e}
	c := dnSetNewlineDelimited{}
	p := &httpPuller{
		configMapClient: configMapClient,
		secretsClient:   secretsClient,
		client:          client,
		feed:            f.DeepCopy(),
		needsUpdate:     true,
		persistence:     d,
		content:         c,
	}

	p.period = util.ParseFeedDuration(p.feed)

	return p
}

func canonicalizeDNSName(name string) string {
	return redundantDots.ReplaceAllString(strings.ToLower(strings.Trim(name, ".")), ".")
}
