// Copyright 2019 Tigera Inc. All rights reserved.

package watcher

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	libcalicov3 "github.com/projectcalico/libcalico-go/lib/apis/v3"

	. "github.com/onsi/gomega"
	v3 "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/tigera/intrusion-detection/controller/pkg/alert/elastic"
	"github.com/tigera/intrusion-detection/controller/pkg/calico"
	idsElastic "github.com/tigera/intrusion-detection/controller/pkg/elastic"
	"github.com/tigera/intrusion-detection/controller/pkg/util"
)

var testClient = &http.Client{Transport: &util.MockRoundTripper{Error: errors.New("mock error")}}

func TestWatcher_processQueue(t *testing.T) {
	g := NewGomegaWithT(t)

	gaf := &calico.MockGlobalAlertInterface{
		GlobalAlert: &v3.GlobalAlert{},
	}
	xpack := &idsElastic.MockXPackWatcher{
		Err: errors.New("test"),
	}
	c := elastic.NewMockAlertsController()

	w := NewWatcher(gaf, c, xpack, testClient).(*watcher)

	g.Expect(w).ShouldNot(BeNil())

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()
	w.ctx = ctx

	// a non-existing alert is deleted.
	err := w.processQueue(cache.Deltas{
		{
			Type: cache.Deleted,
			Object: &v3.GlobalAlert{
				ObjectMeta: v1.ObjectMeta{
					Name: "nonexisting",
				},
			},
		},
	})
	g.Expect(err).NotTo(HaveOccurred())

	// a alert is added
	err = w.processQueue(cache.Deltas{
		{
			Type: cache.Added,
			Object: &v3.GlobalAlert{
				ObjectMeta: v1.ObjectMeta{
					Name: "alert1",
				},
			},
		},
	})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(w.listAlertWatchers()).Should(HaveLen(1))

	// a non-existing alert is updated (should never happen)
	err = w.processQueue(cache.Deltas{
		{
			Type: cache.Updated,
			Object: &v3.GlobalAlert{
				ObjectMeta: v1.ObjectMeta{
					Name: "alert2",
				},
			},
		},
	})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(w.listAlertWatchers()).Should(HaveLen(2))

	// an existing alert is added again
	err = w.processQueue(cache.Deltas{
		{
			Type: cache.Added,
			Object: &v3.GlobalAlert{
				ObjectMeta: v1.ObjectMeta{
					Name: "alert1",
				},
				Spec: libcalicov3.GlobalAlertSpec{
					Description: "test",
				},
			},
		},
	})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(w.listAlertWatchers()).Should(HaveLen(2))
	aw, ok := w.getAlertWatcher("alert1")
	g.Expect(ok).Should(BeTrue())
	g.Expect(aw.alert.Spec.Description).Should(Equal("test"))

	// an existing alert is modified
	err = w.processQueue(cache.Deltas{
		{
			Type: cache.Added,
			Object: &v3.GlobalAlert{
				ObjectMeta: v1.ObjectMeta{
					Name: "alert1",
				},
				Spec: libcalicov3.GlobalAlertSpec{
					Description: "test2",
				},
			},
		},
	})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(w.listAlertWatchers()).Should(HaveLen(2))
	aw, ok = w.getAlertWatcher("alert1")
	g.Expect(ok).Should(BeTrue())
	g.Expect(aw.alert.Spec.Description).Should(Equal("test2"))

	// an existing alert is deleted
	err = w.processQueue(cache.Deltas{
		{
			Type: cache.Deleted,
			Object: &v3.GlobalAlert{
				ObjectMeta: v1.ObjectMeta{
					Name: "alert1",
				},
			},
		},
	})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(w.listAlertWatchers()).Should(HaveLen(1))
	_, ok = w.getAlertWatcher("alert1")
	g.Expect(ok).Should(BeFalse())
}

func TestWatcher_startAlert_stopAlert(t *testing.T) {
	g := NewGomegaWithT(t)

	a := util.NewGlobalAlert("mock")

	gaf := &calico.MockGlobalAlertInterface{
		GlobalAlert: &v3.GlobalAlert{},
	}
	xpack := &idsElastic.MockXPackWatcher{
		Err: errors.New("test"),
	}
	c := elastic.NewMockAlertsController()

	w := NewWatcher(gaf, c, xpack, testClient).(*watcher)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	w.startAlertWatcher(ctx, a)

	aw, ok := w.getAlertWatcher(a.Name)
	g.Expect(ok).Should(BeTrue(), "AlertWatchers map contains alert")
	g.Expect(w.listAlertWatchers()).To(HaveLen(1), "Only one AlertWatcher")

	g.Expect(aw.alert).Should(Equal(a))
	g.Expect(c.NotGCable()).Should(HaveKey(a.Name))
	g.Expect(aw.job).ShouldNot(BeNil())
	g.Expect(aw.statser).ShouldNot(BeNil())

	w.stopAlertWatcher(ctx, a.Name)
	_, ok = w.getAlertWatcher(a.Name)
	g.Expect(ok).Should(BeFalse(), "AlertWatchers map does not contain alert")
	g.Expect(w.listAlertWatchers()).To(HaveLen(0), "No AlertWatchers")
	g.Expect(c.NotGCable()).ShouldNot(HaveKey(a.Name))
}

func TestWatcher_startAlert_Exists(t *testing.T) {
	g := NewGomegaWithT(t)

	a := util.NewGlobalAlert("mock")

	gaf := &calico.MockGlobalAlertInterface{
		GlobalAlert: &v3.GlobalAlert{},
	}
	xpack := &idsElastic.MockXPackWatcher{
		Err: errors.New("test"),
	}
	c := elastic.NewMockAlertsController()

	w := NewWatcher(gaf, c, xpack, testClient).(*watcher)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	w.startAlertWatcher(ctx, a)

	_, ok := w.getAlertWatcher(a.Name)
	g.Expect(ok).Should(BeTrue(), "AlertWatchers map contains alert")
	g.Expect(w.listAlertWatchers()).To(HaveLen(1), "Only one AlertWatcher")

	g.Expect(func() { w.startAlertWatcher(ctx, a) }).Should(Panic())
}

func TestWatcher_stopAlert_notExists(t *testing.T) {
	g := NewGomegaWithT(t)

	gaf := &calico.MockGlobalAlertInterface{
		GlobalAlert: &v3.GlobalAlert{},
	}
	xpack := &idsElastic.MockXPackWatcher{
		Err: errors.New("test"),
	}
	c := elastic.NewMockAlertsController()

	w := NewWatcher(gaf, c, xpack, testClient).(*watcher)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	g.Expect(func() { w.stopAlertWatcher(ctx, "mock") }).Should(Panic())
}

func TestWatcher_updateAlert_NotStarted(t *testing.T) {
	g := NewGomegaWithT(t)

	a := util.NewGlobalAlert("mock")
	b := util.NewGlobalAlert("mock2")

	gaf := &calico.MockGlobalAlertInterface{
		GlobalAlert: &v3.GlobalAlert{},
	}
	xpack := &idsElastic.MockXPackWatcher{
		Err: errors.New("test"),
	}
	c := elastic.NewMockAlertsController()

	w := NewWatcher(gaf, c, xpack, testClient).(*watcher)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	g.Expect(func() { w.updateAlertWatcher(ctx, a, b) }).Should(Panic())
}

func TestWatcher_updateAlert(t *testing.T) {
	g := NewGomegaWithT(t)

	a := util.NewGlobalAlert("mock")

	gaf := &calico.MockGlobalAlertInterface{
		GlobalAlert: &v3.GlobalAlert{},
	}
	xpack := &idsElastic.MockXPackWatcher{
		Err: errors.New("test"),
	}
	c := elastic.NewMockAlertsController()

	w := NewWatcher(gaf, c, xpack, testClient).(*watcher)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	w.startAlertWatcher(ctx, a)

	_, ok := w.getAlertWatcher(a.Name)
	g.Expect(ok).Should(BeTrue(), "AlertWatchers map contains alert")
	g.Expect(w.listAlertWatchers()).To(HaveLen(1), "Only one AlertWatcher")
	g.Expect(c.NotGCable()).Should(HaveKey(a.Name))

	a2 := a.DeepCopy()
	a2.Spec.DataSet = "dns"
	g.Expect(a.Spec.DataSet).ShouldNot(Equal(a2.Spec.DataSet))

	w.updateAlertWatcher(ctx, a, a2)

	_, ok = w.getAlertWatcher(a.Name)
	g.Expect(ok).Should(BeTrue(), "AlertWatchers map contains alert")
	g.Expect(w.listAlertWatchers()).To(HaveLen(1), "Only one AlertWatcher")
	g.Expect(c.NotGCable()).Should(HaveKey(a.Name))
	g.Eventually(func() interface{} { return c.Bodies() }).Should(HaveKey(a2.Name))
	g.Expect(c.Bodies()[a2.Name].Input.Search.Request.Indices).Should(ConsistOf(idsElastic.DNSLogIndex))
}

func TestWatcher_Ping(t *testing.T) {
	g := NewWithT(t)

	gaf := &calico.MockGlobalAlertInterface{
		GlobalAlertList: &v3.GlobalAlertList{},
	}
	xpack := &idsElastic.MockXPackWatcher{
		Err: errors.New("test"),
	}
	c := elastic.NewMockAlertsController()

	uut := NewWatcher(gaf, c, xpack, testClient).(*watcher)

	ch := make(chan struct{})
	defer func() {
		g.Eventually(ch).Should(BeClosed(), "Test cleans up correctly")
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	go func() {
		defer close(ch)
		err := uut.Ping(ctx)
		g.Expect(err).ToNot(HaveOccurred())
	}()
	g.Consistently(ch).ShouldNot(BeClosed(), "Ping does not complete before Run is called")

	uut.Run(ctx)

	g.Eventually(ch).Should(BeClosed(), "Ping completes after Run is called")
}

func TestWatcher_PingFail(t *testing.T) {
	g := NewWithT(t)

	ctx, cancel := context.WithTimeout(context.TODO(), time.Millisecond)
	defer cancel()

	gaf := &calico.MockGlobalAlertInterface{
		GlobalAlertList: &v3.GlobalAlertList{},
	}
	xpack := &idsElastic.MockXPackWatcher{
		Err: errors.New("test"),
	}
	c := elastic.NewMockAlertsController()

	uut := NewWatcher(gaf, c, xpack, testClient).(*watcher)

	err := uut.Ping(ctx)
	g.Expect(err).Should(MatchError(context.DeadlineExceeded), "Ping times out")
}
