package cache_test

import (
	"fmt"
	"reflect"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/calico/continuous-policy-recommendation/pkg/cache"
)

var _ = Describe("Object Cache", func() {

	type testingStruct struct {
		value string
	}
	var objectCache cache.ObjectCache[*testingStruct]

	Context("Single object commands", func() {
		var testValue testingStruct
		var testKey string

		BeforeEach(func() {
			// reset the cache for every test
			objectCache = cache.NewSynchronizedObjectCache[*testingStruct]()
			testValue = testingStruct{
				value: "test",
			}

			testKey = "key"
		})

		It("gets an empty nil value if the specified key isn't stored", func() {
			object := objectCache.Get("empty-key")

			Expect(object).To(BeNil())
		})

		It("gets a value for a key holding the value", func() {
			object := objectCache.Set(testKey, &testValue)
			Expect(object.value).To(Equal(testValue.value))

			object = objectCache.Get(testKey)
			Expect(object.value).To(Equal(testValue.value))
		})

		It("deletes a value for a key holding the value", func() {
			object := objectCache.Set(testKey, &testValue)
			Expect(object.value).To(Equal(testValue.value))

			objectCache.Delete(testKey)

			object = objectCache.Get(testKey)
			Expect(reflect.TypeOf(object).String()).To(Equal("*cache_test.testingStruct"))
			Expect(reflect.ValueOf(object).IsNil()).To(BeTrue())
		})

		It("synchronization test - get from an updated value if entry is updated", func() {
			testUpdated := testingStruct{
				value: "updated",
			}

			objectCache.Set(testKey, &testValue)
			go objectCache.Set(testKey, &testUpdated)

			time.Sleep(500 * time.Millisecond)
			object := objectCache.Get(testKey)
			Expect(object.value).To(Equal(testUpdated.value))
		})

	})

	It("GetAll returns the values as slice", func() {
		objectCache = cache.NewSynchronizedObjectCache[*testingStruct]()

		length := 10
		for i := 1; i <= length; i++ {
			testValue := testingStruct{
				value: fmt.Sprintf("test%d", i),
			}

			testKey := fmt.Sprintf("key%d", i)

			object := objectCache.Set(testKey, &testValue)
			Expect(object.value).To(Equal(testValue.value))
		}

		values := objectCache.GetAll()
		Expect(len(values)).To(Equal(length))

		for i := 1; i <= length; i++ {
			expectdValue := fmt.Sprintf("test%d", i)
			testKey := fmt.Sprintf("key%d", i)

			object := objectCache.Get(testKey)
			Expect(object.value).To(Equal(expectdValue))
		}
	})

})
