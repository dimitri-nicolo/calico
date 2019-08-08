// Copyright 2019 Tigera Inc. All rights reserved.

package statser

import (
	"context"
	"errors"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	v32 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	v3 "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/tigera/intrusion-detection/controller/pkg/calico"
)

func TestStatser_SuccessfulSync(t *testing.T) {
	g := NewGomegaWithT(t)

	name := "test-feed"
	gtf := &calico.MockGlobalThreatFeedInterface{
		GlobalThreatFeed: &v3.GlobalThreatFeed{},
	}

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	st := NewStatser(name, gtf).(*statser)
	st.Run(ctx)

	g.Expect(st.lastSuccessfulSync).Should(Equal(time.Time{}))
	st.SuccessfulSync()
	g.Expect(st.lastSuccessfulSync).Should(BeTemporally("~", time.Now(), time.Second))
	g.Eventually(func() time.Time { return gtf.GlobalThreatFeed.Status.LastSuccessfulSync.Time }).Should(BeTemporally("==", st.lastSuccessfulSync))

	g.Expect(st.lastSuccessfulSearch).Should(Equal(time.Time{}))
	g.Consistently(func() time.Time { return gtf.GlobalThreatFeed.Status.LastSuccessfulSearch.Time }).Should(Equal(time.Time{}))

	g.Consistently(func() []v32.ErrorCondition { return gtf.GlobalThreatFeed.Status.ErrorConditions }).Should(HaveLen(0))
}

func TestStatser_SuccessfulSearch(t *testing.T) {
	g := NewGomegaWithT(t)

	name := "test-feed"
	gtf := &calico.MockGlobalThreatFeedInterface{
		GlobalThreatFeed: &v3.GlobalThreatFeed{},
	}

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	st := NewStatser(name, gtf).(*statser)
	st.Run(ctx)

	g.Expect(st.lastSuccessfulSearch).Should(Equal(time.Time{}))
	st.SuccessfulSearch()
	g.Expect(st.lastSuccessfulSearch).Should(BeTemporally("~", time.Now(), time.Second))
	g.Eventually(func() time.Time { return gtf.GlobalThreatFeed.Status.LastSuccessfulSearch.Time }).Should(BeTemporally("==", st.lastSuccessfulSearch))

	g.Expect(st.lastSuccessfulSync).Should(Equal(time.Time{}))
	g.Consistently(func() time.Time { return gtf.GlobalThreatFeed.Status.LastSuccessfulSync.Time }).Should(Equal(time.Time{}))

	g.Consistently(func() []v32.ErrorCondition { return gtf.GlobalThreatFeed.Status.ErrorConditions }).Should(HaveLen(0))
}

func TestStatser_Error_ClearError(t *testing.T) {
	g := NewGomegaWithT(t)

	name := "test-feed"
	gtf := &calico.MockGlobalThreatFeedInterface{
		GlobalThreatFeed: &v3.GlobalThreatFeed{},
	}

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	st := NewStatser(name, gtf).(*statser)
	st.Run(ctx)

	errStr1 := "test1"
	st.Error(ElasticSyncFailed, errors.New(errStr1))

	g.Expect(st.lastSuccessfulSync).Should(Equal(time.Time{}), "lastSuccessfulSync was not modified")
	g.Expect(st.lastSuccessfulSearch).Should(Equal(time.Time{}), "lastSuccessfulSearch was not modified")
	g.Consistently(func() time.Time { return gtf.GlobalThreatFeed.Status.LastSuccessfulSync.Time }).Should(Equal(time.Time{}))
	g.Consistently(func() time.Time { return gtf.GlobalThreatFeed.Status.LastSuccessfulSearch.Time }).Should(Equal(time.Time{}))

	g.Expect(st.errorConditions.TypedErrors(ElasticSyncFailed)).Should(ConsistOf(v32.ErrorCondition{ElasticSyncFailed, errStr1}))
	g.Eventually(func() []v32.ErrorCondition { return gtf.GlobalThreatFeed.Status.ErrorConditions }).Should(HaveLen(1))

	errStr2 := "test2"
	st.Error(ElasticSyncFailed, errors.New(errStr2))

	g.Expect(st.lastSuccessfulSync).Should(Equal(time.Time{}), "lastSuccessfulSync was not modified")
	g.Expect(st.lastSuccessfulSearch).Should(Equal(time.Time{}), "lastSuccessfulSearch was not modified")
	g.Consistently(func() time.Time { return gtf.GlobalThreatFeed.Status.LastSuccessfulSync.Time }).Should(Equal(time.Time{}))
	g.Consistently(func() time.Time { return gtf.GlobalThreatFeed.Status.LastSuccessfulSearch.Time }).Should(Equal(time.Time{}))

	g.Expect(st.errorConditions.TypedErrors(ElasticSyncFailed)).Should(ConsistOf(
		v32.ErrorCondition{ElasticSyncFailed, errStr1},
		v32.ErrorCondition{ElasticSyncFailed, errStr2},
	))
	g.Eventually(func() []v32.ErrorCondition { return gtf.GlobalThreatFeed.Status.ErrorConditions }).Should(HaveLen(2))

	errStr3 := "test3"
	st.Error(PullFailed, errors.New(errStr3))

	g.Expect(st.errorConditions.TypedErrors(ElasticSyncFailed)).Should(ConsistOf(
		v32.ErrorCondition{ElasticSyncFailed, errStr1},
		v32.ErrorCondition{ElasticSyncFailed, errStr2},
	))
	g.Expect(st.errorConditions.TypedErrors(PullFailed)).Should(ConsistOf(
		v32.ErrorCondition{PullFailed, errStr3},
	))
	g.Eventually(func() []v32.ErrorCondition { return gtf.GlobalThreatFeed.Status.ErrorConditions }).Should(HaveLen(3))

	st.ClearError(ElasticSyncFailed)
	g.Expect(st.errorConditions.TypedErrors(PullFailed)).Should(ConsistOf(
		v32.ErrorCondition{PullFailed, errStr3},
	))
	g.Eventually(func() []v32.ErrorCondition { return gtf.GlobalThreatFeed.Status.ErrorConditions }).Should(HaveLen(1))

	st.ClearError(PullFailed)
	g.Expect(st.errorConditions.Errors()).Should(HaveLen(0))
	g.Eventually(func() []v32.ErrorCondition { return gtf.GlobalThreatFeed.Status.ErrorConditions }).Should(HaveLen(0))
}

func TestStatser_Status(t *testing.T) {
	g := NewGomegaWithT(t)

	name := "test-feed"
	origStatus := v32.GlobalThreatFeedStatus{
		LastSuccessfulSync: v1.Time{time.Now().Add(-time.Hour * 24)},
	}
	gtf := &calico.MockGlobalThreatFeedInterface{
		GlobalThreatFeed: &v3.GlobalThreatFeed{Status: origStatus},
	}

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	st := NewStatser(name, gtf).(*statser)
	st.Run(ctx)

	// Try first with nothing set
	status := st.Status()
	g.Expect(st.lastSuccessfulSync).Should(Equal(time.Time{}), "lastSuccessfulSync was not modified")
	g.Expect(st.lastSuccessfulSearch).Should(Equal(time.Time{}), "lastSuccessfulSearch was not modified")
	g.Expect(st.errorConditions.Errors()).Should(HaveLen(0), "No errors were created")

	g.Expect(status.LastSuccessfulSync.Time).Should(Equal(time.Time{}))
	g.Expect(status.LastSuccessfulSearch.Time).Should(Equal(time.Time{}))
	g.Expect(status.ErrorConditions).Should(HaveLen(0))

	g.Consistently(func() v32.GlobalThreatFeedStatus { return gtf.GlobalThreatFeed.Status }).Should(Equal(origStatus), "updateStatus was not called")

	// Try again with some members set
	st.lastSuccessfulSearch = time.Now().Add(-time.Hour)
	st.lastSuccessfulSync = time.Now()
	st.errorConditions.Add(ElasticSyncFailed, errors.New("test1"))
	st.errorConditions.Add(PullFailed, errors.New("test2"))
	st.errorConditions.Add(PullFailed, errors.New("test3"))

	status = st.Status()
	g.Expect(status.LastSuccessfulSync.Time).Should(BeTemporally("==", st.lastSuccessfulSync))
	g.Expect(status.LastSuccessfulSearch.Time).Should(BeTemporally("==", st.lastSuccessfulSearch))
	g.Expect(status.ErrorConditions).Should(HaveLen(3))

	g.Consistently(func() v32.GlobalThreatFeedStatus { return gtf.GlobalThreatFeed.Status }).Should(Equal(origStatus), "updateStatus was not called")
}

func TestStatser_status_deadlock(t *testing.T) {
	g := NewGomegaWithT(t)

	name := "test-feed"
	gtf := &calico.MockGlobalThreatFeedInterface{
		GlobalThreatFeed: &v3.GlobalThreatFeed{},
	}

	st := NewStatser(name, gtf).(*statser)

	ch := make(chan struct{})

	go func() {
		st.lock.Lock()
		defer st.lock.Unlock()
		_ = st.status()
		close(ch)
	}()

	g.Eventually(ch).Should(BeClosed(), "status does not deadlock")
}

func TestStatser_updateStatus(t *testing.T) {
	g := NewGomegaWithT(t)

	name := "test-feed"
	gtf := &calico.MockGlobalThreatFeedInterface{
		GlobalThreatFeed: &v3.GlobalThreatFeed{},
	}

	st := NewStatser(name, gtf).(*statser)

	st.lastSuccessfulSync = time.Now().Add(-time.Hour)
	st.lastSuccessfulSearch = time.Now()
	st.errorConditions.Add(ElasticSyncFailed, errors.New("test error"))

	st.updateStatus(st.status())

	g.Expect(gtf.GlobalThreatFeed.Status.LastSuccessfulSync.Time).Should(BeTemporally("==", st.lastSuccessfulSync))
	g.Expect(gtf.GlobalThreatFeed.Status.LastSuccessfulSearch.Time).Should(BeTemporally("==", st.lastSuccessfulSearch))
	g.Expect(gtf.GlobalThreatFeed.Status.ErrorConditions).Should(ConsistOf(st.errorConditions.TypedErrors(ElasticSyncFailed)))
}
