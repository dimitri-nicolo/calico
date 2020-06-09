// Copyright (c) 2017-2020 Tigera, Inc. All rights reserved.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package testutils

import (
	"fmt"
	"strings"
	"sync"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"

	gomegatypes "github.com/onsi/gomega/types"

	"github.com/projectcalico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
)

// Create a new SyncerTester.  This helper class implements the api.SyncerCallbacks
// and provides a number of useful methods for asserting the data that has been
// supplied on the callbacks.
func NewSyncerTester() *SyncerTester {
	return &SyncerTester{
		cache:  make(map[string]model.KVPair),
		status: UnsetSyncStatus,
	}
}

var (
	UnsetSyncStatus = api.SyncStatus(255)
)

// Encapsulates parse error details for easy handling with a single channel.
type parseError struct {
	rawKey   string
	rawValue string
}

type SyncerTester struct {
	status        api.SyncStatus
	statusChanged bool
	statusBlocker sync.WaitGroup
	updateBlocker sync.WaitGroup
	lock          sync.Mutex

	// Stored update information.
	cache       map[string]model.KVPair
	onUpdates   [][]api.Update
	updates     []api.Update
	parseErrors []parseError
}

// OnStatusUpdated updates the current status and then blocks until a call to
// ExpectStatusUpdate() has been called.
func (st *SyncerTester) OnStatusUpdated(status api.SyncStatus) {
	defer GinkgoRecover()
	st.lock.Lock()
	current := st.status
	st.status = status
	st.statusChanged = true
	st.statusBlocker.Add(1)
	st.lock.Unlock()

	// If this is not the first status event then perform additional validation on the status.
	if current != UnsetSyncStatus {
		// None of the concrete syncers that we are testing expect should have the same
		// status update repeated.  Log and panic.
		if status == current {
			log.WithField("Status", status).Panic("Duplicate identical status updates from syncer")
		}
	}

	log.Infof("Status set and blocking for ack: %s", status)

	// For statuses, this requires the consumer to explicitly expect the status updates
	// to unblock the processing.
	st.statusBlocker.Wait()
	log.Infof("OnStatusUpdated now unblocked waiting for: %s", status)

}

// OnUpdates just stores the update and asserts the state of the cache and the update.
func (st *SyncerTester) OnUpdates(updates []api.Update) {
	defer GinkgoRecover()

	func() {
		// Store the updates and onUpdates.
		st.lock.Lock()
		defer st.lock.Unlock()
		st.onUpdates = append(st.onUpdates, updates)
		for _, u := range updates {
			// Append the updates to the total set of updates.
			st.updates = append(st.updates, u)

			// Update our cache of current entries.
			k, err := model.KeyToDefaultPath(u.Key)
			Expect(err).NotTo(HaveOccurred())
			switch u.UpdateType {
			case api.UpdateTypeKVDeleted:
				log.WithFields(log.Fields{
					"Key": k,
				}).Info("Handling delete cache entry")
				Expect(st.cache).To(HaveKey(k))
				delete(st.cache, k)
			case api.UpdateTypeKVNew:
				log.WithFields(log.Fields{
					"Key":   k,
					"Value": u.KVPair.Value,
				}).Info("Handling new cache entry")
				Expect(st.cache).NotTo(HaveKey(k))
				Expect(u.Value).NotTo(BeNil())
				st.cache[k] = u.KVPair
			case api.UpdateTypeKVUpdated:
				log.WithFields(log.Fields{
					"Key":   k,
					"Value": u.KVPair.Value,
				}).Info("Handling modified cache entry")
				//var rcc v3.RemoteClusterConfiguration
				//switch t := u.Key.(type) {
				//default:
				//case model.ResourceKey:
				//	if t.Kind == v3.KindRemoteClusterConfiguration {
				//		rcc = u.Value.(v3.RemoteClusterConfiguration)
				//	}
				//}
				//Expect(st.cache).To(HaveKey(k), fmt.Sprintf("Update is %+v, %+v", k, rcc))
				Expect(st.cache).To(HaveKey(k), fmt.Sprintf("Update is %+q: %+v", k, u.KVPair.Value))
				Expect(u.Value).NotTo(BeNil(), fmt.Sprintf("Update is %+v", u))
				st.cache[k] = u.KVPair
			}

			// Check that KeyFromDefaultPath supports parsing the path again;
			// this is required for typha to support this resource.
			parsedKey := model.KeyFromDefaultPath(k)
			if !strings.HasPrefix(k, "/calico/felix/v1/remotecluster/") &&
				!strings.HasPrefix(k, "remote-cluster:") {
				Expect(parsedKey).NotTo(BeNil(), fmt.Sprintf(
					"KeyFromDefaultPath unable to parse %s, generated from %+v; typha won't support this key",
					k, u.Key))
			}
		}
	}()

	// We may need to block if the test has blocked the main event processing.
	st.updateBlocker.Wait()
}

// ParseFailed just stores the parse failure.
func (st *SyncerTester) ParseFailed(rawKey string, rawValue string) {
	st.lock.Lock()
	defer st.lock.Unlock()
	st.parseErrors = append(st.parseErrors, parseError{rawKey: rawKey, rawValue: rawValue})
}

// ExpectStatusUpdate verifies a status update message has been received.  This should only
// be called *after* a new status change has occurred.  The possible state changes are:
// WaitingForDatastore -> ResyncInProgress -> InSync -> WaitingForDatastore.
// ExpectStatusUpdate will panic if called with the same status twice in a row.
func (st *SyncerTester) ExpectStatusUpdate(status api.SyncStatus, timeout ...time.Duration) {
	log.Infof("Expecting status of: %s", status)
	cs := func() api.SyncStatus {
		st.lock.Lock()
		defer st.lock.Unlock()
		return st.status
	}
	if len(timeout) == 0 {
		Eventually(cs, 6*time.Second, time.Second).Should(Equal(status))
	} else {
		Eventually(cs, timeout[0], time.Second).Should(Equal(status))
	}
	Consistently(cs).Should(Equal(status))

	log.Infof("Status is at expected status: %s", status)

	// Get the current statusChanged status, and reset it.  Validate that the status was actually
	// updated to this state (i.e. the test code hasn't re-called this with the same status).
	st.lock.Lock()
	current := st.statusChanged
	st.statusChanged = false
	st.lock.Unlock()
	Expect(current).To(BeTrue())

	// If you hit a panic here, it's because you must have called this again with the
	// same status.
	st.statusBlocker.Done()
}

// ExpectStatusUnchanged verifies that the status has not changed since the last ExpectStatusUpdate
// call.
func (st *SyncerTester) ExpectStatusUnchanged() {
	sc := func() bool {
		st.lock.Lock()
		defer st.lock.Unlock()
		return st.statusChanged
	}
	Eventually(sc, 6*time.Second, time.Millisecond).Should(BeFalse())
	Consistently(sc).Should(BeFalse(), "Status changed unexpectedly")
}

// ExpectCacheSize verifies that the cache size is as expected.
func (st *SyncerTester) ExpectCacheSize(size int) {
	EventuallyWithOffset(1, st.CacheSnapshot, 6*time.Second, 1*time.Millisecond).Should(HaveLen(size))
	ConsistentlyWithOffset(1, st.CacheSnapshot).Should(HaveLen(size), "Cache size incorrect")
}

// ExpectData verifies that a KVPair is in the cache.  If a Revision is not supplied, then
// just the value will be compared (useful for Kubernetes node backed resources where the
// revision number is not stable).
func (st *SyncerTester) ExpectData(kvp model.KVPair) {
	st.ExpectDataSanitized(kvp, func(v interface{}) interface{} { return v })
}
func (st *SyncerTester) ExpectDataSanitized(kvp model.KVPair, sanitizer func(v interface{}) interface{}) {
	key, err := model.KeyToDefaultPath(kvp.Key)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("failed to convert key to default path: %v", kvp.Key))

	if kvp.Revision == "" {
		value := func() interface{} {
			return sanitizer(st.GetCacheValue(key))
		}
		EventuallyWithOffset(1, value, 6*time.Second, time.Millisecond).Should(Equal(kvp.Value),
			fmt.Sprintf("Timed out waiting for %v to equal expected value", key))
		ConsistentlyWithOffset(1, value).Should(Equal(kvp.Value), "KVPair data was incorrect")
	} else {
		kv := func() interface{} {
			kv := st.GetCacheKVPair(key)
			kv.Value = sanitizer(kv.Value)
			return kv
		}
		EventuallyWithOffset(1, kv, 6*time.Second, time.Millisecond).Should(Equal(kvp),
			fmt.Sprintf("Timed out waiting for %v to equal expected value @ rev %v", key, kvp.Revision))
		ConsistentlyWithOffset(1, kv).Should(Equal(kvp), "KVPair data (or revision) was incorrect")
	}
}

// ExpectPath verifies that a KVPair with a specified path is in the cache.
func (st *SyncerTester) ExpectPath(path string) {
	kv := func() interface{} {
		return st.GetCacheKVPair(path)
	}
	Eventually(kv, 6*time.Second, time.Millisecond).ShouldNot(BeNil())
	Consistently(kv).ShouldNot(BeNil())
}

// ExpectDataMatch verifies that the KV in the cache exists and matches the GomegaMatcher.
func (st *SyncerTester) ExpectValueMatches(k model.Key, match gomegatypes.GomegaMatcher) {
	key, err := model.KeyToDefaultPath(k)
	Expect(err).NotTo(HaveOccurred())

	value := func() interface{} {
		return st.GetCacheValue(key)
	}

	Eventually(value, 6*time.Second, time.Millisecond).Should(match)
	Consistently(value).Should(match)
}

// ExpectNoData verifies that a Key is not in the cache.
func (st *SyncerTester) ExpectNoData(k model.Key) {
	key, err := model.KeyToDefaultPath(k)
	Expect(err).NotTo(HaveOccurred())

	Eventually(st.CacheSnapshot).ShouldNot(HaveKey(key), fmt.Sprintf("Found key %s in cache - not expected", key))
	Consistently(st.CacheSnapshot).ShouldNot(HaveKey(key), fmt.Sprintf("Found key %s in cache - not expected", key))
}

// GetCacheValue returns the value of the KVPair from the cache.
func (st *SyncerTester) GetCacheKVPair(k string) model.KVPair {
	st.lock.Lock()
	defer st.lock.Unlock()
	return st.cache[k]
}

// GetCacheValue returns the value of the KVPair from the cache or nil if not present.
func (st *SyncerTester) GetCacheValue(k string) interface{} {
	st.lock.Lock()
	defer st.lock.Unlock()
	return st.cache[k].Value
}

// CacheSnapshot returns a copy of the cache.  The copy is made with the lock held.
func (st *SyncerTester) CacheSnapshot() map[string]model.KVPair {
	st.lock.Lock()
	defer st.lock.Unlock()
	cacheCopy := map[string]model.KVPair{}
	for k, v := range st.cache {
		cacheCopy[k] = v
	}
	return cacheCopy
}

// GetCacheEntries returns a slice of the current cache entries.
func (st *SyncerTester) GetCacheEntries() []model.KVPair {
	st.lock.Lock()
	defer st.lock.Unlock()
	es := []model.KVPair{}
	for _, e := range st.cache {
		es = append(es, e)
	}
	return es
}

// waitForNumUpdates waits up to 4s and exits if the number of stored updates is equal to
// the number of expected updates.
func (st *SyncerTester) waitForNumUpdates(expected []api.Update) {
	// Poll until we have at least the correct number of updates to check.
	nu := func() int {
		st.lock.Lock()
		defer st.lock.Unlock()
		return len(st.updates)
	}

	for i := 0; i < 20; i++ {
		if nu() == len(expected) {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// waitForNumOnUpdates waits up to 4s and exits if the number of stored OnUpdates is equal to
// the number of expected OnUpdates.
func (st *SyncerTester) waitForNumOnUpdates(expected [][]api.Update) {
	// Poll until we have at least the correct number of updates to check.
	nu := func() int {
		st.lock.Lock()
		defer st.lock.Unlock()
		return len(st.onUpdates)
	}
	for i := 0; i < 20; i++ {
		if nu() == len(expected) {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// Call to test the onUpdate events (without worrying about which specific
// OnUpdate events were received).
// This removes all updates/onUpdate events from this receiver, so that the
// next call to this just requires the next set of updates.
func (st *SyncerTester) ExpectUpdates(expected []api.Update, checkOrder bool) {
	log.Infof("Expecting updates of %v", expected)

	// Wait for the correct number of updates. If we don't get them then the expect will fail below.
	st.waitForNumUpdates(expected)

	// Extract the updates and remove the updates and onUpdates from our cache.
	st.lock.Lock()
	defer st.lock.Unlock()
	updates := st.updates
	st.updates = nil
	st.onUpdates = nil

	if checkOrder {
		Expect(updates).To(Equal(expected))
	} else {
		Expect(updates).To(ConsistOf(expected))
	}
}

// Call to test a sanitized set of onUpdate events. The supplied sanitizer should modify the supplied
// update to make comparison with the expected set of data. For example, this could remove revision
// information, UUIDs or other dynamic data that would otherwise make comparison difficult.
// This removes all updates/onUpdate events from this receiver, so that the
// next call to this just requires the next set of updates.
func (st *SyncerTester) ExpectUpdatesSanitized(expected []api.Update, checkOrder bool, sanitizer func(u *api.Update) *api.Update) {
	log.Infof("Expecting updates of %v", expected)

	// Wait for the correct number of updates. If we don't get them then the expect will fail below.
	st.waitForNumUpdates(expected)

	// TODO(casey): This is a hack - even though we've waited for the number of updates to exist, we might not have the
	// correct ones. This is because the sanitizing process below might filter some out. Wait a litle bit
	// extra just to make sure we've got everything. If we don't get them within 5 seconds, something must be wrong, right?
	time.Sleep(5 * time.Second)

	// Extract the updates and remove the updates and onUpdates from our cache.
	st.lock.Lock()
	defer st.lock.Unlock()
	updates := st.updates
	log.Infof("In updates of %v", updates)
	st.updates = nil
	st.onUpdates = nil

	remoteClusterStatus := map[string]model.RemoteClusterStatus{}

	sanitized := make([]api.Update, 0, len(expected))
	for i := range updates {
		update := sanitizer(&updates[i])
		if update != nil {
			// The timing of the felixremotecluster_e2e_test tests is such that we can
			// be retrying a connection to the remote cluster at the same time as we're
			// checking these updates; and an extra retry will generate an extra
			// duplicate RemoteClusterStatus.  Square those away by ignoring duplicate
			// connection failed updates.
			if key, ok := update.KVPair.Key.(model.RemoteClusterStatusKey); ok {
				if update.KVPair.Value != nil {
					status := update.KVPair.Value.(*model.RemoteClusterStatus)
					if status.Status == model.RemoteClusterConnectionFailed && *status == remoteClusterStatus[key.Name] {
						// Duplicate update: suppress.
						log.Infof("Suppress status: %v", *status)
						continue
					} else {
						remoteClusterStatus[key.Name] = *status
					}
				}
			}
			sanitized = append(sanitized, *update)
		}
	}

	if checkOrder {
		ExpectWithOffset(1, sanitized).To(Equal(expected))
	} else {
		var asIface []interface{}
		for _, u := range expected {
			asIface = append(asIface, u)
		}
		ExpectWithOffset(1, sanitized).To(ConsistOf(asIface...))
	}
}

// Call to test which onUpdate events were received.
// This removes all updates/onUpdate events from this receiver, so that the
// next call to this just requires the next set of updates.
//
// Note that for this function to be useful, your test code needs to have
// fine grained control over the order in which events occur.
func (st *SyncerTester) ExpectOnUpdates(expected [][]api.Update) {
	log.Infof("Expecting OnUpdates of %v", expected)

	// Wait for the correct number of OnUpdates. If we don't get them then the expect will fail below.
	st.waitForNumOnUpdates(expected)

	// Extract the onUpdates and remove the updates and onUpdates from our cache.
	st.lock.Lock()
	defer st.lock.Unlock()
	onUpdates := st.onUpdates
	st.updates = nil
	st.onUpdates = nil
	Expect(onUpdates).To(Equal(expected))
}

// Call to test the next parse error that we expect to have received.
// This removes the parse error from the receiver.
func (st *SyncerTester) ExpectParseError(key, value string) {
	log.Infof("Expecting parse error: %v=%v", key, value)
	// Poll until we have an error to check.
	ne := func() int {
		st.lock.Lock()
		defer st.lock.Unlock()
		return len(st.parseErrors)
	}
	Eventually(ne).Should(Not(BeZero()))

	// Extract the parse error and remove from our cache.
	st.lock.Lock()
	defer st.lock.Unlock()
	pe := st.parseErrors[0]
	st.parseErrors = st.parseErrors[1:]
	Expect(pe.rawKey).To(Equal(key))
	Expect(pe.rawValue).To(Equal(value))
}

// Block the update handling.
func (st *SyncerTester) BlockUpdateHandling() {
	st.updateBlocker.Add(1)
}

// Unblock the update handling.
func (st *SyncerTester) UnblockUpdateHandling() {
	st.updateBlocker.Done()
}
