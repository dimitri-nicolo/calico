// Copyright (c) 2017 Tigera, Inc. All rights reserved.
//
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

package fvtests_test

import (
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	log "github.com/Sirupsen/logrus"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/projectcalico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	calinet "github.com/projectcalico/libcalico-go/lib/net"
	. "github.com/projectcalico/typha/fv-tests"
	"github.com/projectcalico/typha/pkg/calc"
	"github.com/projectcalico/typha/pkg/snapcache"
	"github.com/projectcalico/typha/pkg/syncclient"
	"github.com/projectcalico/typha/pkg/syncproto"
	"github.com/projectcalico/typha/pkg/syncserver"
)

var (
	configFoobarBazzBiff = api.Update{
		KVPair: model.KVPair{
			Key:      model.GlobalConfigKey{Name: "foobar"},
			Value:    "bazzbiff",
			Revision: "1234",
			TTL:      12,
		},
		UpdateType: api.UpdateTypeKVNew,
	}
	configFoobarDeleted = api.Update{
		KVPair: model.KVPair{
			Key:      model.GlobalConfigKey{Name: "foobar"},
			Revision: "1235",
		},
		UpdateType: api.UpdateTypeKVDeleted,
	}
	configFoobar2BazzBiff = api.Update{
		KVPair: model.KVPair{
			Key:      model.GlobalConfigKey{Name: "foobar2"},
			Value:    "bazzbiff",
			Revision: "1237",
		},
		UpdateType: api.UpdateTypeKVNew,
	}
	// Simulates an invalid key, which we treat as a deletion.
	configFoobar2Invalid = api.Update{
		KVPair: model.KVPair{
			Key:      model.GlobalConfigKey{Name: "foobar2"},
			Revision: "1238",
		},
		UpdateType: api.UpdateTypeKVUpdated,
	}
)

// Tests that rely on starting a real Typha syncserver.Server (on a real TCP port) in this process.
// We drive the server via a real snapshot cache using the snapshot cache's function API.
var _ = Describe("With an in-process Server", func() {
	// We'll create this pipeline for updates to flow through:
	//
	//    This goroutine -> callback -chan-> validation -> snapshot -> server
	//                      decoupler        filter        cache
	//
	var (
		decoupler    *calc.SyncerCallbacksDecoupler
		valFilter    *calc.ValidationFilter
		cacheCxt     context.Context
		cacheCancel  context.CancelFunc
		cache        *snapcache.Cache
		server       *syncserver.Server
		serverCxt    context.Context
		serverCancel context.CancelFunc
	)

	// Each client we create gets recorded here for cleanup.
	type clientState struct {
		clientCxt    context.Context
		clientCancel context.CancelFunc
		client       *syncclient.SyncerClient
		recorder     *StateRecorder
	}
	var clientStates []clientState

	createClient := func(id interface{}) clientState {
		clientCxt, clientCancel := context.WithCancel(context.Background())
		recorder := NewRecorder()
		client := syncclient.New(
			fmt.Sprintf("127.0.0.1:%d", server.Port()),
			"test-version",
			fmt.Sprintf("test-host-%v", id),
			"test-info",
			recorder,
		)

		err := client.Start(clientCxt)
		Expect(err).NotTo(HaveOccurred())

		cs := clientState{
			clientCxt:    clientCxt,
			client:       client,
			clientCancel: clientCancel,
			recorder:     recorder,
		}
		return cs
	}

	createClients := func(n int) {
		clientStates = nil
		for i := 0; i < n; i++ {
			cs := createClient(i)
			clientStates = append(clientStates, cs)
		}
	}

	BeforeEach(func() {
		// Set up a pipeline:
		//
		//    This goroutine -> callback -chan-> validation -> snapshot -> server
		//                      decoupler        filter        cache
		//
		decoupler = calc.NewSyncerCallbacksDecoupler()
		cache = snapcache.New(snapcache.Config{
			// Set the batch size small so we can force new Breadcrumbs easily.
			MaxBatchSize: 10,
			// Reduce the wake up interval from the default to give us faster tear down.
			WakeUpInterval: 50 * time.Millisecond,
		})
		cacheCxt, cacheCancel = context.WithCancel(context.Background())
		valFilter = calc.NewValidationFilter(cache)
		go decoupler.SendToContext(cacheCxt, valFilter)
		server = syncserver.New(cache, syncserver.Config{
			PingInterval: 10 * time.Second,
			Port:         syncserver.PortRandom,
			DropInterval: 50 * time.Millisecond,
		})
		cache.Start(cacheCxt)
		serverCxt, serverCancel = context.WithCancel(context.Background())
		server.Start(serverCxt)
	})

	AfterEach(func() {
		for _, c := range clientStates {
			c.clientCancel()
			if c.client != nil {
				log.Info("Waiting for client to shut down.")
				c.client.Finished.Wait()
				log.Info("Done waiting for client to shut down.")
			}
		}

		serverCancel()
		log.Info("Waiting for server to shut down")
		server.Finished.Wait()
		log.Info("Done waiting for server to shut down")
		cacheCancel()
	})

	It("should choose a port", func() {
		Expect(server.Port()).ToNot(BeZero())
	})

	sendNUpdatesThenInSync := func(n int) map[string]api.Update {
		expectedEndState := map[string]api.Update{}
		decoupler.OnStatusUpdated(api.ResyncInProgress)
		for i := 0; i < n; i++ {
			update := api.Update{
				KVPair: model.KVPair{
					Key: model.GlobalConfigKey{
						Name: fmt.Sprintf("foo%v", i),
					},
					Value:    fmt.Sprintf("baz%v", i),
					Revision: fmt.Sprintf("%v", i),
				},
				UpdateType: api.UpdateTypeKVNew,
			}
			path, err := model.KeyToDefaultPath(update.Key)
			Expect(err).NotTo(HaveOccurred())
			expectedEndState[path] = update
			decoupler.OnUpdates([]api.Update{update})
		}
		decoupler.OnStatusUpdated(api.InSync)
		return expectedEndState
	}

	Describe("with a client connection", func() {
		var clientCancel context.CancelFunc
		var recorder *StateRecorder

		BeforeEach(func() {
			createClients(1)
			clientCancel = clientStates[0].clientCancel
			recorder = clientStates[0].recorder
		})

		// expectClientState asserts that the client eventually reaches the given state.  Then, it
		// simulates a second connection and check that that also converges to the given state.
		expectClientState := func(status api.SyncStatus, kvs map[string]api.Update) {
			// Wait until we reach that state.
			Eventually(recorder.Status).Should(Equal(status))
			Eventually(recorder.KVs).Should(Equal(kvs))

			// Now, a newly-connecting client should also reach the same state.
			log.Info("Starting transient client to read snapshot.")

			transientClient := createClient("transient")
			defer func() {
				log.Info("Stopping transient client.")
				transientClient.clientCancel()
				transientClient.client.Finished.Wait()
				log.Info("Stopped transient client.")
			}()
			Eventually(transientClient.recorder.Status).Should(Equal(status))
			Eventually(transientClient.recorder.KVs).Should(Equal(kvs))
		}

		It("should drop a bad KV", func() {
			// Bypass the validation filter (which also converts Nodes to HostIPs).
			cache.OnStatusUpdated(api.ResyncInProgress)
			cache.OnUpdates([]api.Update{{
				KVPair: model.KVPair{
					// NodeKeys can't be serialized right now.
					Key:      model.NodeKey{Hostname: "foobar"},
					Value:    "bazzbiff",
					Revision: "1234",
					TTL:      12,
				},
				UpdateType: api.UpdateTypeKVNew,
			}})
			cache.OnStatusUpdated(api.InSync)
			expectClientState(api.InSync, map[string]api.Update{})
		})

		It("validation should drop a bad Node", func() {
			valFilter.OnStatusUpdated(api.ResyncInProgress)
			valFilter.OnUpdates([]api.Update{{
				KVPair: model.KVPair{
					Key:      model.NodeKey{Hostname: "foobar"},
					Value:    "bazzbiff",
					Revision: "1234",
					TTL:      12,
				},
				UpdateType: api.UpdateTypeKVNew,
			}})
			valFilter.OnStatusUpdated(api.InSync)
			expectClientState(api.InSync, map[string]api.Update{})
		})

		It("validation should convert a valid Node", func() {
			valFilter.OnStatusUpdated(api.ResyncInProgress)
			valFilter.OnUpdates([]api.Update{{
				KVPair: model.KVPair{
					Key: model.NodeKey{Hostname: "foobar"},
					Value: &model.Node{
						FelixIPv4: calinet.ParseIP("10.0.0.1"),
					},
					Revision: "1234",
				},
				UpdateType: api.UpdateTypeKVNew,
			}})
			valFilter.OnStatusUpdated(api.InSync)
			expectClientState(api.InSync, map[string]api.Update{
				"/calico/v1/host/foobar/bird_ip": {
					KVPair: model.KVPair{
						Key:      model.HostIPKey{Hostname: "foobar"},
						Value:    calinet.ParseIP("10.0.0.1"),
						Revision: "1234",
					},
					UpdateType: api.UpdateTypeKVNew,
				}})
		})

		It("should pass through a KV and status", func() {
			decoupler.OnStatusUpdated(api.ResyncInProgress)
			decoupler.OnUpdates([]api.Update{configFoobarBazzBiff})
			decoupler.OnStatusUpdated(api.InSync)
			Eventually(recorder.Status).Should(Equal(api.InSync))
			expectClientState(
				api.InSync,
				map[string]api.Update{
					"/calico/v1/config/foobar": configFoobarBazzBiff,
				},
			)
		})

		It("should handle deletions", func() {
			// Create two keys, then delete them.  One of the keys happens to have a
			// default path that is the prefix of the other, just to make sure the Ctrie
			// doesn't accidentally delete the whole prefix.
			decoupler.OnUpdates([]api.Update{configFoobarBazzBiff})
			decoupler.OnUpdates([]api.Update{configFoobar2BazzBiff})
			decoupler.OnStatusUpdated(api.InSync)
			expectClientState(api.InSync, map[string]api.Update{
				"/calico/v1/config/foobar":  configFoobarBazzBiff,
				"/calico/v1/config/foobar2": configFoobar2BazzBiff,
			})
			decoupler.OnUpdates([]api.Update{configFoobarDeleted})
			expectClientState(api.InSync, map[string]api.Update{
				"/calico/v1/config/foobar2": configFoobar2BazzBiff,
			})
			decoupler.OnUpdates([]api.Update{configFoobar2Invalid})
			expectClientState(api.InSync, map[string]api.Update{})
		})

		It("should pass through many KVs", func() {
			expectedEndState := sendNUpdatesThenInSync(1000)
			expectClientState(api.InSync, expectedEndState)
		})

		It("should report the correct number of connections", func() {
			expectGaugeValue("typha_connections_active", 1.0)
		})

		It("should report the correct number of connections after killing the client", func() {
			clientCancel()
			expectGaugeValue("typha_connections_active", 0.0)
		})
	})

	Describe("with 100 client connections", func() {
		BeforeEach(func() {
			createClients(100)
		})

		// expectClientState asserts that every client eventually reaches the given state.
		expectClientStates := func(status api.SyncStatus, kvs map[string]api.Update) {
			for _, s := range clientStates {
				// Wait until we reach that state.
				Eventually(s.recorder.Status, 10*time.Second, 200*time.Millisecond).Should(Equal(status))
				Eventually(s.recorder.KVs, 10*time.Second).Should(Equal(kvs))
			}
		}

		It("should drop expected number of connections", func() {
			// Start a goroutine to watch each client and send us a message on the channel when it stops.
			finishedC := make(chan int)
			for _, s := range clientStates {
				go func(s clientState) {
					s.client.Finished.Wait()
					finishedC <- 1
				}(s)
			}

			// We start with 100 connections, set the max to 60 so we kill 40 connections.
			server.SetMaxConns(60)

			// We set the drop interval to 50ms so it should take 2-2.2 seconds (due to jitter) to drop the
			// connections.  Wait 3 seconds so that we verify that the server doesn't go on to kill any
			// more than the target.
			timeout := time.NewTimer(3 * time.Second)
			oneSec := time.NewTimer(1 * time.Second)
			numFinished := 0
		loop:
			for {
				select {
				case <-timeout.C:
					break loop
				case <-oneSec.C:
					// Check the rate is in the right ballpark: after one second we should have
					// dropped approximately 20 clients.
					Expect(numFinished).To(BeNumerically(">", 10))
					Expect(numFinished).To(BeNumerically("<", 30))
				case c := <-finishedC:
					numFinished += c
				}
			}
			// After the timeout we should have dropped exactly the right number of connections.
			Expect(numFinished).To(Equal(40))
			expectGaugeValue("typha_connections_active", 60.0)
		})

		It("should pass through a KV and status", func() {
			decoupler.OnStatusUpdated(api.ResyncInProgress)
			decoupler.OnUpdates([]api.Update{configFoobarBazzBiff})
			decoupler.OnStatusUpdated(api.InSync)
			expectClientStates(
				api.InSync,
				map[string]api.Update{
					"/calico/v1/config/foobar": configFoobarBazzBiff,
				},
			)
		})

		It("should pass through many KVs", func() {
			expectedEndState := sendNUpdatesThenInSync(1000)
			expectClientStates(api.InSync, expectedEndState)
		})

		It("should report the correct number of connections", func() {
			expectGaugeValue("typha_connections_active", 100.0)
		})

		It("should report the correct number of connections after killing the clients", func() {
			for _, c := range clientStates {
				c.clientCancel()
			}
			expectGaugeValue("typha_connections_active", 0.0)
		})

		It("with churn, it should report the correct number of connections after killing the clients", func() {
			// Generate some churn while we disconnect the clients.
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				sendNUpdatesThenInSync(1000)
				wg.Done()
			}()
			defer wg.Wait()
			for _, c := range clientStates {
				c.clientCancel()
				time.Sleep(100 * time.Microsecond)
			}
			expectGaugeValue("typha_connections_active", 0.0)
		})
	})
})

var _ = Describe("With an in-process Server with short ping timeout", func() {
	var (
		cacheCxt     context.Context
		cacheCancel  context.CancelFunc
		cache        *snapcache.Cache
		server       *syncserver.Server
		serverCxt    context.Context
		serverCancel context.CancelFunc
		serverAddr   string
	)

	BeforeEach(func() {
		cache = snapcache.New(snapcache.Config{
			// Set the batch size small so we can force new Breadcrumbs easily.
			MaxBatchSize: 10,
			// Reduce the wake up interval from the default to give us faster tear down.
			WakeUpInterval: 50 * time.Millisecond,
		})
		server = syncserver.New(cache, syncserver.Config{
			PingInterval: 100 * time.Millisecond,
			PongTimeout:  500 * time.Millisecond,
			Port:         syncserver.PortRandom,
			DropInterval: 50 * time.Millisecond,
		})
		cacheCxt, cacheCancel = context.WithCancel(context.Background())
		cache.Start(cacheCxt)
		serverCxt, serverCancel = context.WithCancel(context.Background())
		server.Start(serverCxt)
		serverAddr = fmt.Sprintf("127.0.0.1:%d", server.Port())
	})

	AfterEach(func() {
		if server != nil {
			serverCancel()
			log.Info("Waiting for server to shut down")
			server.Finished.Wait()
			log.Info("Done waiting for server to shut down")
		}
		if cache != nil {
			cacheCancel()
		}
	})

	It("should not disconnect a responsive client", func() {
		// Start a real client, which will respond correctly to pings.
		clientCxt, clientCancel := context.WithCancel(context.Background())
		recorder := NewRecorder()
		client := syncclient.New(
			serverAddr,
			"test-version",
			"test-host",
			"test-info",
			recorder,
		)
		err := client.Start(clientCxt)
		Expect(err).NotTo(HaveOccurred())
		defer func() {
			clientCancel()
			client.Finished.Wait()
		}()

		// Wait until we should have been dropped if we were unresponsive.  I.e. 1 pong timeout + 1 ping
		// interval for the check to take place.
		time.Sleep(1 * time.Second)

		// Then send an update.
		cache.OnStatusUpdated(api.InSync)
		Eventually(recorder.Status).Should(Equal(api.InSync))
	})

	Describe("with a raw connection", func() {
		var rawConn net.Conn
		var w *gob.Encoder
		var r *gob.Decoder

		BeforeEach(func() {
			var err error
			rawConn, err = net.DialTimeout("tcp", serverAddr, 10*time.Second)
			Expect(err).NotTo(HaveOccurred())

			w = gob.NewEncoder(rawConn)
			r = gob.NewDecoder(rawConn)
		})

		AfterEach(func() {
			err := rawConn.Close()
			if err != nil {
				log.WithError(err).Info("Error recorded while closing conn.")
			}
		})

		expectDisconnection := func(after time.Duration) {
			var envelope syncproto.Envelope
			startTime := time.Now()
			for {
				err := r.Decode(&envelope)
				if err != nil {
					return // Success!
				}
				if time.Since(startTime) > after {
					Fail("Client should have been disconnected by now")
				}
			}
			expectGaugeValue("typha_connections_active", 0.0)
		}

		It("should clean up if the hello doesn't get sent", func() {
			expectGaugeValue("typha_connections_active", 1.0)
			rawConn.Close()
			expectGaugeValue("typha_connections_active", 0.0)
		})

		Describe("After sending Hello", func() {
			BeforeEach(func() {
				err := w.Encode(syncproto.Envelope{
					Message: syncproto.MsgClientHello{
						Hostname: "me",
						Version:  "test",
						Info:     "test info",
					},
				})
				Expect(err).NotTo(HaveOccurred())
			})

			It("should disconnect a client that sends a nil update", func() {
				var envelope syncproto.Envelope
				err := w.Encode(envelope)
				Expect(err).NotTo(HaveOccurred())
				expectDisconnection(100 * time.Millisecond)
			})

			It("should disconnect a client that sends an unexpected update", func() {
				var envelope syncproto.Envelope
				envelope.Message = 42
				err := w.Encode(envelope)
				Expect(err).NotTo(HaveOccurred())
				expectDisconnection(100 * time.Millisecond)
			})

			It("should disconnect a client that sends a garbage update", func() {
				rawConn.Write([]byte("dsjfkldjsklfajdskjfk;dajskfjaoirefmuweioufijsdkfjkdsjkfjasd;"))
				// We don't get dropped as quickly as above because the gob decoder doesn't raise an
				// error for the above data (presumably, it's still waiting for more data to decode).
				// We should still get dropped byt he ping timeout though...
				expectDisconnection(time.Second)
			})

			It("should disconnect an unresponsive client", func() {
				done := make(chan struct{})
				pings := make(chan *syncproto.MsgPing)
				go func() {
					defer close(done)
					defer close(pings)
					for {
						var envelope syncproto.Envelope
						err := r.Decode(&envelope)
						if err != nil {
							return
						}
						if m, ok := envelope.Message.(syncproto.MsgPing); ok {
							pings <- &m
						}
					}
				}()
				timeout := time.NewTimer(1 * time.Second)
				startTime := time.Now()
				gotPing := false
				for {
					select {
					case m := <-pings:
						if m == nil {
							pings = nil
							continue
						}
						Expect(time.Since(m.Timestamp)).To(BeNumerically("<", time.Second))
						gotPing = true
					case <-done:
						// Check we didn't get dropped too soon.
						Expect(gotPing).To(BeTrue())
						Expect(time.Since(startTime)).To(BeNumerically(">=", 500*time.Millisecond))
						timeout.Stop()
						return
					case <-timeout.C:
						Fail("timed out waiting for unresponsive client to be dropped")
					}
				}
			})
		})
	})
})

func getGauge(name string) (float64, error) {
	mfs, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		return 0, err
	}
	for _, mf := range mfs {
		if mf.GetName() == name {
			return mf.Metric[0].GetGauge().GetValue(), nil
		}
	}
	return 0, errors.New("not found")
}

func expectGaugeValue(name string, value float64) {
	Eventually(func() (float64, error) {
		return getGauge(name)
	}).Should(Equal(value))
}
