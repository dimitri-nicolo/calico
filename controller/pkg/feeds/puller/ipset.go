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

func (i ipSetNewlineDelimited) makeGNS(name string, labels map[string]string, snapshot interface{}) *calico.GlobalNetworkSet {
	nets := snapshot.(db.IPSetSpec)
	gns := util.NewGlobalNetworkSet(name)
	gns.Labels = make(map[string]string)
	for k, v := range labels {
		gns.Labels[k] = v
	}
	gns.Spec.Nets = append([]string{}, nets...)
	return gns
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

func (i ipSetPersistence) get(ctx context.Context, name string) (interface{}, error) {
	return i.d.GetIPSet(ctx, name)
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
	p := &httpPuller{
		configMapClient: configMapClient,
		secretsClient:   secretsClient,
		client:          client,
		feed:            f.DeepCopy(),
		needsUpdate:     true,
		gnsController:   gnsController,
		persistence:     d,
		content:         c,
	}

	p.period = util.ParseFeedDuration(p.feed)

	return p
}
