// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package capture

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/google/gopacket/pcapgo"
	log "github.com/sirupsen/logrus"
)

// PacketInfoLen represents the size of packet header. The packet header will be written
// before any packet data captures
const PacketInfoLen = 16

// GlobalHeaderLen represents the size global packet header written once per pcap file.
const GlobalHeaderLen = 24

// maxSizePerPacket represents the max size captured per packet
const maxSizePerPacket = 65536

// Timeout used for non-blocking read for capturing packets for a network interface
const defaultReadTimeout = 1 * time.Second

// Capture starts/stops a packet capture for an active interface
type Capture interface {
	Start() error
	Stop()
}

// PcapFile writes packets captured from an active interface to a pcap file
type PcapFile interface {
	// Writes packets to disk that are being read from network interface
	Write(chan gopacket.Packet) error
	// Stop capture and closes all used resources. Should not be used without Write()
	Done()
}

type rotatingPcapFile struct {
	// parameters to adjust packet capture
	directory       string
	baseName        string
	deviceName      string
	maxSizeBytes    int
	rotationSeconds int
	maxFiles        int
	done            chan struct{}
	isDone          bool

	// the parameters below should not be made available to users
	currentSize  int
	lastRotation time.Time
	output       *os.File
	writer       *pcapgo.Writer
	handle       *pcap.Handle
	ticker       *time.Ticker
}

type Option func(file *rotatingPcapFile)

// WithTicker changes default ticker that performs time based rotation
func WithTicker(t *time.Ticker) Option {
	return func(c *rotatingPcapFile) {
		c.ticker = t
	}
}

// WithMaxSizeBytes changes default value for pcap file size
func WithMaxSizeBytes(v int) Option {
	return func(c *rotatingPcapFile) {
		c.maxSizeBytes = v
	}
}

// WithRotationSeconds changes default value for time based rotation
func WithRotationSeconds(v int) Option {
	return func(c *rotatingPcapFile) {
		c.rotationSeconds = v
	}
}

// WithMaxFiles changes default value for maximum pcap backups
func WithMaxFiles(v int) Option {
	return func(c *rotatingPcapFile) {
		c.maxFiles = v
	}
}

// NewRotatingPcapFile creates a rotatingPcapFile. It will capture traffic from a live interface
// defined by deviceName and store under a specified directory. Traffic will be stored on disk
// using pcap file format. All pcap files will have a name that matches baseName. The pcap
// file that will is currently used for logging will have {baseName}.pcap format, while older
// files will have {baseName}{rotationTimestamp}.pcap. Pcap files will be rotated using both
// time and size and only keep a predefined number of backup files.
func NewRotatingPcapFile(dir, baseName, deviceName string, opts ...Option) *rotatingPcapFile {

	const (
		defaultMaxSizeBytes    = 10 * 1000 * 1000
		defaultRotationSeconds = 3600
		defaultMaxFiles        = 2
	)

	var capture = &rotatingPcapFile{
		directory:       dir,
		baseName:        baseName,
		deviceName:      deviceName,
		maxSizeBytes:    defaultMaxSizeBytes,
		rotationSeconds: defaultRotationSeconds,
		maxFiles:        defaultMaxFiles,
		done:            make(chan struct{}),
	}

	for _, opt := range opts {
		opt(capture)
	}

	if capture.ticker == nil {
		capture.ticker = time.NewTicker(time.Duration(capture.rotationSeconds) * time.Second)
	}

	return capture
}

func (capture *rotatingPcapFile) open() error {
	var err error

	log.WithField("CAPTURE", capture.deviceName).Debugf("Creating base directory %s", capture.directory)
	err = os.MkdirAll(capture.directory, 0755)
	if err != nil {
		return err
	}

	var currentFile = fmt.Sprintf("%s/%s.pcap", capture.directory, capture.baseName)
	var info os.FileInfo
	if info, err = os.Stat(currentFile); err == nil {
		log.WithField("CAPTURE", capture.deviceName).Debug("Open existing pcap file")
		capture.output, err = os.OpenFile(currentFile, os.O_APPEND|os.O_WRONLY, 0644)
	} else {
		log.WithField("CAPTURE", capture.deviceName).Debug("Creating pcap file")
		capture.output, err = os.OpenFile(currentFile, os.O_CREATE|os.O_WRONLY, 0644)
	}

	if err != nil {
		return err
	}

	log.WithField("CAPTURE", capture.deviceName).Debug("Opening a new writer")
	capture.writer = pcapgo.NewWriter(capture.output)
	if info == nil {
		capture.currentSize = 0
		if err = capture.writeHeader(); err != nil {
			return err
		}
	} else {
		capture.currentSize = int(info.Size())
	}

	return err
}

func (capture *rotatingPcapFile) close() error {
	log.WithField("CAPTURE", capture.deviceName).Debug("Closing pcap file")
	return capture.output.Close()
}

func (capture *rotatingPcapFile) tryToRotate() error {
	// We do not rotate if a previous rotation was just issued
	// or if no traffic was written
	var diff = time.Since(capture.lastRotation)
	if capture.currentSize > GlobalHeaderLen && (diff.Seconds() >= float64(capture.rotationSeconds)) {
		// When a size based rotation was been currently issued
		// we need to wait rotationSeconds until we rotate
		// in order to avoid small file creation
		return capture.rotate()
	} else if capture.currentSize >= capture.maxSizeBytes {
		// When a time based rotation was been currently issued
		// we need to wait until currentSize reached maxSizeBytes until we rotate
		// in order to avoid small file creation
		return capture.rotate()
	}

	return nil
}

func (capture *rotatingPcapFile) rotate() error {
	var err error
	if err = capture.close(); err != nil {
		return err
	}

	var currentTime = time.Now()
	var newName = fmt.Sprintf("%s/%s-%d.pcap", capture.directory, capture.baseName, currentTime.UnixNano()/1000)

	log.WithField("CAPTURE", capture.deviceName).Debugf("Rename pcap file to %s", newName)
	err = os.Rename(fmt.Sprintf("%s/%s.pcap", capture.directory, capture.baseName), newName)
	if err != nil {
		return err
	}

	capture.lastRotation = currentTime
	if err = capture.open(); err != nil {
		return err
	}

	capture.cleanOlderFiles()

	return nil
}

func (capture *rotatingPcapFile) cleanOlderFiles() {
	var files []os.FileInfo
	var err error

	if capture.maxFiles == 0 {
		return
	}

	if _, err = os.Stat(capture.directory); err != nil {
		return
	}

	err = filepath.Walk(capture.directory, func(path string, info os.FileInfo, err error) error {
		if info != nil && !info.IsDir() && strings.HasSuffix(info.Name(), ".pcap") {
			if info.Name() != fmt.Sprintf("%s.pcap", capture.baseName) {
				files = append(files, info)
			}
		}
		return nil
	})

	if err != nil {
		log.WithField("CAPTURE", capture.deviceName).WithError(err).Errorf("Failed to list directory %s", capture.directory)
	}

	// Sort files in a descending order using last modification timestamp
	sort.Slice(files, func(current, next int) bool {
		return files[current].ModTime().UnixNano() < files[next].ModTime().UnixNano()
	})

	if len(files) <= capture.maxFiles {
		return
	}

	for _, file := range files[:capture.maxFiles-1] {
		log.WithField("CAPTURE", capture.deviceName).Debugf("Removing %s", file.Name())
		err = os.Remove(fmt.Sprintf("%s/%s", capture.directory, file.Name()))
		if err != nil {
			log.WithField("CAPTURE", capture.deviceName).WithError(err).Errorf("Failed to remove file %s", file.Name())
		}
	}
}

func (capture *rotatingPcapFile) Write(packets chan gopacket.Packet) error {
	if capture.isDone {
		return fmt.Errorf("capture has been already closed")
	}

	var err error
	log.WithField("CAPTURE", capture.deviceName).Debug("Start writing packets to pcap files")
	if err = capture.open(); err != nil {
		return err
	}
	defer capture.doDone()

	for {
		select {
		case packet := <-packets:
			if packet == nil {
				continue
			}

			// check if rotation is needed due to size
			if packet.Metadata().CaptureLength+PacketInfoLen+capture.currentSize > capture.maxSizeBytes {
				log.WithField("CAPTURE", capture.deviceName).Debug("Will exceed maxSize. Will invoke rotation")
				if err = capture.tryToRotate(); err != nil {
					log.WithError(err).WithField("CAPTURE", capture.deviceName).Error("Could not rotate file")
					return err
				}
			}

			// write the packets to file
			if err = capture.writePacket(packet); err != nil {
				return err
			}
		case <-capture.ticker.C:
			// rotate based on time
			log.WithField("CAPTURE", capture.deviceName).Debug("Wil exceed time limit. Will invoke rotation")
			if err = capture.tryToRotate(); err != nil {
				log.WithError(err).WithField("CAPTURE", capture.deviceName).Error("Could not rotate file")
				return err
			}
		case <-capture.done:
			return nil
		}
	}
}

func (capture *rotatingPcapFile) doDone() {
	var err error
	if err = capture.close(); err != nil {
		log.WithError(err).WithField("CAPTURE", capture.deviceName).Error("Could not close file")
	}
	capture.isDone = true
	capture.ticker.Stop()
}

func (capture *rotatingPcapFile) Done() {
	capture.done <- struct{}{}
}

func (capture *rotatingPcapFile) writePacket(packet gopacket.Packet) error {
	var err = capture.writer.WritePacket(packet.Metadata().CaptureInfo, packet.Data())
	if err != nil {
		log.WithError(err).WithField("CAPTURE", capture.deviceName).Error("Could not write packet")
		return err
	}
	capture.currentSize += len(packet.Data()) + PacketInfoLen
	return nil
}

func (capture *rotatingPcapFile) writeHeader() error {
	if capture.currentSize == 0 {
		var err = capture.writer.WriteFileHeader(uint32(maxSizePerPacket), layers.LinkTypeEthernet)
		if err != nil {
			log.WithError(err).WithField("CAPTURE", capture.deviceName).Error("Could not write global headers")
			return err
		}
		capture.currentSize += GlobalHeaderLen
	}
	return nil
}

func (capture *rotatingPcapFile) Start() error {
	var err error

	capture.handle, err = pcap.OpenLive(capture.deviceName, int32(maxSizePerPacket), false, defaultReadTimeout)
	if err != nil {
		return err
	}

	packetSource := gopacket.NewPacketSource(capture.handle, capture.handle.LinkType())
	return capture.Write(packetSource.Packets())
}

func (capture *rotatingPcapFile) Stop() {
	capture.Done()
	capture.handle.Close()
}
