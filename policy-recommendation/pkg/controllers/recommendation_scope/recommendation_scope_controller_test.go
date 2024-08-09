// Copyright (c) 2024 Tigera, Inc. All rights reserved.
package recommendation_scope_controller

import (
	"sync"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
)

var _ = Describe("PolicyRecommendationScopeController", func() {
	Context("Run PolicyRecommendationScopeController", func() {
		var (
			logEntry *log.Entry
			watcher  *mockWatcher
			stopChan chan struct{}
		)

		BeforeEach(func() {
			watcher = newMockWatcher()
			// Create a new Logrus logger instance
			logger := log.New()
			// Create a new recommendation scope cluster logger entry
			logEntry = logger.WithField("PolicyRecommendationScope", "controller")
			// Create a channel to signal when the controller should stop.
			stopChan = make(chan struct{})
		})

		It("should start and stop the controller", func() {
			// Create a new recommendationScopeController instance
			controller := &recommendationScopeController{
				watcher: watcher,
				clog:    logEntry,
				reconciler: &recommendationScopeReconciler{
					mutex: sync.Mutex{},
				},
			}

			// Verify that the controller has started.
			go func() {
				// Eventually, the controller will set the watcher to "Running".
				Eventually(func() chan struct{} {
					if watcher.event == "Running" {
						// Signal that the controller has started.
						close(stopChan)
					}
					return stopChan
				}, 10*time.Second).Should(BeClosed())
			}()

			go func() {
				// Start the controller.
				Eventually(func() chan struct{} {
					if watcher.event == "Not Running" {
						controller.Run(stopChan)
					}
					return stopChan
				}, 10*time.Second).ShouldNot(BeClosed())
			}()

			// Wait for the controller to stop.
			<-stopChan

			// Verify that the watcher has stopped.
			Eventually(func() bool {
				return watcher.event == "Not Running"
			}, 10*time.Second).Should(BeTrue())
		})
	})
})

type mockWatcher struct {
	event string
}

func (w *mockWatcher) Run(stopChan chan struct{}) {
	w.event = "Running"

	// Wait for the stop signal.
	<-stopChan

	w.event = "Not Running"
}

func newMockWatcher() *mockWatcher {
	return &mockWatcher{
		event: "Not Running",
	}
}
