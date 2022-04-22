// Copyright 2019 Tigera Inc. All rights reserved.

package puller

import (
	"context"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/idna"
	core "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/controller"
	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/db"
	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/feeds/cacher"
	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/feeds/utils"
	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/util"

	calico "github.com/tigera/api/pkg/apis/projectcalico/v3"
)

var (
	redundantDots = regexp.MustCompile(`\.\.+`)

	idnaProfile = idna.New()
)

type dnSetContent struct {
	lock   sync.RWMutex
	parser parser
}

type dnSetPersistence struct {
	d db.DomainNameSet
	c controller.Controller
}

type dnSetGNSHandler struct {
	enabled bool
}

func (d *dnSetContent) snapshot(r io.Reader) (interface{}, error) {
	var snapshot db.DomainNameSetSpec

	// line handler
	h := func(n int, entry string) {
		if len(entry) == 0 {
			return
		}
		entry = canonicalizeDNSName(entry)
		// We could check here whether the entry represents a valid domain name, but we won't
		// because although a properly configured DNS server will not successfully resolve an
		// invalid name, that doesn't stop an attacker from actually querying for an invalid name.
		// For example, the attacker could direct the query to a DNS server under their control and
		// we want to be able to detect such an action.
		snapshot = append(snapshot, entry)
	}

	err := d.parser(r, h)
	return snapshot, err
}

func (p dnSetPersistence) lastModified(ctx context.Context, name string) (time.Time, error) {
	return p.d.GetDomainNameSetModified(ctx, name)
}

func (p dnSetPersistence) add(ctx context.Context, name string, snapshot interface{}, f func(error), feedCacher cacher.GlobalThreatFeedCacher) {
	p.c.Add(ctx, name, snapshot.(db.DomainNameSetSpec), f, feedCacher)
}

func (d *dnSetGNSHandler) handleSnapshot(ctx context.Context, snapshot interface{}, feedCacher cacher.GlobalThreatFeedCacher, f SyncFailFunction) {
	if d.enabled {
		utils.AddErrorToFeedStatus(feedCacher, cacher.GlobalNetworkSetSyncFailed, errors.New("sync not supported for domain name set"))
	} else {
		utils.ClearErrorFromFeedStatus(feedCacher, cacher.GlobalNetworkSetSyncFailed)
	}
}

func (d *dnSetGNSHandler) syncFromDB(ctx context.Context, feedCacher cacher.GlobalThreatFeedCacher) {
	if d.enabled {
		utils.AddErrorToFeedStatus(feedCacher, cacher.GlobalNetworkSetSyncFailed, errors.New("sync not supported for domain name set"))
	} else {
		utils.ClearErrorFromFeedStatus(feedCacher, cacher.GlobalNetworkSetSyncFailed)
	}
}

func NewDomainNameSetHTTPPuller(
	f *calico.GlobalThreatFeed,
	ddb db.DomainNameSet,
	configMapClient core.ConfigMapInterface,
	secretsClient core.SecretInterface,
	client *http.Client,
	e controller.Controller,
) Puller {
	d := dnSetPersistence{d: ddb, c: e}
	c := &dnSetContent{
		parser: getParserForFormat(f.Spec.Pull.HTTP.Format),
	}
	g := &dnSetGNSHandler{}
	if f.Spec.GlobalNetworkSet != nil {
		g.enabled = true
	}
	p := &httpPuller{
		configMapClient: configMapClient,
		secretsClient:   secretsClient,
		client:          client,
		feed:            f.DeepCopy(),
		needsUpdate:     true,
		persistence:     d,
		content:         c,
		gnsHandler:      g,
	}

	p.period = util.ParseFeedDuration(p.feed)

	return p
}

func canonicalizeDNSName(name string) string {
	uname, err := idnaProfile.ToUnicode(name)
	if err != nil {
		return redundantDots.ReplaceAllString(strings.ToLower(strings.Trim(name, ".")), ".")
	}
	return redundantDots.ReplaceAllString(strings.ToLower(strings.Trim(uname, ".")), ".")
}
