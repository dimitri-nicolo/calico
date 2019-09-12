// Copyright 2019 Tigera Inc. All rights reserved.

package puller

import (
	"context"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	calico "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
	"golang.org/x/net/idna"
	core "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/tigera/intrusion-detection/controller/pkg/db"
	"github.com/tigera/intrusion-detection/controller/pkg/feeds/statser"
	"github.com/tigera/intrusion-detection/controller/pkg/feeds/sync/elastic"
	"github.com/tigera/intrusion-detection/controller/pkg/util"
)

var (
	redundantDots = regexp.MustCompile(`\.\.+`)

	idnaProfile *idna.Profile
	ipOnce      sync.Once
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
		if len(line) == 0 {
			return
		}
		line = canonicalizeDNSName(line)
		// We could check here whether the line represents a valid domain name, but we won't
		// because although a properly configured DNS server will not successfully resolve an
		// invalid name, that doesn't stop an attacker from actually querying for an invalid name.
		// For example, the attacker could direct the query to a DNS server under their control and
		// we want to be able to detect such an action.
		snapshot = append(snapshot, line)
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
	ipOnce.Do(func() {
		idnaProfile = idna.New()
	})
	uname, err := idnaProfile.ToUnicode(name)
	if err != nil {
		return redundantDots.ReplaceAllString(strings.ToLower(strings.Trim(name, ".")), ".")
	}
	return redundantDots.ReplaceAllString(strings.ToLower(strings.Trim(uname, ".")), ".")
}
