package puller

import (
	"bufio"
	"context"
	"github.com/tigera/intrusion-detection/controller/pkg/feed"
	"github.com/tigera/intrusion-detection/controller/pkg/statser"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

const CommentPrefix = "#"

// httpPuller is a feed that periodically pulls Puller sets from a URL
type httpPuller struct {
	feed      feed.Feed
	name      string
	namespace string
	period    time.Duration
	url       *url.URL
	header    http.Header
	cancel    context.CancelFunc
}

func NewHTTPPuller(feed feed.Feed, u *url.URL, period time.Duration, header http.Header) Puller {
	return &httpPuller{
		feed:   feed,
		url:    u,
		period: period,
		header: header,
	}
}

func (h *httpPuller) Run(ctx context.Context, statser statser.Statser) <-chan feed.IPSet {
	snapshots := make(chan feed.IPSet)
	ctx, h.cancel = context.WithCancel(ctx)

	go h.mainloop(ctx, snapshots, statser)

	return snapshots
}

func (h *httpPuller) Close() {
	h.cancel()
}

func (h *httpPuller) Name() string {
	return h.feed.Name()
}

func (h *httpPuller) Namespace() string {
	return h.feed.Namespace()
}

func (h *httpPuller) mainloop(ctx context.Context, snapshots chan<- feed.IPSet, statser statser.Statser) {
	// Query on a timer until the context is cancelled.
	t := time.NewTicker(h.period)
	for {
		h.query(ctx, snapshots, statser)
		select {
		case <-ctx.Done():
			t.Stop()
			return
		case <-t.C:
			// proceed
		}
	}
}

func (h *httpPuller) query(ctx context.Context, snapshots chan<- feed.IPSet, statser statser.Statser) {
	c := http.Client{}
	req := &http.Request{Method: "GET", Header: h.header, URL: h.url}
	req = req.WithContext(ctx)
	resp, err := c.Do(req)
	if err != nil {
		log.WithError(err).Error("failed to query ")
		statser.Error(statserType, err)
		return
	}

	// Response format is one Puller address per line.
	s := bufio.NewScanner(resp.Body)
	var snapshot []string
	var n = 0
	for s.Scan() {
		n++
		l := s.Text()
		// filter comments
		if strings.HasPrefix(l, CommentPrefix) {
			continue
		}
		// filter blank lines
		if len(strings.TrimSpace(l)) == 0 {
			continue
		}
		// filter invalid IP addresses, dropping warning
		ip := net.ParseIP(l)
		if ip == nil {
			log.WithFields(log.Fields{
				"feed":     h.namespace + "/" + h.name,
				"line_num": n,
				"line":     l,
			}).Warn("could not parse IP address")
		}

		snapshot = append(snapshot, s.Text())
	}
	snapshots <- snapshot
	statser.ClearError(statserType)
}
