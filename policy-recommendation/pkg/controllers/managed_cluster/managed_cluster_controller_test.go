// Copyright (c) 2024 Tigera, Inc. All rights reserved.
package managed_cluster_controller

import (
	"bytes"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/policy-recommendation/utils"
)

var _ = Describe("ManagedClusterController", func() {
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
			// Create a new managed cluster logger entry
			logEntry = logger.WithField("ManagedCluster", "controller")

			watcher = newMockWatcher()

			// Create a channel to signal when the controller should stop.
			stopChan = make(chan struct{})
		})

		It("should start and stop the controller", func() {
			// Create a new managedClusterController instance
			controller := &managedClusterController{
				watcher: watcher,
				managedClusters: map[string]*managedClusterCtrlContext{
					"managed-cluster-1": {
						stopChan: make(chan struct{}),
					},
					"managed-cluster-2": {
						stopChan: make(chan struct{}),
					},
				},
				clog: logEntry,
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
				}, 10*time.Second).Should(BeClosed())
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
				}, 10*time.Second).Should(BeClosed())
			}()

			// Wait for the controller to start.
			<-controllerStarted

			// Stop the controller.
			close(stopChan)

			// Wait for the controller to stop.
			<-controllerStopped

			_, ok := controller.managedClusters["managed-cluster-1"]
			Expect(ok).To(BeFalse())
			_, ok = controller.managedClusters["managed-cluster-2"]
			Expect(ok).To(BeFalse())
		})

		It("restart the controller", func() {
			// Create a new managedClusterController instance
			controller := &managedClusterController{
				watcher: watcher,
				managedClusters: map[string]*managedClusterCtrlContext{
					"managed-cluster-1": {
						stopChan: make(chan struct{}),
					},
					"managed-cluster-2": {
						stopChan: make(chan struct{}),
					},
				},
				clog: logEntry,
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
				}, 10*time.Second).Should(BeClosed())
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
				}, 10*time.Second).Should(BeClosed())
			}()

			// Wait for the controller to start.
			<-controllerStarted

			// Stop the controller.
			close(stopChan)

			// Wait for the controller to stop.
			<-controllerStopped

			_, ok := controller.managedClusters["managed-cluster-1"]
			Expect(ok).To(BeFalse())
			_, ok = controller.managedClusters["managed-cluster-2"]
			Expect(ok).To(BeFalse())

			// Restart the controller

			// Reset the buffer
			buffer.Reset()

			// Create a channel to signal when the controller has started.
			controllerStarted = make(chan struct{})
			// Create a channel to signal when the controller has stopped.
			controllerStopped = make(chan struct{})
			// Create a new stop channel
			stopChan = make(chan struct{})
			// Create a new managedClusterController instance
			controller = &managedClusterController{
				watcher: watcher,
				managedClusters: map[string]*managedClusterCtrlContext{
					"managed-cluster-1": {
						stopChan: make(chan struct{}),
					},
					"managed-cluster-2": {
						stopChan: make(chan struct{}),
					},
				},
				clog: logEntry,
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
				}, 10*time.Second).Should(BeClosed())
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
				}, 10*time.Second).Should(BeClosed())
			}()

			// Wait for the controller to start.
			<-controllerStarted

			// Stop the controller.
			close(stopChan)

			// Wait for the controller to stop.
			<-controllerStopped

			_, ok = controller.managedClusters["managed-cluster-1"]
			Expect(ok).To(BeFalse())
			_, ok = controller.managedClusters["managed-cluster-2"]
			Expect(ok).To(BeFalse())
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
