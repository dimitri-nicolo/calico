// Copyright 2019 Tigera Inc. All rights reserved.

package puller

import (
	"context"
	"io"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	core "k8s.io/client-go/kubernetes/typed/core/v1"

	calico "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"github.com/tigera/intrusion-detection/controller/pkg/controller"
	"github.com/tigera/intrusion-detection/controller/pkg/db"
	"github.com/tigera/intrusion-detection/controller/pkg/feeds/cacher"
	"github.com/tigera/intrusion-detection/controller/pkg/feeds/sync/globalnetworksets"
	"github.com/tigera/intrusion-detection/controller/pkg/feeds/utils"
	"github.com/tigera/intrusion-detection/controller/pkg/util"
)

var (
	wrappedInBracketsRegexp = regexp.MustCompile(`^\[.*\]$`)
)

type ipSetPersistence struct {
	d db.IPSet
	c controller.Controller
}

type ipSetGNSHandler struct {
	name          string
	labels        map[string]string
	enabled       bool
	gnsController globalnetworksets.Controller
	d             db.IPSet
}

type ipSetContent struct {
	name   string
	parser parser
}

func (i *ipSetContent) snapshot(r io.Reader) (interface{}, error) {
	var snapshot db.IPSetSpec
	var once sync.Once

	// entry handler
	h := func(n int, entry string) {
		snapshot = append(snapshot, parseIP(entry, log.WithField("name", i.name), n, &once)...)
	}

	err := i.parser(r, h)
	return snapshot, err
}

func parseIP(entry string, logContext *log.Entry, n int, once *sync.Once) db.IPSetSpec {
	if wrappedInBracketsRegexp.MatchString(entry) {
		entry = entry[1 : len(entry)-1]
	}
	if strings.Contains(entry, "/") {
		// filter invalid IP addresses, dropping warning
		_, ipNet, err := net.ParseCIDR(entry)
		if err != nil {
			once.Do(func() {
				logContext.WithError(err).WithFields(log.Fields{
					"entry_num": n,
					"entry":     entry,
				}).Warn("could not parse IP network")
			})
			return nil
		} else {
			return db.IPSetSpec{ipNet.String()}
		}
	} else {
		ip := net.ParseIP(entry)
		if ip == nil {
			once.Do(func() {
				log.WithFields(log.Fields{
					"entry_num": n,
					"entry":     entry,
				}).Warn("could not parse IP address")
			})
			return nil
		} else {
			// Elastic ip_range requires all addresses to be in CIDR notation
			var ipStr string
			if len(ip.To4()) == net.IPv4len {
				ipStr = ip.String() + "/32"
			} else {
				ipStr = ip.String() + "/128"
			}
			return db.IPSetSpec{ipStr}
		}
	}
}

func (i ipSetPersistence) lastModified(ctx context.Context, name string) (time.Time, error) {
	return i.d.GetIPSetModified(ctx, name)
}

func (i ipSetPersistence) add(ctx context.Context, name string, snapshot interface{}, f func(error), feedCacher cacher.GlobalThreatFeedCacher) {
	i.c.Add(ctx, name, snapshot.(db.IPSetSpec), f, feedCacher)
}

func (h ipSetGNSHandler) get(ctx context.Context) (interface{}, error) {
	return h.d.GetIPSet(ctx, h.name)
}

func (h *ipSetGNSHandler) syncFromDB(ctx context.Context, feedCacher cacher.GlobalThreatFeedCacher) {
	if h.enabled {
		log.WithField("feed", h.name).Info("synchronizing GlobalNetworkSet from cached feed contents")
		ipSet, err := h.get(ctx)
		if err != nil {
			log.WithError(err).WithField("feed", h.name).Error("failed to load cached feed contents")
			utils.AddErrorToFeedStatus(feedCacher, cacher.GlobalNetworkSetSyncFailed, err)
		} else {
			g := h.makeGNS(ipSet)
			h.gnsController.Add(g, func(error) {}, feedCacher)
		}
	} else {
		utils.ClearErrorFromFeedStatus(feedCacher, cacher.GlobalNetworkSetSyncFailed)
	}
}

func (h *ipSetGNSHandler) makeGNS(snapshot interface{}) *calico.GlobalNetworkSet {
	nets := snapshot.(db.IPSetSpec)
	gns := util.NewGlobalNetworkSet(h.name)
	gns.Labels = make(map[string]string)
	for k, v := range h.labels {
		gns.Labels[k] = v
	}
	gns.Spec.Nets = append([]string{}, nets...)
	return gns
}

func (h *ipSetGNSHandler) handleSnapshot(ctx context.Context, snapshot interface{}, feedCacher cacher.GlobalThreatFeedCacher, f SyncFailFunction) {
	if h.enabled {
		g := h.makeGNS(snapshot)
		h.gnsController.Add(g, f, feedCacher)
	} else {
		utils.ClearErrorFromFeedStatus(feedCacher, cacher.GlobalNetworkSetSyncFailed)
	}
}

func NewIPSetHTTPPuller(
	f *calico.GlobalThreatFeed,
	ipSet db.IPSet,
	configMapClient core.ConfigMapInterface,
	secretsClient core.SecretInterface,
	client *http.Client,
	gnsController globalnetworksets.Controller,
	elasticIPSet controller.Controller,
) Puller {
	d := ipSetPersistence{d: ipSet, c: elasticIPSet}
	c := &ipSetContent{name: f.Name, parser: getParserForFormat(f.Spec.Pull.HTTP.Format)}
	g := &ipSetGNSHandler{
		name:          f.Name,
		gnsController: gnsController,
		d:             ipSet,
	}
	if f.Spec.GlobalNetworkSet != nil {
		g.enabled = true
		g.labels = make(map[string]string)
		for k, v := range f.Spec.GlobalNetworkSet.Labels {
			g.labels[k] = v
		}
	}
	p := &httpPuller{
		configMapClient: configMapClient,
		secretsClient:   secretsClient,
		client:          client,
		feed:            f.DeepCopy(),
		needsUpdate:     true,
		gnsHandler:      g,
		persistence:     d,
		content:         c,
	}

	p.period = util.ParseFeedDuration(p.feed)

	return p
}
