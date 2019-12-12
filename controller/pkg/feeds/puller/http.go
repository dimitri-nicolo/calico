// Copyright 2019 Tigera Inc. All rights reserved.

package puller

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	retry "github.com/avast/retry-go"
	log "github.com/sirupsen/logrus"
	calico "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	core "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/tigera/intrusion-detection/controller/pkg/feeds/statser"
	"github.com/tigera/intrusion-detection/controller/pkg/runloop"
)

const (
	CommentPrefix = "#"
	retryAttempts = 3
	retryDelay    = 60 * time.Second
)

// httpPuller is a feed that periodically pulls Puller sets from a URL
type httpPuller struct {
	configMapClient     core.ConfigMapInterface
	secretsClient       core.SecretInterface
	client              *http.Client
	feed                *calico.GlobalThreatFeed
	needsUpdate         bool
	url                 *url.URL
	header              http.Header
	period              time.Duration
	gnsHandler          gnsHandler
	persistence         persistence
	enqueueSyncFunction func()
	syncFailFunction    SyncFailFunction
	cancel              context.CancelFunc
	once                sync.Once
	lock                sync.RWMutex
	content             content
}

type persistence interface {
	lastModified(ctx context.Context, name string) (time.Time, error)
	add(ctx context.Context, name string, snapshot interface{}, f func(error), st statser.Statser)
}

type content interface {
	setFeed(f *calico.GlobalThreatFeed)
	snapshot(r io.Reader) (interface{}, error)
}

type gnsHandler interface {
	setFeed(f *calico.GlobalThreatFeed) bool
	handleSnapshot(ctx context.Context, snapshot interface{}, st statser.Statser, f SyncFailFunction)
	syncFromDB(ctx context.Context, st statser.Statser)
}

func (h *httpPuller) SetFeed(f *calico.GlobalThreatFeed) {
	h.lock.Lock()
	defer h.lock.Unlock()

	needsSync := h.gnsHandler.setFeed(f)
	h.content.setFeed(f)

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
		h.syncFailFunction = func(error) { _ = rescheduleFunc() }

		syncRunFunc, enqueueSyncFunction := runloop.OnDemand()
		go syncRunFunc(ctx, func(ctx context.Context, i interface{}) {
			h.gnsHandler.syncFromDB(ctx, s)
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
			h.gnsHandler.syncFromDB(ctx, s)

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

func (h *httpPuller) setFeedURIAndHeader(f *calico.GlobalThreatFeed) error {
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
				configMap, err := h.configMapClient.Get(header.ValueFrom.ConfigMapKeyRef.Name, meta.GetOptions{})
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
				secret, err := h.secretsClient.Get(header.ValueFrom.SecretKeyRef.Name, meta.GetOptions{})
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
	lastModified, err := h.persistence.lastModified(ctx, h.feed.Name)
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
func (h *httpPuller) queryInfo(s statser.Statser) (name string, u *url.URL, header http.Header, err error) {
	h.lock.RLock()

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
		h.lock.Unlock()
	} else {
		name = h.feed.Name
		u = h.url
		header = h.header
		h.lock.RUnlock()
	}
	return
}

func (h *httpPuller) query(ctx context.Context, st statser.Statser, attempts uint, delay time.Duration) error {
	name, u, header, err := h.queryInfo(st)
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

	snapshot, err := h.content.snapshot(resp.Body)
	if err != nil {
		log.WithError(err).Error("failed to parse snapshot")
		st.Error(statser.PullFailed, err)
		return err
	}

	h.persistence.add(ctx, name, snapshot, h.syncFailFunction, st)
	h.gnsHandler.handleSnapshot(ctx, snapshot, st, h.syncFailFunction)
	st.ClearError(statser.PullFailed)
	st.SuccessfulSync()

	return nil
}
