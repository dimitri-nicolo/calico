// Copyright 2021 Tigera Inc. All rights reserved.

package cacher

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/calico"
)

func TestGlobalThreatFeedCache_GetCachedGlobalThreatFeed(t *testing.T) {
	g := NewGomegaWithT(t)

	name := "test-feed"
	globalThreatFeed := &v3.GlobalThreatFeed{}
	globalThreatFeed.SetName(name)
	clientInterface := &calico.MockGlobalThreatFeedInterface{
		GlobalThreatFeed: globalThreatFeed,
	}

	ctx := context.TODO()

	feedCacher := NewGlobalThreatFeedCache(name, clientInterface)
	feedCacher.Run(ctx)
	defer feedCacher.Close()

	cachedGlobalThreatFeed := feedCacher.GetGlobalThreatFeed().GlobalThreatFeed
	g.Expect(cachedGlobalThreatFeed != globalThreatFeed).To(BeTrue())
	g.Expect(cachedGlobalThreatFeed).Should(Equal(globalThreatFeed))
}

func TestGlobalThreatFeedCache_UpdateCachedGlobalThreatFeed(t *testing.T) {
	g := NewGomegaWithT(t)

	name := "test-feed"
	globalThreatFeed := &v3.GlobalThreatFeed{}
	clientInterface := &calico.MockGlobalThreatFeedInterface{
		GlobalThreatFeed: globalThreatFeed,
	}

	ctx := context.TODO()

	feedCacher := NewGlobalThreatFeedCache(name, clientInterface)
	feedCacher.Run(ctx)
	defer feedCacher.Close()

	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		cachedFeed := feedCacher.GetGlobalThreatFeed().GlobalThreatFeed
		cachedFeed.SetName("new-test-feed")
		feedCacher.UpdateGlobalThreatFeed(cachedFeed)
		wg.Done()
	}()

	var cachedGlobalThreatFeed *v3.GlobalThreatFeed
	go func() {
		time.Sleep(time.Second)
		cachedGlobalThreatFeed = feedCacher.GetGlobalThreatFeed().GlobalThreatFeed
		wg.Done()
	}()

	wg.Wait()
	g.Expect(cachedGlobalThreatFeed.Name).Should(Equal("new-test-feed"))
}

func TestGlobalThreatFeedCache_UpdateCachedGlobalThreatFeedConcurrently(t *testing.T) {
	g := NewGomegaWithT(t)

	name := "test-feed"
	globalThreatFeed := &v3.GlobalThreatFeed{}
	globalThreatFeed.SetName(name)
	clientInterface := &calico.MockGlobalThreatFeedInterface{
		GlobalThreatFeed: globalThreatFeed,
	}

	ctx := context.TODO()

	feedCacher := NewGlobalThreatFeedCache(name, clientInterface)
	feedCacher.Run(ctx)
	defer feedCacher.Close()

	wg := sync.WaitGroup{}
	wg.Add(2)

	newNameOne := "new-test-feed-1"
	newNameTwo := "new-test-feed-2"
	go func() {
		defer wg.Done()
		cachedFeed := feedCacher.GetGlobalThreatFeed().GlobalThreatFeed
		g.Expect(cachedFeed.GetName()).To(Equal(name))
		cachedFeed.SetName(newNameOne)
		feedCacher.UpdateGlobalThreatFeed(cachedFeed)
	}()

	go func() {
		defer wg.Done()
		time.Sleep(time.Second)
		cachedFeed := feedCacher.GetGlobalThreatFeed().GlobalThreatFeed
		g.Expect(cachedFeed.GetName()).To(Equal(newNameOne))
		cachedFeed.SetName(newNameTwo)
		feedCacher.UpdateGlobalThreatFeed(cachedFeed)
	}()

	wg.Wait()
	g.Expect(feedCacher.GetGlobalThreatFeed().GlobalThreatFeed.Name).Should(Equal(newNameTwo))
}

func TestGlobalThreatFeedCache_UpdateCachedGlobalThreatFeedStatus(t *testing.T) {
	g := NewGomegaWithT(t)

	name := "test-feed"
	globalThreatFeed := &v3.GlobalThreatFeed{}
	clientInterface := &calico.MockGlobalThreatFeedInterface{
		GlobalThreatFeed: globalThreatFeed,
	}

	ctx := context.TODO()

	feedCacher := NewGlobalThreatFeedCache(name, clientInterface)
	feedCacher.Run(ctx)
	defer feedCacher.Close()

	now := time.Now()
	errorConditions := []v3.ErrorCondition{{Type: "testErrType1", Message: "testErrMessage1"}, {Type: "testErrType2", Message: "testErrMessage2"}}

	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()
		cachedFeed := feedCacher.GetGlobalThreatFeed().GlobalThreatFeed
		g.Expect(cachedFeed.Status.LastSuccessfulSearch).Should(BeNil())
		g.Expect(cachedFeed.Status.LastSuccessfulSync).Should(BeNil())
		g.Expect(cachedFeed.Status.ErrorConditions).Should(BeEmpty())
		cachedFeed.Status.LastSuccessfulSearch = &metav1.Time{Time: now}
		cachedFeed.Status.LastSuccessfulSync = &metav1.Time{Time: now}
		cachedFeed.Status.ErrorConditions = errorConditions
		feedCacher.UpdateGlobalThreatFeedStatus(cachedFeed)
	}()

	var cachedGlobalThreatFeed *v3.GlobalThreatFeed
	go func() {
		time.Sleep(time.Second)
		cachedGlobalThreatFeed = feedCacher.GetGlobalThreatFeed().GlobalThreatFeed
		wg.Done()
	}()

	wg.Wait()
	g.Expect(cachedGlobalThreatFeed.Status.LastSuccessfulSearch.Time).Should(Equal(now))
	g.Expect(cachedGlobalThreatFeed.Status.LastSuccessfulSync.Time).Should(Equal(now))
	g.Expect(reflect.DeepEqual(cachedGlobalThreatFeed.Status.ErrorConditions, errorConditions)).Should(BeTrue())
}

func TestGlobalThreatFeedCache_UpdateCachedGlobalThreatFeedStatusConcurrently(t *testing.T) {
	g := NewGomegaWithT(t)

	name := "test-feed"
	globalThreatFeed := &v3.GlobalThreatFeed{}
	clientInterface := &calico.MockGlobalThreatFeedInterface{
		GlobalThreatFeed: globalThreatFeed,
	}

	ctx := context.TODO()

	feedCacher := NewGlobalThreatFeedCache(name, clientInterface)
	feedCacher.Run(ctx)
	defer feedCacher.Close()

	now := time.Now()
	oneMinuteAgo := now.Add(-1 * time.Minute)
	errorConditions := []v3.ErrorCondition{{Type: "testErrType1", Message: "testErrMessage1"}, {Type: "testErrType2", Message: "testErrMessage2"}}

	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()
		cachedFeed := feedCacher.GetGlobalThreatFeed().GlobalThreatFeed
		g.Expect(cachedFeed.Status.LastSuccessfulSearch).Should(BeNil())
		g.Expect(cachedFeed.Status.LastSuccessfulSync).Should(BeNil())
		g.Expect(cachedFeed.Status.ErrorConditions).Should(BeEmpty())
		cachedFeed.Status.LastSuccessfulSearch = &metav1.Time{Time: oneMinuteAgo}
		cachedFeed.Status.LastSuccessfulSync = &metav1.Time{Time: oneMinuteAgo}
		cachedFeed.Status.ErrorConditions = errorConditions
		feedCacher.UpdateGlobalThreatFeedStatus(cachedFeed)
	}()

	go func() {
		defer wg.Done()
		time.Sleep(time.Second)
		cachedFeed := feedCacher.GetGlobalThreatFeed().GlobalThreatFeed
		g.Expect(cachedFeed.Status.LastSuccessfulSearch.Time).Should(Equal(oneMinuteAgo))
		g.Expect(cachedFeed.Status.LastSuccessfulSync.Time).Should(Equal(oneMinuteAgo))
		g.Expect(cachedFeed.Status.ErrorConditions).Should(Equal(errorConditions))
		cachedFeed.Status.LastSuccessfulSearch = &metav1.Time{Time: now}
		cachedFeed.Status.LastSuccessfulSync = &metav1.Time{Time: now}
		cachedFeed.Status.ErrorConditions = append(errorConditions, v3.ErrorCondition{Type: "testErrType3", Message: "testErrMessage3"})
		feedCacher.UpdateGlobalThreatFeedStatus(cachedFeed)
	}()

	wg.Wait()
	cachedGlobalThreatFeed := feedCacher.GetGlobalThreatFeed().GlobalThreatFeed
	g.Expect(cachedGlobalThreatFeed.Status.LastSuccessfulSearch.Time).Should(Equal(now))
	g.Expect(cachedGlobalThreatFeed.Status.LastSuccessfulSync.Time).Should(Equal(now))
	g.Expect(reflect.DeepEqual(
		cachedGlobalThreatFeed.Status.ErrorConditions,
		append(errorConditions, v3.ErrorCondition{Type: "testErrType3", Message: "testErrMessage3"}))).Should(BeTrue())
}
