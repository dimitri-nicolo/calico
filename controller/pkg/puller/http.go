package puller

import (
	"bufio"
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	retry "github.com/avast/retry-go"
	log "github.com/sirupsen/logrus"

	"github.com/tigera/intrusion-detection/controller/pkg/feed"
	"github.com/tigera/intrusion-detection/controller/pkg/runloop"
	"github.com/tigera/intrusion-detection/controller/pkg/statser"
)

const (
	CommentPrefix = "#"
	retryAttempts = 3
	retryDelay    = 60 * time.Second
)

// httpPuller is a feed that periodically pulls Puller sets from a URL
type httpPuller struct {
	client       *http.Client
	feed         feed.Feed
	name         string
	namespace    string
	period       time.Duration
	url          *url.URL
	header       http.Header
	startupDelay time.Duration
	cancel       context.CancelFunc
}

func NewHTTPPuller(feed feed.Feed, client *http.Client, u *url.URL, header http.Header, period, startupDelay time.Duration) Puller {
	return &httpPuller{
		client:       client,
		feed:         feed,
		url:          u,
		period:       period,
		header:       header,
		startupDelay: startupDelay,
	}
}

func (h *httpPuller) Run(ctx context.Context, s statser.Statser) (<-chan feed.IPSet, SyncFailFunction) {
	snapshots := make(chan feed.IPSet)
	ctx, h.cancel = context.WithCancel(ctx)

	runFunc, rescheduleFunc := runloop.RunLoopWithReschedule()
	go func() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(h.startupDelay):
			break
		}
		runFunc(ctx, func() { h.query(ctx, snapshots, s, retryAttempts, retryDelay) }, h.period, func() {}, h.period/3)
	}()

	return snapshots, func() { rescheduleFunc() }
}

func (h *httpPuller) Close() {
	h.cancel()
}

func (h *httpPuller) query(ctx context.Context, snapshots chan<- feed.IPSet, statser statser.Statser, attempts uint, delay time.Duration) {
	req := &http.Request{Method: "GET", Header: h.header, URL: h.url}
	req = req.WithContext(ctx)
	var resp *http.Response
	err := retry.Do(
		func() error {
			var err error
			resp, err = h.client.Do(req)
			if err != nil {
				return err
			}
			if resp.StatusCode >= 500 {
				return &url.Error{
					req.Method,
					h.url.String(),
					TemporaryError(resp.Status),
				}
			}
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				return &url.Error{
					req.Method,
					h.url.String(),
					errors.New(resp.Status),
				}
			}
			return nil
		},
		retry.Attempts(attempts),
		retry.Delay(delay),
		retry.RetryIf(
			func(err error) bool {
				switch err.(type) {
				case net.Error:
					return err.(net.Error).Temporary()
				default:
					return false
				}
			},
		),
		retry.OnRetry(
			func(n uint, err error) {
				log.WithError(err).WithFields(log.Fields{
					"n":   n,
					"url": h.url,
				}).Infof("Retrying")
			},
		),
	)
	if err != nil {
		log.WithError(err).Error("failed to query ")
		statser.Error(statserType, err)
		return
	}
	defer resp.Body.Close()

	// Response format is one Puller address per line.
	s := bufio.NewScanner(resp.Body)
	var snapshot []string
	var n = 0
	for s.Scan() {
		n++
		// strip whitespace
		l := strings.TrimSpace(s.Text())
		// filter comments
		if strings.HasPrefix(l, CommentPrefix) {
			continue
		}
		// filter blank lines
		if len(l) == 0 {
			continue
		}
		if strings.Contains(l, "/") {
			// filter invalid IP addresses, dropping warning
			_, ipNet, err := net.ParseCIDR(l)
			if err != nil {
				log.WithError(err).WithFields(log.Fields{
					"feed":     h.namespace + "/" + h.name,
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
					"feed":     h.namespace + "/" + h.name,
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
	snapshots <- snapshot
	statser.ClearError(statserType)
}
