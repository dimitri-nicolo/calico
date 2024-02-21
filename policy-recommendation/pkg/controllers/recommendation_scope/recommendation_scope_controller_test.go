// Copyright (c) 2024 Tigera, Inc. All rights reserved.
package recommendation_scope_controller

import (
	"bytes"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/policy-recommendation/utils"
)

var _ = Describe("PolicyRecommendationScopeController", func() {
	Context("Run", func() {
		var (
			buffer   *bytes.Buffer
			logEntry *log.Entry
			watcher  *mockWatcher
			stopChan chan struct{}
		)

		BeforeEach(func() {
			buffer = &bytes.Buffer{}
			// Create a new Logrus logger instance
			logger := log.New()
			// Set the logger's output to the buffer
			logger.SetOutput(buffer)
			// Create a new recommendation scope cluster logger entry
			logEntry = logger.WithField("PolicyRecommendationScope", "controller")

			watcher = newMockWatcher()

			// Create a channel to signal when the controller should stop.
			stopChan = make(chan struct{})
		})

		It("should start and stop the controller", func() {
			// Create a new recommendationScopeController instance
			controller := &recommendationScopeController{
				watcher: watcher,
				clog:    logEntry,
			}

			compareMessages := func(extractedMessages, expectedMessages []string) bool {
				// Compare the lengths first
				if len(extractedMessages) != len(expectedMessages) {
					return false
				}

				// Compare each message
				for i, msg := range extractedMessages {
					if msg != expectedMessages[i] {
						return false
					}
				}

				return true
			}

			// Create a channel to signal when the controller has started.
			controllerStarted := make(chan struct{})
			// Create a channel to signal when the controller has stopped.
			controllerStopped := make(chan struct{})

			expectedStartLogs := []string{
				"Started controller",
			}

			expectedStopLogs := []string{
				"Started controller",
				"Stopped controller",
			}

			// Start the controller in a goroutine.
			go func() {
				controller.Run(stopChan)

				// Eventually, the controller will log the expected stop message.
				Eventually(func() chan struct{} {
					if compareMessages(utils.ExtractMessages(buffer.String()), expectedStopLogs) {
						// Signal that the controller has stopped.
						close(controllerStopped)
					}
					return controllerStopped
				}, 1000*time.Second).Should(BeClosed())
			}()

			// Verify that the controller has started.
			go func() {
				// Eventually, the controller will log the expected start message.
				Eventually(func() chan struct{} {
					if compareMessages(utils.ExtractMessages(buffer.String()), expectedStartLogs) && watcher.event == "Running" {
						// Signal that the controller has started.
						close(controllerStarted)
					}
					return controllerStarted
				}, 1000*time.Second).Should(BeClosed())
			}()

			// Wait for the controller to start.
			<-controllerStarted

			// Stop the controller.
			close(stopChan)

			// Wait for the controller to stop.
			<-controllerStopped
		})

		It("should re-start the controller", func() {
			// Create a new recommendationScopeController instance
			controller := &recommendationScopeController{
				watcher: watcher,
				clog:    logEntry,
			}

			compareMessages := func(extractedMessages, expectedMessages []string) bool {
				// Compare the lengths first
				if len(extractedMessages) != len(expectedMessages) {
					return false
				}

				// Compare each message
				for i, msg := range extractedMessages {
					if msg != expectedMessages[i] {
						return false
					}
				}

				return true
			}

			// Create a channel to signal when the controller has started.
			controllerStarted := make(chan struct{})
			// Create a channel to signal when the controller has stopped.
			controllerStopped := make(chan struct{})

			expectedStartLogs := []string{
				"Started controller",
			}

			expectedStopLogs := []string{
				"Started controller",
				"Stopped controller",
			}

			// Start the controller in a goroutine.
			go func() {
				controller.Run(stopChan)

				// Eventually, the controller will log the expected stop message.
				Eventually(func() chan struct{} {
					if compareMessages(utils.ExtractMessages(buffer.String()), expectedStopLogs) {
						// Signal that the controller has stopped.
						close(controllerStopped)
					}
					return controllerStopped
				}, 1000*time.Second).Should(BeClosed())
			}()

			// Verify that the controller has started.
			go func() {
				// Eventually, the controller will log the expected start message.
				Eventually(func() chan struct{} {
					if compareMessages(utils.ExtractMessages(buffer.String()), expectedStartLogs) && watcher.event == "Running" {
						// Signal that the controller has started.
						close(controllerStarted)
					}
					return controllerStarted
				}, 1000*time.Second).Should(BeClosed())
			}()

			// Wait for the controller to start.
			<-controllerStarted

			// Stop the controller.
			close(stopChan)

			// Wait for the controller to stop.
			<-controllerStopped

			// Restart the controller

			// Reset the buffer
			buffer.Reset()

			// Create a channel to signal when the controller has re-started.
			controllerRestarted := make(chan struct{})
			// Create a channel to signal when the controller has stopped.
			controllerStopped = make(chan struct{})
			// Create a new stop channel
			stopChan = make(chan struct{})
			// Start the controller in a goroutine.
			go func() {
				controller.Run(stopChan)

				// Eventually, the controller will log the expected stop message.
				Eventually(func() chan struct{} {
					if compareMessages(utils.ExtractMessages(buffer.String()), expectedStopLogs) {
						// Signal that the controller has stopped.
						close(controllerStopped)
					}
					return controllerStopped
				}, 1000*time.Second).Should(BeClosed())
			}()

			// Verify that the controller has started.
			go func() {
				// Eventually, the controller will log the expected start message.
				Eventually(func() chan struct{} {
					if compareMessages(utils.ExtractMessages(buffer.String()), expectedStartLogs) && watcher.event == "Running" {
						// Signal that the controller has started.
						close(controllerRestarted)
					}
					return controllerRestarted
				}, 1000*time.Second).Should(BeClosed())
			}()

			// Wait for the controller to start.
			<-controllerRestarted

			// Stop the controller.
			close(stopChan)

			// Wait for the controller to stop.
			<-controllerStopped
		})
	})
})

type mockWatcher struct {
	event string
}

func (w *mockWatcher) Run(stopChan chan struct{}) {
	w.event = "Running"
}

func newMockWatcher() *mockWatcher {
	return &mockWatcher{
		event: "Not Running",
	}
}
