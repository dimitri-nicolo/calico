// Copyright 2019 Tigera Inc. All rights reserved.

package puller

import (
	"bufio"
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	retry "github.com/avast/retry-go"
	log "github.com/sirupsen/logrus"
	v3 "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/tigera/intrusion-detection/controller/pkg/db"
	"github.com/tigera/intrusion-detection/controller/pkg/feeds/statser"
	"github.com/tigera/intrusion-detection/controller/pkg/feeds/sync/elastic"
	"github.com/tigera/intrusion-detection/controller/pkg/feeds/sync/globalnetworksets"
	"github.com/tigera/intrusion-detection/controller/pkg/runloop"
	"github.com/tigera/intrusion-detection/controller/pkg/util"
)

const (
	CommentPrefix = "#"
	retryAttempts = 3
	retryDelay    = 60 * time.Second
)

// httpPuller is a feed that periodically pulls Puller sets from a URL
type httpPuller struct {
	ipSet               db.Sets
	configMapClient     v1.ConfigMapInterface
	secretsClient       v1.SecretInterface
	client              *http.Client
	feed                *v3.GlobalThreatFeed
	needsUpdate         bool
	url                 *url.URL
	header              http.Header
	period              time.Duration
	gnsController       globalnetworksets.Controller
	elasticController   elastic.IPSetController
	enqueueSyncFunction func()
	syncFailFunction    SyncFailFunction
	cancel              context.CancelFunc
	once                sync.Once
	lock                sync.RWMutex
}

func NewHTTPPuller(
	f *v3.GlobalThreatFeed,
	ipSet db.Sets,
	configMapClient v1.ConfigMapInterface,
	secretsClient v1.SecretInterface,
	client *http.Client,
	gnsController globalnetworksets.Controller,
	elasticController elastic.IPSetController,
) Puller {
	p := &httpPuller{
		ipSet:             ipSet,
		configMapClient:   configMapClient,
		secretsClient:     secretsClient,
		client:            client,
		feed:              f.DeepCopy(),
		needsUpdate:       true,
		gnsController:     gnsController,
		elasticController: elasticController,
	}

	p.period = util.ParseFeedDuration(p.feed)

	return p
}

func (h *httpPuller) SetFeed(f *v3.GlobalThreatFeed) {
	h.lock.Lock()
	defer h.lock.Unlock()

	needsSync := h.feed.Spec.GlobalNetworkSet == nil && f.Spec.GlobalNetworkSet != nil

	h.feed = f.DeepCopy()
	h.needsUpdate = true

	if needsSync {
		h.enqueueSyncFunction()
	}
}

func (h *httpPuller) Run(ctx context.Context, s statser.Statser) {
	h.once.Do(func() {

		h.lock.RLock()
		log.WithField("feed", h.feed.Name).Debug("started HTTP puller")
		h.lock.RUnlock()
		ctx, h.cancel = context.WithCancel(ctx)

		runFunc, rescheduleFunc := runloop.RunLoopWithReschedule()
		h.syncFailFunction = func() { _ = rescheduleFunc() }

		syncRunFunc, enqueueSyncFunction := runloop.OnDemand()
		go syncRunFunc(ctx, func(ctx context.Context, i interface{}) {
			h.syncGNSFromDB(ctx, s)
		})
		h.enqueueSyncFunction = func() {
			enqueueSyncFunction(0)
		}
		go func() {
			defer h.cancel()
			if h.period == 0 {
				return
			}

			// Synchronize the GlobalNetworkSet on startup
			h.syncGNSFromDB(ctx, s)

			delay := h.getStartupDelay(ctx)
			if delay > 0 {
				h.lock.RLock()
				log.WithField("delay", delay).WithField("feed", h.feed.Name).Info("Delaying start")
				h.lock.RUnlock()
			}

			select {
			case <-ctx.Done():
				return
			case <-time.After(delay):
				break
			}
			_ = runFunc(ctx, func() { _ = h.query(ctx, s, retryAttempts, retryDelay) }, h.period, func() {}, h.period/3)
		}()

	})

	return
}

func (h *httpPuller) Close() {
	h.cancel()
}

func (h *httpPuller) setFeedURIAndHeader(f *v3.GlobalThreatFeed) error {
	u, err := url.Parse(f.Spec.Pull.HTTP.URL)
	if err != nil {
		return err
	}

	headers := http.Header{}
	for _, header := range f.Spec.Pull.HTTP.Headers {
		ok := true
		value := header.Value
		if value == "" && header.ValueFrom != nil {
			ok = false
			switch {
			case header.ValueFrom.ConfigMapKeyRef != nil:
				configMap, err := h.configMapClient.Get(header.ValueFrom.ConfigMapKeyRef.Name, metav1.GetOptions{})
				if err != nil {
					if header.ValueFrom.ConfigMapKeyRef.Optional != nil && *header.ValueFrom.ConfigMapKeyRef.Optional {
						log.WithError(err).WithFields(log.Fields{"feed": f.Name, "header": header.Name, "configMapKeyRef": header.ValueFrom.ConfigMapKeyRef.Name, "key": header.ValueFrom.ConfigMapKeyRef.Key}).Debug("Skipping header")
						continue
					}
					return FatalError("could not get ConfigMap %s, %s", header.ValueFrom.ConfigMapKeyRef.Name, err.Error())
				}
				value, ok = configMap.Data[header.ValueFrom.ConfigMapKeyRef.Key]
				if ok {
					log.WithField("value", value).Debug("Loaded config")
				} else if header.ValueFrom.ConfigMapKeyRef.Optional != nil && *header.ValueFrom.ConfigMapKeyRef.Optional {
					log.WithFields(log.Fields{"feed": f.Name, "header": header.Name, "configMapKeyRef": header.ValueFrom.ConfigMapKeyRef.Name, "key": header.ValueFrom.ConfigMapKeyRef.Key}).Debug("Skipping header")
					continue
				} else {
					return FatalError("configMap %s key %s not found", header.ValueFrom.ConfigMapKeyRef.Name, header.ValueFrom.ConfigMapKeyRef.Key)
				}
			case header.ValueFrom.SecretKeyRef != nil:
				secret, err := h.secretsClient.Get(header.ValueFrom.SecretKeyRef.Name, metav1.GetOptions{})
				if err != nil {
					if header.ValueFrom.SecretKeyRef.Optional != nil && *header.ValueFrom.SecretKeyRef.Optional {
						log.WithError(err).WithFields(log.Fields{"feed": f.Name, "header": header.Name, "secretKeyRef": header.ValueFrom.SecretKeyRef.Name, "key": header.ValueFrom.SecretKeyRef.Key}).Debug("Skipping header")
						continue
					}
					return FatalError("could not get Secret %s, %s", header.ValueFrom.SecretKeyRef.Name, err.Error())
				}

				var bvalue []byte
				bvalue, ok = secret.Data[header.ValueFrom.SecretKeyRef.Key]
				value = string(bvalue)
				if ok {
					log.Debug("Loaded secret")
				} else if header.ValueFrom.SecretKeyRef.Optional != nil && *header.ValueFrom.SecretKeyRef.Optional {
					log.WithFields(log.Fields{"feed": f.Name, "header": header.Name, "secretKeyRef": header.ValueFrom.SecretKeyRef.Name, "key": header.ValueFrom.SecretKeyRef.Key}).Debug("Skipping header")
					continue
				} else {
					return FatalError("secrets %s key %s not found", header.ValueFrom.SecretKeyRef.Name, header.ValueFrom.SecretKeyRef.Key)
				}
			default:
				return FatalError("neither ConfigMap nor SecretKey was set")
			}
		}
		headers.Add(header.Name, value)
	}

	h.url = u
	h.header = headers
	h.needsUpdate = false

	return nil
}

func (h *httpPuller) getStartupDelay(ctx context.Context) time.Duration {
	lastModified, err := h.ipSet.GetIPSetModified(ctx, h.feed.Name)
	if err != nil {
		return 0
	}
	since := time.Now().Sub(lastModified)
	if since < h.period {
		return h.period - since
	}
	return 0
}

// queryInfo gets the information required for a query in a threadsafe way
func (h *httpPuller) queryInfo() (name string, u *url.URL, header http.Header, labels map[string]string, gns bool, err error) {
	h.lock.RLock()
	name = h.feed.Name
	u = h.url
	header = h.header
	if h.feed.Spec.GlobalNetworkSet != nil {
		gns = true
		labels = h.feed.Spec.GlobalNetworkSet.Labels
	}

	if h.needsUpdate {
		h.lock.RUnlock()
		h.lock.Lock()

		if h.needsUpdate {
			err = h.setFeedURIAndHeader(h.feed)
			if err != nil {
				h.lock.Unlock()
				return
			}
		}

		name = h.feed.Name
		u = h.url
		header = h.header
		if h.feed.Spec.GlobalNetworkSet != nil {
			gns = true
			labels = h.feed.Spec.GlobalNetworkSet.Labels
		} else {
			gns = false
		}
		h.lock.Unlock()
	} else {
		h.lock.RUnlock()
	}
	return
}

func (h *httpPuller) query(ctx context.Context, st statser.Statser, attempts uint, delay time.Duration) error {
	name, u, header, labels, gns, err := h.queryInfo()
	if err != nil {
		log.WithError(err).Error("failed to query")
		st.Error(statser.PullFailed, err)
		return err
	}
	log.WithField("feed", name).Debug("querying HTTP feed")

	req := &http.Request{Method: "GET", Header: header, URL: u}
	req = req.WithContext(ctx)
	var resp *http.Response
	err = retry.Do(
		func() error {
			var err error
			resp, err = h.client.Do(req)
			if err != nil {
				return err
			}
			if resp.StatusCode >= 500 {
				return &url.Error{
					Op:  req.Method,
					URL: u.String(),
					Err: TemporaryError(resp.Status),
				}
			}
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				return &url.Error{
					Op:  req.Method,
					URL: u.String(),
					Err: errors.New(resp.Status),
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
					"url": u,
				}).Infof("Retrying")
			},
		),
	)
	if err != nil {
		log.WithError(err).Error("failed to query")
		st.Error(statser.PullFailed, err)
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

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
					"feed":     name,
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
					"feed":     name,
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
	h.elasticController.Add(ctx, name, snapshot, h.syncFailFunction, st)
	if gns {
		h.gnsController.Add(makeGNS(name, labels, snapshot), h.syncFailFunction, st)
	}
	st.ClearError(statser.PullFailed)
	st.SuccessfulSync()

	return nil
}

func (h *httpPuller) syncGNSFromDB(ctx context.Context, s statser.Statser) {
	name, _, _, labels, gns, err := h.queryInfo()
	if err != nil {
		log.WithError(err).WithField("feed", name).Error("Failed to query")
		s.Error(statser.GlobalNetworkSetSyncFailed, err)
	} else if gns {
		log.WithField("feed", name).Info("Synchronizing GlobalNetworkSet from cached feed contents")
		ipSet, err := h.ipSet.GetIPSet(ctx, name)
		if err != nil {
			log.WithError(err).WithField("feed", name).Error("Failed to load cached feed contents")
			s.Error(statser.GlobalNetworkSetSyncFailed, err)
		} else {
			h.gnsController.Add(makeGNS(name, labels, ipSet), func() {}, s)
		}
	}
}

func makeGNS(name string, labels map[string]string, snapshot []string) *v3.GlobalNetworkSet {
	gns := util.NewGlobalNetworkSet(name)
	gns.Labels = make(map[string]string)
	for k, v := range labels {
		gns.Labels[k] = v
	}
	gns.Spec.Nets = append([]string{}, snapshot...)
	return gns
}
