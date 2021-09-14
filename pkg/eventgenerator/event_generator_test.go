// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package eventgenerator_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	cache2 "github.com/tigera/deep-packet-inspection/pkg/cache"
	"github.com/tigera/deep-packet-inspection/pkg/config"
	"github.com/tigera/deep-packet-inspection/pkg/dpiupdater"
	"github.com/tigera/deep-packet-inspection/pkg/elastic"
	"github.com/tigera/deep-packet-inspection/pkg/eventgenerator"

	es "github.com/olivere/elastic/v7"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	bapi "github.com/projectcalico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/libcalico-go/lib/net"
)

var _ = Describe("File Parser", func() {
	dpiName := "dpi-name"
	dpiNs := "dpi-ns"
	dpiKey := model.ResourceKey{
		Name:      dpiName,
		Namespace: dpiNs,
		Kind:      "DeepPacketInspection",
	}
	podName := "podname"
	podName2 := "podname2"
	wepKey := model.WorkloadEndpointKey{
		Hostname:       "127.0.0.1",
		OrchestratorID: "k8s",
		WorkloadID:     "dpi-ns/podname",
		EndpointID:     "eth0",
	}
	orgFile := "1_alert_fast.txt"
	expectedFile := "alert_fast.txt"
	elasticRetrySendInterval := 1 * time.Second
	ifaceName1 := "wepKey1-iface"

	var cfg *config.Config
	var ctx context.Context
	var mockESForwarder *elastic.MockESForwarder
	var mockESClient *elastic.MockClient
	var mockDPIUpdater *dpiupdater.MockDPIStatusUpdater

	mockClientFn := func(esCLI *es.Client, elasticIndexSuffix string) elastic.Client {
		return mockESClient
	}

	BeforeEach(func() {
		mockDPIUpdater = &dpiupdater.MockDPIStatusUpdater{}
		mockDPIUpdater.AssertExpectations(GinkgoT())
		mockESClient = &elastic.MockClient{}
		mockESClient.AssertExpectations(GinkgoT())
		ctx = context.Background()
		mockESForwarder = &elastic.MockESForwarder{}
		mockESForwarder.AssertExpectations(GinkgoT())
		cfg = &config.Config{SnortAlertFileBasePath: "test"}

		//Cleanup
		path := fmt.Sprintf("%s/%s/%s/%s", cfg.SnortAlertFileBasePath, dpiKey.Namespace, dpiKey.Name, podName)
		_ = os.RemoveAll(path)
		_ = os.MkdirAll(path, os.ModePerm)

		path = fmt.Sprintf("%s/%s/%s/%s", cfg.SnortAlertFileBasePath, dpiKey.Namespace, dpiKey.Name, podName2)
		_ = os.RemoveAll(path)
		_ = os.MkdirAll(path, os.ModePerm)
	})

	AfterEach(func() {
		//Cleanup
		path := fmt.Sprintf("%s/%s/%s/%s", cfg.SnortAlertFileBasePath, dpiKey.Namespace, dpiKey.Name, podName)
		_ = os.RemoveAll(path)
		_ = os.MkdirAll(path, os.ModePerm)

		path = fmt.Sprintf("%s/%s/%s/%s", cfg.SnortAlertFileBasePath, dpiKey.Namespace, dpiKey.Name, podName2)
		_ = os.RemoveAll(path)
		_ = os.MkdirAll(path, os.ModePerm)
	})

	It("should start tailing alert file, parse and send it to ElasticSearch", func() {
		// Copy and create an alert file
		path := fmt.Sprintf("%s/%s/%s/%s", cfg.SnortAlertFileBasePath, dpiKey.Namespace, dpiKey.Name, podName)
		copyAlertFile(path, orgFile, expectedFile)

		esDoc := elastic.Doc{
			Alert:           "dpi.dpi-ns/dpi-name",
			Time:            1630343977,
			Type:            "alert",
			Host:            "",
			SourceIP:        "74.125.124.100",
			SourcePort:      "9090",
			SourceName:      "",
			SourceNamespace: "",
			DestIP:          "10.28.0.13",
			DestPort:        "",
			DestName:        "podname",
			DestNamespace:   "dpi-ns",
			Description:     "Signature Triggered Alert",
			Severity:        100,
			Record:          elastic.Record{SnortSignatureID: "1000005", SnortSignatureRevision: "1", SnortAlert: "21/08/30-17:19:37.337831 [**] [1:1000005:1] \"msg:1_alert_fast\" [**] [Priority: 0] {ICMP} 74.125.124.100:9090 -> 10.28.0.13"},
		}
		docID := fmt.Sprintf("%s_%s_1630343977337831000_%s_%s_%s_%s_%s", dpiKey.Namespace, dpiKey.Name, esDoc.SourceIP, esDoc.SourcePort, esDoc.DestIP, esDoc.DestPort, esDoc.Host)
		mockESForwarder.On("Forward", elastic.EventData{ID: docID, Doc: esDoc}).Return(nil).Times(1)

		// GenerateEventsForWEP should parse file and call elastic service.
		wepCache := cache2.NewWEPCache()
		wepCache.Update(bapi.UpdateTypeKVNew, model.KVPair{
			Key: wepKey,
			Value: &model.WorkloadEndpoint{
				IPv4Nets: []net.IPNet{mustParseNet("10.28.0.13/32")},
			},
		})
		r := eventgenerator.NewEventGenerator(cfg, mockESForwarder, mockDPIUpdater, dpiKey, wepCache)
		r.GenerateEventsForWEP(wepKey)
		Eventually(func() int { return len(mockESForwarder.Calls) }, 5*time.Second).Should(Equal(1))

		// StopGeneratingEventsForWEP should delete the alert file after parsing all alerts
		r.StopGeneratingEventsForWEP(wepKey)
		Eventually(func() error {
			_, err := os.Stat(fmt.Sprintf("%s/%s", path, expectedFile))
			return err
		}, 5*time.Second).Should(HaveOccurred())

		_, err := os.Stat(fmt.Sprintf("%s/%s", path, expectedFile))
		Expect(os.IsNotExist(err)).Should(BeTrue())
	})

	It("should stop tailing alert file on reaching EOF if snort is no longer running", func() {
		esForwarder, err := elastic.NewESForwarder(cfg, mockClientFn, elasticRetrySendInterval)
		esForwarder.Run(ctx)
		Expect(err).ShouldNot(HaveOccurred())

		mockDPIUpdater.On("UpdateStatusWithError", mock.Anything, mock.Anything, false, mock.Anything).Return(nil).Times(1)

		// Copy and create an alert file
		path := fmt.Sprintf("%s/%s/%s/%s", cfg.SnortAlertFileBasePath, dpiKey.Namespace, dpiKey.Name, podName)
		copyAlertFile(path, "2_alert_fast.txt", expectedFile)

		numberOfCallsToSend := 0
		totalAlertsInFile := 14
		mockESClient.On("Upsert", mock.Anything, mock.Anything, mock.Anything).Return().Run(
			func(args mock.Arguments) {
				numberOfCallsToSend++
				for _, c := range mockESClient.ExpectedCalls {
					if c.Method == "Upsert" {
						c.ReturnArguments = mock.Arguments{nil}
					}
				}
			}).Times(totalAlertsInFile)

		wepCache := cache2.NewWEPCache()
		r := eventgenerator.NewEventGenerator(cfg, esForwarder, mockDPIUpdater, dpiKey, wepCache)
		r.GenerateEventsForWEP(wepKey)
		Eventually(func() int { return len(mockESClient.Calls) }, 10*time.Second).Should(BeNumerically(">", 2))

		// StopGeneratingEventsForWEP should delete the alert file after parsing all alerts
		r.StopGeneratingEventsForWEP(wepKey)
		Eventually(func() error {
			_, err := os.Stat(fmt.Sprintf("%s/%s", path, expectedFile))
			return err
		}, 1*time.Second).Should(HaveOccurred())

		_, err = os.Stat(fmt.Sprintf("%s/%s", path, expectedFile))
		Expect(os.IsNotExist(err)).Should(BeTrue())
	})

	It("should send pod name and namespace in Alert when available", func() {
		// Copy and create an alert file
		path := fmt.Sprintf("%s/%s/%s/%s", cfg.SnortAlertFileBasePath, dpiKey.Namespace, dpiKey.Name, podName)
		copyAlertFile(path, orgFile, expectedFile)
		cfg.NodeName = "node0"

		esDoc := elastic.Doc{
			Alert:           "dpi.dpi-ns/dpi-name",
			Time:            1630343977,
			Type:            "alert",
			Host:            cfg.NodeName,
			SourceIP:        "74.125.124.100",
			SourcePort:      "9090",
			SourceName:      "",
			SourceNamespace: "",
			DestIP:          "10.28.0.13",
			DestPort:        "",
			DestName:        podName,
			DestNamespace:   dpiNs,
			Description:     "Signature Triggered Alert",
			Severity:        100,
			Record:          elastic.Record{SnortSignatureID: "1000005", SnortSignatureRevision: "1", SnortAlert: "21/08/30-17:19:37.337831 [**] [1:1000005:1] \"msg:1_alert_fast\" [**] [Priority: 0] {ICMP} 74.125.124.100:9090 -> 10.28.0.13"},
		}
		docID := fmt.Sprintf("%s_%s_1630343977337831000_%s_%s_%s_%s_%s", dpiKey.Namespace, dpiKey.Name, esDoc.SourceIP, esDoc.SourcePort, esDoc.DestIP, esDoc.DestPort, esDoc.Host)
		mockESForwarder.On("Forward", elastic.EventData{Doc: esDoc, ID: docID}).Return(nil).Times(1)

		wepCache := cache2.NewWEPCache()
		r := eventgenerator.NewEventGenerator(cfg, mockESForwarder, mockDPIUpdater, dpiKey, wepCache)

		wepCache.Update(bapi.UpdateTypeKVNew,
			model.KVPair{
				Key: wepKey,
				Value: &model.WorkloadEndpoint{
					Name:     ifaceName1,
					IPv4Nets: []net.IPNet{mustParseNet("10.28.0.13/32")},
				},
			})
		r.GenerateEventsForWEP(wepKey)
		Eventually(func() int { return len(mockESForwarder.Calls) }, 5*time.Second).Should(Equal(1))

		// StopGeneratingEventsForWEP should delete the alert file after parsing all alerts
		r.StopGeneratingEventsForWEP(wepKey)
		Eventually(func() error {
			_, err := os.Stat(fmt.Sprintf("%s/%s", path, expectedFile))
			return err
		}, 5*time.Second).Should(HaveOccurred())

		_, err := os.Stat(fmt.Sprintf("%s/%s", path, expectedFile))
		Expect(os.IsNotExist(err)).Should(BeTrue())
	})

	It("should send current pod name and namespace in Alert", func() {
		// Copy and create an alert file
		path := fmt.Sprintf("%s/%s/%s/%s", cfg.SnortAlertFileBasePath, dpiKey.Namespace, dpiKey.Name, podName)
		copyAlertFile(path, orgFile, expectedFile)
		cfg.NodeName = "node0"

		esDoc1 := elastic.Doc{
			Alert:           "dpi.dpi-ns/dpi-name",
			Time:            1630343977,
			Type:            "alert",
			Host:            cfg.NodeName,
			SourceIP:        "74.125.124.100",
			SourceName:      "",
			SourceNamespace: "",
			DestIP:          "10.28.0.13",
			DestName:        podName,
			DestNamespace:   dpiNs,
			Description:     "Signature Triggered Alert",
			Severity:        100,
			Record:          elastic.Record{SnortSignatureID: "1000005", SnortSignatureRevision: "1", SnortAlert: "21/08/30-17:19:37.337831 [**] [1:1000005:1] \"msg:1_alert_fast\" [**] [Priority: 0] {ICMP} 74.125.124.100 -> 10.28.0.13"},
		}
		docID1 := fmt.Sprintf("%s_%s_1630343977337831000_%s_%s_%s", dpiKey.Namespace, dpiKey.Name, esDoc1.SourceIP, esDoc1.DestIP, esDoc1.Host)

		esDoc2 := elastic.Doc{
			Alert:           "dpi.dpi-ns/dpi-name",
			Time:            1630343977,
			Type:            "alert",
			Host:            cfg.NodeName,
			SourceIP:        "74.125.124.100",
			SourceName:      "",
			SourceNamespace: "",
			DestIP:          "10.28.0.13",
			DestName:        "",
			DestNamespace:   "dpiNs",
			Description:     "Signature Triggered Alert",
			Severity:        100,
			Record:          elastic.Record{SnortSignatureID: "1000005", SnortSignatureRevision: "1", SnortAlert: "21/08/30-17:19:37.337831 [**] [1:1000005:1] \"msg:1_alert_fast\" [**] [Priority: 0] {ICMP} 74.125.124.100 -> 10.28.0.13"},
		}
		docID2 := fmt.Sprintf("%s_%s_1630343977337831000_%s_%s_%s", dpiKey.Namespace, dpiKey.Name, esDoc2.SourceIP, esDoc2.DestIP, esDoc2.Host)

		numberOfCallsToSend := 0
		mockESForwarder.On("Forward", mock.Anything).Run(
			func(args mock.Arguments) {
				numberOfCallsToSend++
				for _, c := range mockESClient.ExpectedCalls {
					if c.Method == "Forward" {
						if numberOfCallsToSend <= 1 {
							Expect(c.Arguments.Get(1)).Should(BeEquivalentTo(esDoc1))
							Expect(c.Arguments.Get(2)).Should(BeEquivalentTo(docID1))
						} else {
							Expect(c.Arguments.Get(1)).Should(BeEquivalentTo(esDoc2))
							Expect(c.Arguments.Get(2)).Should(BeEquivalentTo(docID2))
						}
					}
				}
			}).Return(nil, false, nil).Times(2)

		wepCache := cache2.NewWEPCache()
		r := eventgenerator.NewEventGenerator(cfg, mockESForwarder, mockDPIUpdater, dpiKey, wepCache)

		wepCache.Update(bapi.UpdateTypeKVNew,
			model.KVPair{
				Key: wepKey,
				Value: &model.WorkloadEndpoint{
					Name:     ifaceName1,
					IPv4Nets: []net.IPNet{mustParseNet("10.28.0.13/32")},
				},
			})
		r.GenerateEventsForWEP(wepKey)
		Eventually(func() int { return len(mockESForwarder.Calls) }, 5*time.Second).Should(Equal(1))

		// StopGeneratingEventsForWEP should delete the alert file after parsing all alerts
		r.StopGeneratingEventsForWEP(wepKey)
		Eventually(func() error {
			_, err := os.Stat(fmt.Sprintf("%s/%s", path, expectedFile))
			return err
		}, 5*time.Second).Should(HaveOccurred())

		_, err := os.Stat(fmt.Sprintf("%s/%s", path, expectedFile))
		Expect(os.IsNotExist(err)).Should(BeTrue())

		By("Deleting the WEP key sends podName and namespace as empty string")
		wepCache.Update(bapi.UpdateTypeKVDeleted,
			model.KVPair{
				Key: wepKey,
			})

		copyAlertFile(path, orgFile, expectedFile)

		r.GenerateEventsForWEP(wepKey)
		Eventually(func() int { return len(mockESForwarder.Calls) }, 5*time.Second).Should(Equal(2))

		// StopGeneratingEventsForWEP should delete the alert file after parsing all alerts
		r.StopGeneratingEventsForWEP(wepKey)
		Eventually(func() error {
			_, err := os.Stat(fmt.Sprintf("%s/%s", path, expectedFile))
			return err
		}, 5*time.Second).Should(HaveOccurred())

		_, err = os.Stat(fmt.Sprintf("%s/%s", path, expectedFile))
		Expect(os.IsNotExist(err)).Should(BeTrue())
	})

	It("should handle multiple snorts producing alerts", func() {
		// Copy and create an alert file
		path1 := fmt.Sprintf("%s/%s/%s/%s", cfg.SnortAlertFileBasePath, dpiKey.Namespace, dpiKey.Name, podName)
		copyAlertFile(path1, orgFile, expectedFile)

		path2 := fmt.Sprintf("%s/%s/%s/%s", cfg.SnortAlertFileBasePath, dpiKey.Namespace, dpiKey.Name, podName2)
		copyAlertFile(path2, orgFile, expectedFile)

		wepKey2 := model.WorkloadEndpointKey{
			Hostname:       "127.0.0.1",
			OrchestratorID: "k8s",
			WorkloadID:     "dpi-ns/podname2",
			EndpointID:     "eth0",
		}

		mockESForwarder.On("Forward", mock.Anything, mock.Anything, mock.Anything).Return(nil, false, nil).Times(2)
		wepCache := cache2.NewWEPCache()
		r := eventgenerator.NewEventGenerator(cfg, mockESForwarder, mockDPIUpdater, dpiKey, wepCache)
		r.GenerateEventsForWEP(wepKey)
		r.GenerateEventsForWEP(wepKey2)

		Eventually(func() int { return len(mockESForwarder.Calls) }, 5*time.Second).Should(Equal(2))

		// StopGeneratingEventsForWEP should delete the alert file after parsing all alerts
		r.Close()

		Eventually(func() error {
			_, err := os.Stat(fmt.Sprintf("%s/%s", path1, expectedFile))
			return err
		}, 5*time.Second).Should(HaveOccurred())

		_, err := os.Stat(fmt.Sprintf("%s/%s", path1, expectedFile))
		Expect(os.IsNotExist(err)).Should(BeTrue())

		Eventually(func() error {
			_, err := os.Stat(fmt.Sprintf("%s/%s", path2, expectedFile))
			return err
		}, 5*time.Second).Should(HaveOccurred())

		_, err = os.Stat(fmt.Sprintf("%s/%s", path2, expectedFile))
		Expect(os.IsNotExist(err)).Should(BeTrue())
	})

	It("should process all previous leftover files during startup", func() {
		esForwarder, err := elastic.NewESForwarder(cfg, mockClientFn, elasticRetrySendInterval)
		esForwarder.Run(ctx)
		Expect(err).ShouldNot(HaveOccurred())

		mockDPIUpdater.On("UpdateStatusWithError", mock.Anything, mock.Anything, false, mock.Anything).Return(nil).Times(1)

		// Copy and create an alert file
		path1 := fmt.Sprintf("%s/%s/%s/%s", cfg.SnortAlertFileBasePath, dpiKey.Namespace, dpiKey.Name, podName)
		copyAlertFile(path1, orgFile, expectedFile)
		copyAlertFile(path1, "2_alert_fast.txt", "alert_fast.txt.1631063433")
		copyAlertFile(path1, "3_alert_fast.txt", "alert_fast.txt.1731063433")
		copyAlertFile(path1, "4_alert_fast.txt", "alert_fast.txt.1831063433")

		mockESClient.On("Upsert", mock.Anything, mock.Anything, mock.Anything).Return(nil).Times(20)

		wepCache := cache2.NewWEPCache()
		r := eventgenerator.NewEventGenerator(cfg, esForwarder, mockDPIUpdater, dpiKey, wepCache)
		r.GenerateEventsForWEP(wepKey)
		Eventually(func() int { return len(mockESClient.Calls) }, 5*time.Second).Should(Equal(20))

		// StopGeneratingEventsForWEP should delete the alert file after parsing all alerts
		r.StopGeneratingEventsForWEP(wepKey)
		Eventually(func() error {
			if _, err := os.Stat(fmt.Sprintf("%s/%s", path1, expectedFile)); err != nil {
				return err
			} else if _, err := os.Stat(fmt.Sprintf("%s/%s", path1, "alert_fast.txt.1631063433")); err != nil {
				return err
			} else if _, err := os.Stat(fmt.Sprintf("%s/%s", path1, "alert_fast.txt.1731063433")); err != nil {
				return err
			} else if _, err := os.Stat(fmt.Sprintf("%s/%s", path1, "alert_fast.txt.1831063433")); err != nil {
				return err
			}
			return nil
		}, 5*time.Second).Should(HaveOccurred())

		_, err = os.Stat(fmt.Sprintf("%s/%s", path1, expectedFile))
		Expect(os.IsNotExist(err)).Should(BeTrue())

	})
})

func copyAlertFile(path, src, dst string) {
	input, err := ioutil.ReadFile(fmt.Sprintf("test/data/%s", src))
	Expect(err).ShouldNot(HaveOccurred())
	err = ioutil.WriteFile(fmt.Sprintf("%s/%s", path, dst), input, 0644)
	Expect(err).ShouldNot(HaveOccurred())
}

func mustParseNet(n string) net.IPNet {
	_, cidr, err := net.ParseCIDR(n)
	Expect(err).ShouldNot(HaveOccurred())
	return *cidr
}
