// Copyright (c) 2020-2021 Tigera, Inc. All rights reserved.

package capture

import (
	"errors"
	"os"

	"k8s.io/utils/strings"

	log "github.com/sirupsen/logrus"
)

// ActiveCaptures stores the state of the current active capture
// Adding a new capture triggers a capture start
// Removing a capture triggers a capture end
type ActiveCaptures interface {
	Add(key Key, deviceName string) error
	Remove(key Key) (error, string)
}

// ErrNotFound will be returned when trying to remove a capture that has not been marked as active
var ErrNotFound = errors.New("no capture is active")

// ErrDuplicate will be returned when trying to add the same capture twice
var ErrDuplicate = errors.New("an active capture is already in progress")

// ErrNoSpaceLeft will be returned when no free space is detected to start a new capture
var ErrNoSpaceLeft = errors.New("no space left for capture")

// Key represent a unique identifier for the capture
type Key struct {
	WorkloadEndpointId string
	Namespace          string
	CaptureName        string
}

type activeCaptures struct {
	deviceRef       map[Key]string
	cache           map[Key]Capture
	captureDir      string
	maxSizeBytes    int
	rotationSeconds int
	maxFiles        int
	statusUpdates   chan interface{}
}

func NewActiveCaptures(config Config, statusUpdates chan interface{}) (ActiveCaptures, error) {
	var err = os.MkdirAll(config.Directory, 0755)
	if err != nil {
		return nil, err
	}

	return &activeCaptures{
		cache:           map[Key]Capture{},
		deviceRef:       map[Key]string{},
		captureDir:      config.Directory,
		maxSizeBytes:    config.MaxSizeBytes,
		rotationSeconds: config.RotationSeconds,
		maxFiles:        config.MaxFiles,
		statusUpdates:   statusUpdates,
	}, nil
}

func (activeCaptures *activeCaptures) Add(key Key, deviceName string) error {
	log.WithField("CAPTURE", key.CaptureName).Infof("Adding capture for device name %s for %s", deviceName, key)

	var err error
	_, ok := activeCaptures.cache[key]
	if ok {
		return ErrDuplicate
	}

	size, err := GetFreeDiskSize(activeCaptures.captureDir)
	if err != nil {
		return err
	}

	// This will check if the free disk capacity can accommodate another capture
	// The free disk capacity is calculated per OS

	// A capture can have at most activeCaptures.maxFiles+1 (max files represents number of rotated files + current file)
	// of size maxSizeBytes
	if size <= uint64((activeCaptures.maxFiles+1)*activeCaptures.maxSizeBytes) {
		return ErrNoSpaceLeft
	}

	var _, podName = strings.SplitQualifiedName(key.WorkloadEndpointId)

	var newCapture = NewRotatingPcapFile(activeCaptures.captureDir,
		key.Namespace,
		key.CaptureName,
		podName,
		deviceName,
		activeCaptures.statusUpdates,
		WithMaxSizeBytes(activeCaptures.maxSizeBytes),
		WithRotationSeconds(activeCaptures.rotationSeconds),
		WithMaxFiles(activeCaptures.maxFiles),
	)

	go func() {
		log.WithField("CAPTURE", key.CaptureName).Info("Start")
		err = newCapture.Start()
		if err != nil {
			log.WithField("CAPTURE", key.CaptureName).WithError(err).Error("Failed to start capture or capture ended prematurely")
		}
	}()

	activeCaptures.cache[key] = newCapture
	activeCaptures.deviceRef[key] = deviceName

	return nil
}

func (activeCaptures *activeCaptures) Remove(key Key) (error, string) {
	log.WithField("CAPTURE", key.CaptureName).Infof("Removing capture %s", key)

	_, ok := activeCaptures.cache[key]
	if !ok {
		return ErrNotFound, ""
	}

	activeCaptures.cache[key].Stop()
	delete(activeCaptures.cache, key)
	deviceName := activeCaptures.deviceRef[key]
	delete(activeCaptures.deviceRef, key)

	return nil, deviceName
}
