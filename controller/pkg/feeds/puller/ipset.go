// Copyright 2019 Tigera Inc. All rights reserved.

package puller

import (
	"bufio"
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	calico "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
	core "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/tigera/intrusion-detection/controller/pkg/db"
	"github.com/tigera/intrusion-detection/controller/pkg/feeds/statser"
	"github.com/tigera/intrusion-detection/controller/pkg/feeds/sync/elastic"
	"github.com/tigera/intrusion-detection/controller/pkg/feeds/sync/globalnetworksets"
	"github.com/tigera/intrusion-detection/controller/pkg/util"
)

type ipSetNewlineDelimited struct{}

type ipSetPersistence struct {
	d db.IPSet
	c elastic.IPSetController
}

type ipSetGNSHandler struct {
	name          string
	labels        map[string]string
	enabled       bool
	gnsController globalnetworksets.Controller
	d             db.IPSet
}

func (i ipSetNewlineDelimited) parse(r io.Reader, logContext *log.Entry) interface{} {
	var snapshot db.IPSetSpec

	// line handler
	h := func(n int, l string) {
		if strings.Contains(l, "/") {
			// filter invalid IP addresses, dropping warning
			_, ipNet, err := net.ParseCIDR(l)
			if err != nil {
				logContext.WithError(err).WithFields(log.Fields{
					"line_num": n,
					"line":     l,
				}).Warn("could not parse IP network")
			} else {
				snapshot = append(snapshot, ipNet.String())
			}
		} else {
			ip := net.ParseIP(l)
			if ip == nil {
				log.WithFields(log.Fields{
					"line_num": n,
					"line":     l,
				}).Warn("could not parse IP address")
			} else {
				// Elastic ip_range requires all addresses to be in CIDR notation
				var ipStr string
				if len(ip.To4()) == net.IPv4len {
					ipStr = ip.String() + "/32"
				} else {
					ipStr = ip.String() + "/128"
				}
				snapshot = append(snapshot, ipStr)
			}
		}
	}

	parseNewlineDelimited(r, h)
	return snapshot
}

func parseNewlineDelimited(r io.Reader, lineHander func(n int, l string)) {
	// Response format is one item per line.
	s := bufio.NewScanner(r)
	var n = 0
	for s.Scan() {
		n++
		l := s.Text()
		// filter comments
		i := strings.Index(l, CommentPrefix)
		if i >= 0 {
			l = l[0:i]
		}
		// strip whitespace
		l = strings.TrimSpace(l)
		// filter blank lines
		if len(l) == 0 {
			continue
		}
		lineHander(n, l)
	}
}

func (i ipSetPersistence) lastModified(ctx context.Context, name string) (time.Time, error) {
	return i.d.GetIPSetModified(ctx, name)
}

func (i ipSetPersistence) add(ctx context.Context, name string, snapshot interface{}, f func(), st statser.Statser) {
	i.c.Add(ctx, name, snapshot.(db.IPSetSpec), f, st)
}

func (h ipSetGNSHandler) get(ctx context.Context) (interface{}, error) {
	return h.d.GetIPSet(ctx, h.name)
}

func (h *ipSetGNSHandler) syncFromDB(ctx context.Context, s statser.Statser) {
	if h.enabled {
		log.WithField("feed", h.name).Info("Synchronizing GlobalNetworkSet from cached feed contents")
		ipSet, err := h.get(ctx)
		if err != nil {
			log.WithError(err).WithField("feed", h.name).Error("Failed to load cached feed contents")
			s.Error(statser.GlobalNetworkSetSyncFailed, err)
		} else {
			g := h.makeGNS(ipSet)
			h.gnsController.Add(g, func() {}, s)
		}
	} else {
		s.ClearError(statser.GlobalNetworkSetSyncFailed)
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

func (h *ipSetGNSHandler) handleSnapshot(ctx context.Context, snapshot interface{}, st statser.Statser, f SyncFailFunction) {
	if h.enabled {
		g := h.makeGNS(snapshot)
		h.gnsController.Add(g, f, st)
	} else {
		st.ClearError(statser.GlobalNetworkSetSyncFailed)
	}
}

func (h *ipSetGNSHandler) setFeed(f *calico.GlobalThreatFeed) bool {
	oldEnabled := h.enabled
	h.name = f.Name
	if f.Spec.GlobalNetworkSet != nil {
		h.enabled = true
		h.labels = make(map[string]string)
		for k, v := range f.Spec.GlobalNetworkSet.Labels {
			h.labels[k] = v
		}
	} else {
		h.enabled = false
		h.labels = nil
	}
	return h.enabled && !oldEnabled
}

func NewIPSetHTTPPuller(
	f *calico.GlobalThreatFeed,
	ipSet db.IPSet,
	configMapClient core.ConfigMapInterface,
	secretsClient core.SecretInterface,
	client *http.Client,
	gnsController globalnetworksets.Controller,
	elasticIPSet elastic.IPSetController,
) Puller {
	d := ipSetPersistence{d: ipSet, c: elasticIPSet}
	c := ipSetNewlineDelimited{}
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
