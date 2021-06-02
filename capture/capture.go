// Copyright (c) 2020-2021 Tigera, Inc. All rights reserved.

package capture

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/projectcalico/felix/proto"

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
	// Write writes packets to disk that are being read from network interface
	Write(chan gopacket.Packet) error
	// Done stops capture and closes all used resources. Should not be used without Write()
	Done()
}

type rotatingPcapFile struct {
	// parameters to adjust packet capture
	directory       string
	baseName        string
	deviceName      string
	namespace       string
	captureName     string
	maxSizeBytes    int
	rotationSeconds int
	maxFiles        int
	done            chan struct{}
	isDone          bool
	statusUpdates   chan interface{}

	// the parameters below should not be made available to users
	currentSize  int
	lastRotation time.Time
	output       *os.File
	writer       *pcapgo.Writer
	handle       *pcap.Handle
	ticker       *time.Ticker
	loggingID    string
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
// defined by deviceName and store under a specified directory. The directory used to stored traffic is
// defined by {directory}/{namespace}/{captureName}, where directory is a general directory to store all
// capture files, and namespace and captureName represent the namespace and name of a PacketCapture resource.
// Traffic will be stored on disk using pcap file format. All pcap files will have a name that matches
// {podName}_{deviceName}. The pcap file that is currently used for logging will have {podName}_{deviceName}.pcap
// format, while older files will have {podName}_{deviceName}.{rotationTimestamp}.pcap. Pcap files will be rotated using
// both time and size and we only keep a predefined number of backup files.
func NewRotatingPcapFile(directory, namespace, captureName, podName, deviceName string, statusUpdates chan interface{}, opts ...Option) *rotatingPcapFile {

	const (
		defaultMaxSizeBytes    = 10 * 1000 * 1000
		defaultRotationSeconds = 3600
		defaultMaxFiles        = 2
	)

	var captureDirectory = filepath.Join(directory, namespace, captureName)
	var baseName = fmt.Sprintf("%s_%s", podName, deviceName)
	var loggingID = fmt.Sprintf("%s/%s/%s", namespace, captureName, deviceName)

	var capture = &rotatingPcapFile{
		directory:       captureDirectory,
		baseName:        baseName,
		deviceName:      deviceName,
		namespace:       namespace,
		captureName:     captureName,
		maxSizeBytes:    defaultMaxSizeBytes,
		rotationSeconds: defaultRotationSeconds,
		maxFiles:        defaultMaxFiles,
		done:            make(chan struct{}),
		statusUpdates:   statusUpdates,
		loggingID:       loggingID,
	}

	for _, opt := range opts {
		opt(capture)
	}

	if capture.ticker == nil {
		capture.ticker = time.NewTicker(time.Duration(capture.rotationSeconds) * time.Second)
	}

	return capture
}
func (capture *rotatingPcapFile) currentCaptureFileAbsolutePath() string {
	return filepath.Join(capture.directory, capture.currentCaptureFileName())
}

func (capture *rotatingPcapFile) currentCaptureFileName() string {
	return fmt.Sprintf("%s.pcap", capture.baseName)
}

func (capture *rotatingPcapFile) open() error {
	var err error

	log.WithField("CAPTURE", capture.loggingID).Debugf("Creating base directory %s", capture.directory)
	err = os.MkdirAll(capture.directory, 0755)
	if err != nil {
		return err
	}

	var currentFile = capture.currentCaptureFileAbsolutePath()
	var info os.FileInfo
	if info, err = os.Stat(currentFile); err == nil {
		log.WithField("CAPTURE", capture.loggingID).Debug("Open existing pcap file")
		capture.output, err = os.OpenFile(currentFile, os.O_APPEND|os.O_WRONLY, 0644)
	} else {
		log.WithField("CAPTURE", capture.loggingID).Debug("Creating pcap file")
		capture.output, err = os.OpenFile(currentFile, os.O_CREATE|os.O_WRONLY, 0644)
	}

	if err != nil {
		return err
	}

	log.WithField("CAPTURE", capture.loggingID).Debug("Opening a new writer")
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
	log.WithField("CAPTURE", capture.loggingID).Debug("Closing pcap file")
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
	var newName = filepath.Join(capture.directory, fmt.Sprintf("%s-%d.pcap", capture.baseName, currentTime.UnixNano()/1000))

	log.WithField("CAPTURE", capture.loggingID).Debugf("Rename pcap file to %s", newName)
	err = os.Rename(filepath.Join(capture.directory, capture.currentCaptureFileName()), newName)
	if err != nil {
		return err
	}

	capture.lastRotation = currentTime
	if err = capture.open(); err != nil {
		return err
	}

	var files = capture.cleanOlderFiles()

	go capture.updateStatus(capture.extractFileNames(files))

	return nil
}

func (capture *rotatingPcapFile) extractFileNames(files []os.FileInfo) []string {
	var fileNames []string
	for _, f := range files {
		fileNames = append(fileNames, f.Name())
	}

	// current capture file was previously filtered when listing the files
	// and needs to be appended to the received files
	fileNames = append(fileNames, capture.currentCaptureFileName())

	// Sort files in an ascending alphanumerical
	sort.Strings(fileNames)

	return fileNames
}

func (capture *rotatingPcapFile) cleanOlderFiles() []os.FileInfo {
	var err error

	if capture.maxFiles == 0 {
		return nil
	}

	if _, err = os.Stat(capture.directory); err != nil {
		return nil
	}

	err, files := capture.listFiles()

	if err != nil {
		log.WithField("CAPTURE", capture.loggingID).WithError(err).Errorf("Failed to list directory %s", capture.directory)
	}

	// Sort files in an ascending order using last modification timestamp
	sort.Slice(files, func(current, next int) bool {
		return files[current].ModTime().UnixNano() < files[next].ModTime().UnixNano()
	})

	if len(files) <= capture.maxFiles {
		return files
	}

	// We only need to keep the latest maxFiles; older files will be clean up
	var cutOffIndex = len(files) - capture.maxFiles
	for _, file := range files[:cutOffIndex] {
		log.WithField("CAPTURE", capture.loggingID).Debugf("Removing %s", file.Name())
		err = os.Remove(filepath.Join(capture.directory, file.Name()))
		if err != nil {
			log.WithField("CAPTURE", capture.loggingID).WithError(err).Errorf("Failed to remove file %s", file.Name())
		}
	}

	return files[cutOffIndex:]
}

func (capture *rotatingPcapFile) listFiles() (error, []os.FileInfo) {
	var files []os.FileInfo
	var err error

	err = filepath.Walk(capture.directory, func(path string, info os.FileInfo, err error) error {
		if info != nil && !info.IsDir() && strings.HasSuffix(info.Name(), ".pcap") {
			if info.Name() != capture.currentCaptureFileName() {
				files = append(files, info)
			}
		}
		return nil
	})
	return err, files
}

func (capture *rotatingPcapFile) Write(packets chan gopacket.Packet) error {
	if capture.isDone {
		return fmt.Errorf("capture has been already closed")
	}

	var err error
	log.WithField("CAPTURE", capture.loggingID).Debug("Start writing packets to pcap files")
	if err = capture.open(); err != nil {
		return err
	}
	defer capture.doDone()

	err, files := capture.listFiles()
	if err != nil {
		return err
	}
	go capture.updateStatus(capture.extractFileNames(files))

	for {
		select {
		case packet := <-packets:
			if packet == nil {
				continue
			}

			// check if rotation is needed due to size
			if packet.Metadata().CaptureLength+PacketInfoLen+capture.currentSize > capture.maxSizeBytes {
				log.WithField("CAPTURE", capture.loggingID).Debug("Will exceed maxSize. Will invoke rotation")
				if err = capture.tryToRotate(); err != nil {
					log.WithError(err).WithField("CAPTURE", capture.loggingID).Error("Could not rotate file")
					return err
				}
			}

			// write the packets to file
			if err = capture.writePacket(packet); err != nil {
				return err
			}
		case <-capture.ticker.C:
			// rotate based on time
			log.WithField("CAPTURE", capture.loggingID).Debug("Wil exceed time limit. Will invoke rotation")
			if err = capture.tryToRotate(); err != nil {
				log.WithError(err).WithField("CAPTURE", capture.loggingID).Error("Could not rotate file")
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
		log.WithError(err).WithField("CAPTURE", capture.loggingID).Error("Could not close file")
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
		log.WithError(err).WithField("CAPTURE", capture.loggingID).Error("Could not write packet")
		return err
	}
	capture.currentSize += len(packet.Data()) + PacketInfoLen
	return nil
}

func (capture *rotatingPcapFile) writeHeader() error {
	if capture.currentSize == 0 {
		var err = capture.writer.WriteFileHeader(uint32(maxSizePerPacket), layers.LinkTypeEthernet)
		if err != nil {
			log.WithError(err).WithField("CAPTURE", capture.loggingID).Error("Could not write global headers")
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

func (capture *rotatingPcapFile) updateStatus(fileNames []string) {
	var update = &proto.PacketCaptureStatusUpdate{
		Id:           &proto.PacketCaptureID{Name: capture.captureName, Namespace: capture.namespace},
		CaptureFiles: fileNames,
	}
	log.WithField("CAPTURE", capture.loggingID).Debugf("Sending PacketCaptureStatusUpdate to dataplane for files %v", fileNames)
	capture.statusUpdates <- update
}
