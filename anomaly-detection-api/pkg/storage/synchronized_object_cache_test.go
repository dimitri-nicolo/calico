package storage_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/calico/anomaly-detection-api/pkg/storage"
)

var _ = Describe("Synchronized Object Cache test", func() {

	var objectCache storage.ObjectCache

	type testingStruct struct {
		value string
	}

	BeforeEach(func() {
		// reset the cache for every test
		objectCache = storage.NewSynchronizedObjectCache()
	})

	It("gets an empty nil value if the specified key isn't stored", func() {
		object := objectCache.Get("empty-key")

		Expect(object).To(BeNil())
	})

	It("gets a value for a key holding the value", func() {
		testValue := testingStruct{
			value: "test",
		}

		testKey := "key"

		object := objectCache.Set(testKey, testValue)
		Expect(object).To(Equal(testValue))

		object = objectCache.Get(testKey)
		Expect(object).To(Equal(testValue))
	})

	It("synchronization test - get from an updated value if entry is updated", func() {
		testValue := testingStruct{
			value: "test",
		}

		testUpdated := testingStruct{
			value: "updated",
		}

		testKey := "key"

		objectCache.Set(testKey, testValue)
		go objectCache.Set(testKey, testUpdated)

		time.Sleep(500 * time.Millisecond)
		object := objectCache.Get(testKey)
		Expect(object).To(Equal(testUpdated))
	})

})
