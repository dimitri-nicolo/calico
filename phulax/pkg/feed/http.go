package feed

import (
	"bufio"
	"context"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

const CommentPrefix = "#"

// httpPuller is a feed that periodically pulls IPSetPuller sets from a URL
type httpPuller struct {
	name      string
	namespace string
	period    time.Duration
	url       *url.URL
	header    http.Header
}

func NewHTTPPuller(name, namespace string, u *url.URL, period time.Duration, header http.Header) IPSetPuller {
	return &httpPuller{
		name:      name,
		namespace: namespace,
		url:       u,
		period:    period,
		header:    header,
	}
}

func (h *httpPuller) Run(ctx context.Context) <-chan []string {
	snapshots := make(chan []string)

	go h.mainloop(ctx, snapshots)

	return snapshots
}

func (h *httpPuller) Name() string {
	return h.name
}

func (h *httpPuller) Namespace() string {
	return h.namespace
}

func (h *httpPuller) mainloop(ctx context.Context, snapshots chan<- []string) {

	// do one query at start of day to make sure we have data
	h.query(snapshots)

	// Query on a timer until the context is cancelled.
	t := time.NewTicker(h.period)
	for {
		select {
		case <-ctx.Done():
			t.Stop()
			return
		case <-t.C:
			// proceed
		}
		h.query(snapshots)
	}
}

func (h *httpPuller) query(snapshots chan<- []string) {
	c := http.Client{}
	req := &http.Request{Method: "GET", Header: h.header, URL: h.url}
	resp, err := c.Do(req)
	if err != nil {
		log.WithError(err).Error("failed to query ")
		return
	}

	// Response format is one IPSetPuller address per line.
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
}
