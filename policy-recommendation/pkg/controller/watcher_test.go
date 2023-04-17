package controller_test

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/projectcalico/calico/policy-recommendation/pkg/controller"
)

const (
	testResourceName      = "TestName"
	testResourceNamespace = "TestNamespace"
)

var _ = Describe("Watcher", func() {
	Context("Successful Reconcile Tests", func() {
		var _ = DescribeTable("Watcher Reconcile Tests",
			func(eventType watch.EventType) {
				nameChan := make(chan types.NamespacedName)
				defer close(nameChan)

				r := &MockReconciler{
					r: func(name types.NamespacedName) error {
						nameChan <- name
						return nil
					},
				}

				var listFunc func() (runtime.Object, error)
				// If the eventType is either Modified or Deleted the resource must already exist for the event to be triggered
				if eventType == watch.Modified || eventType == watch.Deleted {
					listFunc = func() (runtime.Object, error) {
						return &corev1.SecretList{Items: []corev1.Secret{{
							ObjectMeta: metav1.ObjectMeta{
								Name:      testResourceName,
								Namespace: testResourceNamespace,
							},
						}}}, nil
					}
				} else {
					listFunc = func() (runtime.Object, error) { return &corev1.SecretList{}, nil }
				}

				mockLW := NewMockListerWatcher(listFunc)
				defer mockLW.Stop()

				w := controller.NewWatcher(r, mockLW, &corev1.Secret{})

				stop := make(chan struct{})
				defer close(stop)

				go w.Run(stop)

				mockLW.AddEvent(watch.Event{
					Type: eventType,
					Object: &corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      testResourceName,
							Namespace: testResourceNamespace,
						},
					},
				})

				// how a reconcile receives the watched object
				var name types.NamespacedName

				select {
				case name = <-nameChan:
				case <-time.NewTicker(500 * time.Millisecond).C:
					Fail("timeout waiting for name")
				}

				Expect(name).Should(Equal(types.NamespacedName{Name: testResourceName, Namespace: testResourceNamespace}))

			},
			Entry("new resource is added", watch.Added),
			Entry("resource is updated", watch.Modified),
			Entry("resource is deleted", watch.Deleted),
		)
	})

	Context("Unsuccessful Reconcile Error handling", func() {
		It("does not run reconcile after the max requests is reached", func() {
			nameChan := make(chan types.NamespacedName, 5)
			defer close(nameChan)

			var count int32
			var mu sync.Mutex

			mockReconciler := &MockReconciler{
				r: func(name types.NamespacedName) error {
					mu.Lock()
					defer mu.Unlock()
					nameChan <- name
					atomic.AddInt32(&count, 1)
					// trigger error every run
					return errors.New("error")
				},
			}

			listFunc := func() (runtime.Object, error) { return &corev1.SecretList{}, nil }

			mockLW := NewMockListerWatcher(listFunc)
			defer mockLW.Stop()

			w := controller.NewWatcher(mockReconciler, mockLW, &corev1.Secret{})

			stop := make(chan struct{})
			defer close(stop)

			go w.Run(stop)

			deployOverMaxRequests := controller.DefaultMaxRequeueAttempts + 5
			for i := 0; i < deployOverMaxRequests; i++ {
				mockLW.AddEvent(watch.Event{
					Type: watch.Added,
					Object: &corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      testResourceName,
							Namespace: testResourceNamespace,
						},
					},
				})
			}

			var name types.NamespacedName

		done:
			for {
				select {
				case name = <-nameChan:
					Expect(name).Should(Equal(types.NamespacedName{
						Name:      testResourceName,
						Namespace: testResourceNamespace,
					}))
				case <-time.NewTicker(500 * time.Millisecond).C: // wait 500 ms
					break done
				}
			}

			mu.Lock()
			defer mu.Unlock()
			// Reconciled should be run iterations [0,DefaultMaxRequeueAttempts+1] < deployOverMaxRequests
			Expect(count).To(Equal(int32(controller.DefaultMaxRequeueAttempts + 2)))

		})
	})
})

type MockReconciler struct {
	r func(name types.NamespacedName) error
	c func()
}

func (m *MockReconciler) Reconcile(name types.NamespacedName) error {
	return m.r(name)
}

func (m *MockReconciler) Close() {
	m.c()
}

type MockWatch struct {
	ch chan watch.Event
}

func (m *MockWatch) ResultChan() <-chan watch.Event {
	return m.ch
}

func (m *MockWatch) Stop() {
}

type MockListerWatcher struct {
	listFunc func() (runtime.Object, error)
	eventCh  chan watch.Event
}

func NewMockListerWatcher(listFunc func() (runtime.Object, error)) *MockListerWatcher {
	return &MockListerWatcher{
		listFunc: listFunc,
		eventCh:  make(chan watch.Event),
	}
}

func (m *MockListerWatcher) List(options metav1.ListOptions) (runtime.Object, error) {
	if m.listFunc != nil {
		return m.listFunc()
	}

	return nil, nil
}

func (m *MockListerWatcher) Watch(options metav1.ListOptions) (watch.Interface, error) {
	return &MockWatch{m.eventCh}, nil
}

func (m *MockListerWatcher) AddEvent(event watch.Event) {
	m.eventCh <- event
}

func (m *MockListerWatcher) Stop() {
	close(m.eventCh)
}
