// Copyright (c) 2020-2021 Tigera, Inc. All rights reserved.

package capture_test

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/projectcalico/felix/proto"

	"github.com/projectcalico/felix/capture"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("PacketCapture Capture Tests", func() {
	const podName = "test"
	const deviceName = "eth0"
	const namespace = "ns"
	const name = "capture"

	var currentOrderTwoFileSizeOnePacket = outputFile{
		Name:  fmt.Sprintf("%s_%s.pcap", podName, deviceName),
		Size:  capture.GlobalHeaderLen + dummyPacketDataSize() + capture.PacketInfoLen,
		Order: 2,
	}
	var currentOrderTwoFileSizeTwoPackets = outputFile{
		Name:  fmt.Sprintf("%s_%s.pcap", podName, deviceName),
		Size:  capture.GlobalHeaderLen + 2*(dummyPacketDataSize()+capture.PacketInfoLen),
		Order: 2,
	}

	var currentFileOrderOneSizeFivePackets = outputFile{
		Name:  fmt.Sprintf("%s_%s.pcap", podName, deviceName),
		Size:  capture.GlobalHeaderLen + 5*(dummyPacketDataSize()+capture.PacketInfoLen),
		Order: 1,
	}

	var rotatedFileOrderOneSizeOnePacket = outputFile{
		Name:  fmt.Sprintf("%s_%s.[\\d]+.pcap", podName, deviceName),
		Size:  capture.GlobalHeaderLen + dummyPacketDataSize() + capture.PacketInfoLen,
		Order: 1,
	}

	var rotatedFileOrderZeroSizeOnePacket = outputFile{
		Name:  fmt.Sprintf("%s_%s.[\\d]+.pcap", podName, deviceName),
		Size:  capture.GlobalHeaderLen + dummyPacketDataSize() + capture.PacketInfoLen,
		Order: 0,
	}

	var rotatedFileOrderZeroSizeFivePackets = outputFile{
		Name:  fmt.Sprintf("%s_%s.[\\d]+.pcap", podName, deviceName),
		Size:  capture.GlobalHeaderLen + 5*(dummyPacketDataSize()+capture.PacketInfoLen),
		Order: 0,
	}

	var baseDir string
	var captureDir string

	BeforeEach(func() {
		var err error

		baseDir, err = ioutil.TempDir("/tmp", "pcap-tests")
		Expect(err).NotTo(HaveOccurred())
		captureDir = fmt.Sprintf("%s/%s/%s", baseDir, namespace, name)
	})

	AfterEach(func() {
		var err = os.RemoveAll(baseDir)
		Expect(err).NotTo(HaveOccurred())
	})

	It("Writes 1 packet in a pcap file", func(done Done) {
		defer close(done)
		var wg sync.WaitGroup
		var err error
		var numberOfPackets = 1
		var updates = make(chan interface{}, 100)
		defer close(updates)

		// Initialise a new capture
		var pcap capture.PcapFile
		var packets = make(chan gopacket.Packet)
		defer close(packets)
		pcap = capture.NewRotatingPcapFile(baseDir, namespace, name, podName, deviceName, updates)
		defer pcap.Done()

		// Capture listens to incoming packets
		go func() {
			defer GinkgoRecover()

			err = pcap.Write(packets)
			Expect(err).NotTo(HaveOccurred())
		}()

		// Write 1 packet
		wg.Add(numberOfPackets)
		go func() {
			var packet = dummyPacket()

			packets <- packet
			wg.Done()
		}()

		// Wait for all the packets to be written to file
		wg.Wait()

		// Define expected files
		var expectedFiles = []outputFile{
			currentOrderTwoFileSizeOnePacket,
		}

		// Assert written files on disk
		assertPcapFiles(captureDir, expectedFiles)

		// Assert that an update was sent
		var update *proto.PacketCaptureStatusUpdate
		Eventually(updates).Should(Receive(&update))
		assertStatusUpdates(update, expectedFiles, namespace, name)
	}, 10)

	It("Writes 10 packet in a pcap file", func(done Done) {
		defer close(done)
		var wg sync.WaitGroup
		var err error
		var numberOfPackets = 10
		var updates = make(chan interface{}, 100)
		defer close(updates)

		// Initialise a new capture
		var pcap capture.PcapFile
		var packets = make(chan gopacket.Packet)
		defer close(packets)
		pcap = capture.NewRotatingPcapFile(baseDir, namespace, name, podName, deviceName, updates)
		defer pcap.Done()

		// Capture listens to incoming packets
		go func() {
			defer GinkgoRecover()

			err = pcap.Write(packets)
			Expect(err).NotTo(HaveOccurred())
		}()

		// Write 10 packets
		wg.Add(numberOfPackets)
		go func() {

			for i := 0; i < numberOfPackets; i++ {
				packet := dummyPacket()

				packets <- packet
				wg.Done()
			}
		}()

		// Wait for all the packets to be written to file
		wg.Wait()

		// Define expected files
		var expectedFiles = []outputFile{
			{
				Name: fmt.Sprintf("%s_%s.pcap", podName, deviceName),
				Size: capture.GlobalHeaderLen + numberOfPackets*(dummyPacketDataSize()+capture.PacketInfoLen),
			},
		}

		// Assert written files on disk
		assertPcapFiles(captureDir, expectedFiles)

		// Assert that an update was sent
		var update *proto.PacketCaptureStatusUpdate
		Eventually(updates).Should(Receive(&update))
		assertStatusUpdates(update, expectedFiles, namespace, name)
	}, 10)

	/* TODO: https://tigera.atlassian.net/browse/SAAS-1540
	It("Rotates pcap files using size", func(done Done) {
		defer close(done)
		var wg sync.WaitGroup
		var err error
		var numberOfPackets = 3
		var maxSize = capture.GlobalHeaderLen + (dummyPacketDataSize() + capture.PacketInfoLen)
		var updates = make(chan interface{})
		defer close(updates)

		// Initialise a new capture
		var pcap capture.PcapFile
		var packets = make(chan gopacket.Packet)
		defer close(packets)
		pcap = capture.NewRotatingPcapFile(baseDir, namespace, name, podName, deviceName, updates,
			capture.WithMaxSizeBytes(maxSize))
		defer pcap.Done()

		// Capture listens to incoming packets
		go func() {
			defer GinkgoRecover()

			err = pcap.Write(packets)
			Expect(err).NotTo(HaveOccurred())
		}()

		// Write 10 packets
		wg.Add(numberOfPackets)
		go func() {
			for i := 0; i < numberOfPackets; i++ {
				packet := dummyPacket()

				packets <- packet
				wg.Done()
			}
		}()

		// Wait for all the packets to be written to file
		wg.Wait()

		// Define expected files
		var expectedFiles = []outputFile{
			currentOrderTwoFileSizeOnePacket,
			rotatedFileOrderOneSizeOnePacket,
			rotatedFileOrderZeroSizeOnePacket,
		}

		// Assert written files on disk
		assertPcapFiles(captureDir, expectedFiles)

		// Assert that three updates were sent
		var update = make([]*proto.PacketCaptureStatusUpdate, 3)
		Eventually(updates).Should(Receive(&update[0]))
		Eventually(updates).Should(Receive(&update[1]))
		Eventually(updates).Should(Receive(&update[2]))
		assertStatusUpdates(update[0], []outputFile{currentOrderTwoFileSizeOnePacket}, namespace, name)
		assertStatusUpdates(update[1], []outputFile{currentOrderTwoFileSizeOnePacket, rotatedFileOrderOneSizeOnePacket}, namespace, name)
		assertStatusUpdates(update[2], expectedFiles, namespace, name)
	}, 10)*/

	It("Rotates pcap files using time", func(done Done) {
		defer close(done)
		var wg sync.WaitGroup
		var err error
		var maxAge = 1

		var timeChan = make(chan time.Time)
		var ticker = &time.Ticker{C: timeChan}
		var updates = make(chan interface{}, 100)
		defer close(updates)

		// Initialise a new capture
		var pcap capture.PcapFile
		var packets = make(chan gopacket.Packet)
		defer close(packets)
		pcap = capture.NewRotatingPcapFile(baseDir, namespace, name, podName, deviceName,
			updates,
			capture.WithRotationSeconds(maxAge),
			capture.WithTicker(ticker),
		)
		defer pcap.Done()

		// Capture listens to incoming packets
		go func() {
			defer GinkgoRecover()

			err = pcap.Write(packets)
			Expect(err).NotTo(HaveOccurred())
		}()

		wg.Add(1)
		go func() {
			packet := dummyPacket()

			// Write 1 packet and invoke time rotation
			packets <- packet
			timeChan <- time.Now()

			// Write 1 packet and invoke time rotation
			packets <- packet
			time.Sleep(time.Duration(maxAge) * time.Second)
			timeChan <- time.Now()

			// Write 1 packet to flush data being written to current pcap file
			packets <- packet

			wg.Done()
		}()

		// Wait for all the packets to be written to file
		wg.Wait()

		// Assert written files on disk
		assertPcapFiles(captureDir, []outputFile{
			currentOrderTwoFileSizeOnePacket,
			rotatedFileOrderOneSizeOnePacket,
			rotatedFileOrderZeroSizeOnePacket,
		})

		// Assert that three updates were sent
		var update = make([]*proto.PacketCaptureStatusUpdate, 3)
		Eventually(updates).Should(Receive(&update[0]))
		Eventually(updates).Should(Receive(&update[1]))
		Eventually(updates).Should(Receive(&update[2]))
		assertStatusUpdates(update[0], []outputFile{currentOrderTwoFileSizeOnePacket}, namespace, name)
		assertStatusUpdates(update[1], []outputFile{
			currentOrderTwoFileSizeOnePacket,
			rotatedFileOrderOneSizeOnePacket},
			namespace, name)
		assertStatusUpdates(update[2], []outputFile{
			currentOrderTwoFileSizeOnePacket,
			rotatedFileOrderOneSizeOnePacket,
			rotatedFileOrderZeroSizeOnePacket},
			namespace, name)
	}, 10)

	It("Do not rotate an empty file", func(done Done) {
		defer close(done)
		var err error
		var maxAge = 1
		var timeChan = make(chan time.Time)
		var ticker = &time.Ticker{C: timeChan}
		var updates = make(chan interface{}, 100)
		defer close(updates)

		// Initialise a new capture
		var pcap capture.PcapFile
		var packets = make(chan gopacket.Packet)
		defer close(packets)
		pcap = capture.NewRotatingPcapFile(baseDir, namespace, name, podName, deviceName,
			updates,
			capture.WithRotationSeconds(maxAge),
			capture.WithTicker(ticker),
		)
		defer pcap.Done()

		// Capture listens to incoming packets
		go func() {
			defer GinkgoRecover()

			err = pcap.Write(packets)
			Expect(err).NotTo(HaveOccurred())
		}()

		// wait for time rotation to be invoked
		timeChan <- time.Now()

		// Define expected files
		var expectedFiles = []outputFile{
			{
				Name: fmt.Sprintf("%s_%s.pcap", podName, deviceName),
				Size: capture.GlobalHeaderLen,
			},
		}

		// Assert written files on disk
		assertPcapFiles(captureDir, expectedFiles)

		// Assert that an update was sent
		var update *proto.PacketCaptureStatusUpdate
		Eventually(updates).Should(Receive(&update))
		assertStatusUpdates(update, expectedFiles, namespace, name)
	}, 10)

	It("Invoke size rotation before time rotation in a stream of data", func(done Done) {
		defer close(done)
		var wg sync.WaitGroup
		var err error
		var maxAge = 1
		var numberOfPackets = 10
		var half = numberOfPackets / 2
		var maxSize = capture.GlobalHeaderLen + half*(dummyPacketDataSize()+capture.PacketInfoLen)
		var timeChan = make(chan time.Time)
		var ticker = &time.Ticker{C: timeChan}

		// Initialise a new capture
		var pcap capture.PcapFile
		var packets = make(chan gopacket.Packet)
		defer close(packets)
		pcap = capture.NewRotatingPcapFile(baseDir, "", "", podName, deviceName,
			make(chan interface{}, 100),
			capture.WithRotationSeconds(maxAge),
			capture.WithMaxSizeBytes(maxSize),
			capture.WithTicker(ticker),
		)
		defer pcap.Done()

		// Capture listens to incoming packets
		go func() {
			defer GinkgoRecover()

			err = pcap.Write(packets)
			Expect(err).NotTo(HaveOccurred())
		}()

		wg.Add(numberOfPackets)
		go func() {
			packet := dummyPacket()

			for i := 0; i < half; i++ {
				packets <- packet
				wg.Done()
			}

			packets <- packet
			wg.Done()
			timeChan <- time.Now()

			for i := 0; i < half-1; i++ {
				packets <- packet
				wg.Done()
			}
		}()

		// Wait for all the packets to be written to file
		wg.Wait()

		assertPcapFiles(baseDir, []outputFile{
			{
				Name:  fmt.Sprintf("%s_%s.pcap", podName, deviceName),
				Size:  capture.GlobalHeaderLen + half*(dummyPacketDataSize()+capture.PacketInfoLen),
				Order: 1,
			},
			{
				Name:  fmt.Sprintf("%s_%s.[\\d]+.pcap", podName, deviceName),
				Size:  capture.GlobalHeaderLen + half*(dummyPacketDataSize()+capture.PacketInfoLen),
				Order: 0,
			},
		})
	}, 10)

	It("Invoke time rotation before size rotation in a stream of data", func(done Done) {
		defer close(done)
		var wg sync.WaitGroup
		var err error
		var maxAge = 1
		var numberOfPackets = 10
		var half = numberOfPackets / 2
		var maxSize = capture.GlobalHeaderLen + half*(dummyPacketDataSize()+capture.PacketInfoLen)
		var timeChan = make(chan time.Time)
		var ticker = &time.Ticker{C: timeChan}
		var updates = make(chan interface{}, 100)
		defer close(updates)

		// Initialise a new capture
		var pcap capture.PcapFile
		var packets = make(chan gopacket.Packet)
		defer close(packets)
		pcap = capture.NewRotatingPcapFile(baseDir, namespace, name, podName, deviceName,
			updates,
			capture.WithRotationSeconds(maxAge),
			capture.WithMaxSizeBytes(maxSize),
			capture.WithTicker(ticker),
		)
		defer pcap.Done()

		// Capture listens to incoming packets
		go func() {
			defer GinkgoRecover()

			err = pcap.Write(packets)
			Expect(err).NotTo(HaveOccurred())
		}()

		wg.Add(numberOfPackets)
		go func() {
			packet := dummyPacket()

			for i := 0; i < half; i++ {
				packets <- packet
				wg.Done()
			}

			timeChan <- time.Now()
			packets <- packet
			wg.Done()

			for i := 0; i < half-1; i++ {
				packets <- packet
				wg.Done()
			}
		}()

		// Wait for all the packets to be written to file
		wg.Wait()

		// Assert written files on disk
		assertPcapFiles(captureDir, []outputFile{
			currentFileOrderOneSizeFivePackets,
			rotatedFileOrderZeroSizeFivePackets,
		})
		// Assert that two updates were sent
		var update = make([]*proto.PacketCaptureStatusUpdate, 2)
		Eventually(updates).Should(Receive(&update[0]))
		Eventually(updates).Should(Receive(&update[1]))
		assertStatusUpdates(update[0], []outputFile{currentFileOrderOneSizeFivePackets}, namespace, name)
		assertStatusUpdates(update[1], []outputFile{
			currentFileOrderOneSizeFivePackets,
			rotatedFileOrderZeroSizeFivePackets},
			namespace, name)
	}, 10)

	It("Keeps latest files", func(done Done) {
		defer close(done)
		var wg sync.WaitGroup
		var err error
		var numberOfPackets = 10
		var maxSize = capture.GlobalHeaderLen + (dummyPacketDataSize() + capture.PacketInfoLen)
		var maxFiles = 2
		var updates = make(chan interface{}, 100)
		defer close(updates)

		// Initialise a new capture
		var pcap capture.PcapFile
		var packets = make(chan gopacket.Packet)
		defer close(packets)
		pcap = capture.NewRotatingPcapFile(baseDir, namespace, name, podName, deviceName,
			updates,
			capture.WithMaxFiles(maxFiles),
			capture.WithMaxSizeBytes(maxSize),
		)
		defer pcap.Done()

		// Capture listens to incoming packets
		go func() {
			defer GinkgoRecover()

			err = pcap.Write(packets)
			Expect(err).NotTo(HaveOccurred())
		}()

		// Write 10 packets
		wg.Add(numberOfPackets)
		go func() {
			for i := 0; i < numberOfPackets; i++ {
				packet := dummyPacket()

				packets <- packet
				wg.Done()
			}
		}()

		// Wait for all the packets to be written to file
		wg.Wait()

		// Assert written files on disk
		assertPcapFiles(captureDir, []outputFile{
			currentOrderTwoFileSizeOnePacket,
			rotatedFileOrderOneSizeOnePacket,
			rotatedFileOrderZeroSizeOnePacket,
		})

		// Assert that three updates were sent
		var update = make([]*proto.PacketCaptureStatusUpdate, 10)
		for i := 0; i < numberOfPackets; i++ {
			Eventually(updates).Should(Receive(&update[i]))
		}

		assertStatusUpdates(update[0], []outputFile{currentOrderTwoFileSizeOnePacket}, namespace, name)
		assertStatusUpdates(update[1], []outputFile{
			currentOrderTwoFileSizeOnePacket,
			rotatedFileOrderOneSizeOnePacket},
			namespace, name)
		for i := 2; i < numberOfPackets; i++ {
			assertStatusUpdates(update[i], []outputFile{
				currentOrderTwoFileSizeOnePacket,
				rotatedFileOrderOneSizeOnePacket,
				rotatedFileOrderZeroSizeOnePacket},
				namespace, name)

		}

	}, 10)

	It("Start a capture after it has been stopped", func(done Done) {
		defer close(done)

		var err error
		var updates = make(chan interface{}, 100)
		defer close(updates)

		// Initialise a new capture
		var pcap = capture.NewRotatingPcapFile(baseDir, namespace, name, podName, deviceName, updates)

		// Capture listens to incoming packets
		var packets1 = make(chan gopacket.Packet)
		defer close(packets1)
		go func() {
			defer GinkgoRecover()

			err = pcap.Write(packets1)
			Expect(err).NotTo(HaveOccurred())
		}()

		// Write 1 packet
		packet := dummyPacket()
		packets1 <- packet

		pcap.Done()

		assertPcapFiles(captureDir, []outputFile{
			currentOrderTwoFileSizeOnePacket,
		})
		// Assert that an update was sent
		var updateOne *proto.PacketCaptureStatusUpdate
		Eventually(updates).Should(Receive(&updateOne))
		assertStatusUpdates(updateOne, []outputFile{
			currentOrderTwoFileSizeOnePacket,
		}, namespace, name)

		// open another packet with the same base name
		var pcap2 = capture.NewRotatingPcapFile(baseDir, namespace, name, podName, deviceName, updates)
		// Write again
		var packets2 = make(chan gopacket.Packet)
		defer close(packets2)
		go func() {
			defer GinkgoRecover()

			err = pcap2.Write(packets2)
			Expect(err).NotTo(HaveOccurred())
		}()

		// Write 1 packet
		packets2 <- packet
		defer pcap2.Done()

		assertPcapFiles(captureDir, []outputFile{
			currentOrderTwoFileSizeTwoPackets,
		})
		// Assert that an update was sent
		var updateTwo *proto.PacketCaptureStatusUpdate
		Eventually(updates).Should(Receive(&updateTwo))
		assertStatusUpdates(updateTwo, []outputFile{
			currentOrderTwoFileSizeTwoPackets,
		}, namespace, name)
	}, 10)

	It("Close capture after write channel has been stopped", func(done Done) {
		defer close(done)
		var err error
		var updates = make(chan interface{}, 100)
		defer close(updates)

		// Initialise a new capture
		var pcap = capture.NewRotatingPcapFile(baseDir, namespace, name, podName, deviceName, updates)

		// Capture listens to incoming packets
		var packets = make(chan gopacket.Packet)
		go func() {
			defer GinkgoRecover()

			err = pcap.Write(packets)
			Expect(err).NotTo(HaveOccurred())
		}()

		// Write 1 packet
		packet := dummyPacket()
		packets <- packet

		close(packets)
		pcap.Done()

		assertPcapFiles(captureDir, []outputFile{
			currentOrderTwoFileSizeOnePacket,
		})
		// Assert that an update was sent
		var update *proto.PacketCaptureStatusUpdate
		Eventually(updates).Should(Receive(&update))
		assertStatusUpdates(update, []outputFile{
			currentOrderTwoFileSizeOnePacket,
		}, namespace, name)
	}, 10)

	It("Writes packets after it has been stopped", func(done Done) {
		defer close(done)

		var err error
		var wg sync.WaitGroup

		// Initialise a new capture
		var pcap = capture.NewRotatingPcapFile(baseDir, "", "", podName, deviceName, make(chan interface{}, 100))

		// Capture listens to incoming packets
		var packets = make(chan gopacket.Packet)
		defer close(packets)

		wg.Add(1)
		go func() {
			defer GinkgoRecover()

			err = pcap.Write(packets)
			Expect(err).NotTo(HaveOccurred())
			wg.Done()
		}()

		packet := dummyPacket()

		// Write 1 packet
		packets <- packet

		// Close the capture
		pcap.Done()

		// Wait for Write to complete
		wg.Wait()

		// Call write a second time
		err = pcap.Write(packets)
		// Expect an error to be returned
		Expect(err).To(HaveOccurred())

	}, 10)

	It("Provides an update containing previously written files", func(done Done) {
		defer close(done)
		var wg sync.WaitGroup
		var err error
		var numberOfPackets = 1
		var updates = make(chan interface{}, 100)
		defer close(updates)

		// Write a pcap file in order to simulate a previous capture
		err = os.MkdirAll(captureDir, 0755)
		defer os.Remove(captureDir)
		Expect(err).NotTo(HaveOccurred())
		file, err := ioutil.TempFile(captureDir, fmt.Sprintf("%s_%s-*.pcap", podName, deviceName))
		Expect(err).NotTo(HaveOccurred())
		defer os.Remove(file.Name())

		// Initialise a new capture
		var pcap capture.PcapFile
		var packets = make(chan gopacket.Packet)
		defer close(packets)
		pcap = capture.NewRotatingPcapFile(baseDir, namespace, name, podName, deviceName, updates)
		defer pcap.Done()

		// Capture listens to incoming packets
		go func() {
			defer GinkgoRecover()

			err = pcap.Write(packets)
			Expect(err).NotTo(HaveOccurred())
		}()

		// Write 1 packet
		wg.Add(numberOfPackets)
		go func() {
			var packet = dummyPacket()

			packets <- packet
			wg.Done()
		}()

		// Wait for all the packets to be written to file
		wg.Wait()

		var dummyFile = outputFile{
			Name:  fmt.Sprintf("%s_%s.[\\d]+.pcap", podName, deviceName),
			Size:  0,
			Order: 0,
		}
		// Assert written files on disk
		assertPcapFiles(captureDir, []outputFile{
			currentOrderTwoFileSizeOnePacket,
			dummyFile,
		})

		// Assert that an update was sent
		var update *proto.PacketCaptureStatusUpdate
		Eventually(updates).Should(Receive(&update))
		assertStatusUpdates(update, []outputFile{
			currentOrderTwoFileSizeOnePacket,
			dummyFile,
		}, namespace, name)
	}, 10)
})

type outputFile struct {
	Name  string
	Size  int
	Order int
}

func assertPcapFiles(baseDir string, expected []outputFile) {
	Eventually(func() []os.FileInfo { return read(baseDir) }).Should(HaveLen(len(expected)), "wrong length in assertPcapFiles")
	sort.Slice(expected, func(i, j int) bool {
		return expected[i].Order < expected[j].Order
	})

	for i, f := range expected {
		Eventually(func() int { return int(read(baseDir)[i].Size()) }).Should(Equal(f.Size))
		Eventually(func() string { return read(baseDir)[i].Name() }).Should(MatchRegexp(f.Name))
	}
}

func assertStatusUpdates(update *proto.PacketCaptureStatusUpdate, expected []outputFile, expectedNs string,
	expectedCaptureName string) {
	sort.Slice(expected, func(i, j int) bool {
		return expected[i].Order < expected[j].Order
	})
	Expect(update.CaptureFiles).To(HaveLen(len(expected)), "wrong length in assertStatusUpdates")
	for i := range expected {
		Expect(update.CaptureFiles[i]).To(MatchRegexp(expected[i].Name))
	}
	Expect(update.Id.GetNamespace()).To(Equal(expectedNs))
	Expect(update.Id.GetName()).To(Equal(expectedCaptureName))
}

func read(baseDir string) []os.FileInfo {
	var pCaps []os.FileInfo
	var err error
	pCaps, err = ioutil.ReadDir(baseDir)
	if err != nil {
		return pCaps
	}

	sort.Slice(pCaps, func(i, j int) bool {
		return pCaps[i].Name() < pCaps[j].Name()
	})
	return pCaps
}

func dummyPacketDataSize() int {
	return len(dummyPacketData())
}

func dummyPacket() gopacket.Packet {
	data := dummyPacketData()
	packet := gopacket.NewPacket(data, layers.LayerTypeIPv4, gopacket.Default)
	packet.Metadata().CaptureLength = len(data)
	packet.Metadata().Length = len(data)

	return packet
}

func dummyPacketData() []byte {
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{}
	_ = gopacket.SerializeLayers(buf, opts,
		&layers.Ethernet{
			SrcMAC: []byte{0, 0, 0, 0, 0, 1},
			DstMAC: []byte{0, 0, 0, 0, 0, 2},
		},
		&layers.IPv4{
			Version: 4,
			SrcIP:   net.IP{1, 1, 1, 1},
			DstIP:   net.IP{1, 1, 1, 2},
			TTL:     128,
		},
		&layers.TCP{
			SrcPort: layers.TCPPort(1000),
			DstPort: layers.TCPPort(80),
			SYN:     true,
		},
		gopacket.Payload([]byte{1, 2, 3, 4}))
	return buf.Bytes()
}
