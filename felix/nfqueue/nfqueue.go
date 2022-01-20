// Copyright (c) 2021-2022 Tigera, Inc. All rights reserved.

package nfqueue

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/projectcalico/calico/felix/versionparse"

	"github.com/mdlayher/netlink"

	log "github.com/sirupsen/logrus"

	"github.com/prometheus/client_golang/prometheus"

	gonfqueue "github.com/florianl/go-nfqueue"
)

var (
	v3Dot13Dot0 = versionparse.MustParseVersion("3.13.0")
)

const (
	nfMaxPacketLen = 0xFFFF
	nfMaxQueueLen  = 0xFF
	nfReadTimeout  = 100 * time.Millisecond
	nfWriteTimeout = 200 * time.Millisecond
)

var (
	queueLength = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "felix_dns_policy_nfqueue_queue_length",
		Help: "Length of queue",
	})

	nfqueueShutdownCount = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "felix_dns_policy_nfqueue_shutdown_count",
		Help: "Number of times nfqueue was shutdown due to a fatal error",
	})

	PrometheusNfqueueVerdictFailCount = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "felix_dns_policy_nfqueue_monitor_nf_verdict_failed",
		Help: "Count of the number of times that the monitor has failed to set the verdict",
	})
)

func init() {
	prometheus.MustRegister(queueLength)
	prometheus.MustRegister(nfqueueShutdownCount)
	prometheus.MustRegister(PrometheusNfqueueVerdictFailCount)
}

func DefaultNfqueueCreator(queueID int) func() (Nfqueue, error) {
	return func() (Nfqueue, error) {
		log.Infof("Creating new NFQUEUE connection with queue id \"%d\" for dns policy packet processing.", queueID)
		nf, err := NewNfqueue(queueID)
		if err != nil {
			return nil, err
		}

		return nf, nil
	}
}

type nfQueue struct {
	*gonfqueue.Nfqueue

	attrsChannel    chan gonfqueue.Attribute
	closeOnce       sync.Once
	shutdownChannel chan struct{}
}

type Nfqueue interface {
	// SetVerdict signals the kernel the next action for a specified package id. Implementations of this must be
	// thread safe.
	SetVerdict(id uint32, verdict int) error
	SetVerdictWithMark(id uint32, verdict, mark int) error
	PacketAttributesChannel() <-chan gonfqueue.Attribute

	DebugKillConnection() error

	// ShutdownNotificationChannel notifies the listener when if the connection to nfqueue has been terminated. If a
	// signal is sent on this channel the listener must call Close() on the Nfqueue as it is no longer usable.
	ShutdownNotificationChannel() <-chan struct{}
	Close() error
}

func NewNfqueue(queueID int) (Nfqueue, error) {
	defaultConfig := &gonfqueue.Config{
		NfQueue:      uint16(queueID),
		MaxPacketLen: nfMaxPacketLen,
		MaxQueueLen:  nfMaxQueueLen,
		Copymode:     gonfqueue.NfQnlCopyPacket,
		ReadTimeout:  nfReadTimeout,
		WriteTimeout: nfWriteTimeout,
	}
	nfRaw, err := gonfqueue.Open(defaultConfig)
	if err != nil {
		return nil, err
	}

	nf := &nfQueue{
		Nfqueue:         nfRaw,
		attrsChannel:    make(chan gonfqueue.Attribute, 1000),
		shutdownChannel: make(chan struct{}),
	}

	err = nf.RegisterWithErrorFunc(context.Background(), func(a gonfqueue.Attribute) int {
		queueLength.Set(float64(len(nf.attrsChannel)))

		select {
		case nf.attrsChannel <- a:
		default:
			log.Warning("dropping packet because nfqueue channel is full")
			if err := nf.SetVerdict(*a.PacketID, gonfqueue.NfDrop); err != nil {
				log.WithError(err).Error("failed to set verdict for packet")
			}
		}

		return 0
	}, func(err error) int {
		if opError, ok := err.(*netlink.OpError); ok {
			if opError.Timeout() || opError.Temporary() {
				return 0
			}
		}

		nfqueueShutdownCount.Inc()

		// Calling nf.Close() will close the underlying connection for nfqueue. This is done because the owner of this
		// nfqueue instance may want to create a new nfqueue connection and if this connection is not closed then
		// creating a new one will prompt an error.
		err = nf.Close()
		if err != nil {
			log.WithError(err).Warning("an error occurred while closing nfqueue.")
		}

		return 1
	})

	if err != nil {
		return nil, err
	}

	return nf, nil
}

func (nf *nfQueue) PacketAttributesChannel() <-chan gonfqueue.Attribute {
	return nf.attrsChannel
}

func (nf *nfQueue) ShutdownNotificationChannel() <-chan struct{} {
	return nf.shutdownChannel
}

func (nf *nfQueue) Close() error {
	var err error
	nf.closeOnce.Do(func() {
		// Close the underlying connection before we report doing so with the shutdown channel in case the caller is
		// going to create another nfqueue connection. This guarantees that the connection will not be open when the
		// caller creates a new connection (as this results in an error).
		err = nf.Nfqueue.Close()

		close(nf.shutdownChannel)
		close(nf.attrsChannel)
	})

	return err
}

// DebugKillConnection finds the underlying file descriptor for the nfqueue connection and closes it. This is used to
// simulate an unexpected closure of the connection. The underlying nfqueue library may close the connection without
// notification and without restarting it if it encounters errors, so this function is used to force such an error
// so the restart logic can be tested with fv's.
//
// In general, DO NOT USE THIS FUNCTION.
func (nf *nfQueue) DebugKillConnection() error {
	path := []string{"sock", "s", "fd", "file", "pfd", "Sysfd"}
	current := reflect.ValueOf(nf.Con)
	for _, v := range path {
		if current.Kind() == reflect.Interface {
			current = current.Elem()
		}

		if current.Kind() == reflect.Ptr {
			current = current.Elem()
		}

		if current.Kind() != reflect.Struct {
			break
		}

		current = current.FieldByName(v)
		if !current.IsValid() {
			return fmt.Errorf("field path to file descriptor is invalid")
		}
	}

	if !current.IsValid() {
		return fmt.Errorf("field path to file descriptor is invalid")
	}

	fd := reflect.NewAt(current.Type(), unsafe.Pointer(current.UnsafeAddr())).Elem().Interface().(int)

	return syscall.Close(fd)
}

func isAtLeastKernel(v *versionparse.Version) error {
	versionReader, err := versionparse.GetKernelVersionReader()
	if err != nil {
		return fmt.Errorf("failed to get kernel version reader: %v", err)
	}

	kernelVersion, err := versionparse.GetKernelVersion(versionReader)
	if err != nil {
		return fmt.Errorf("failed to get kernel version: %v", err)
	}

	if kernelVersion.Compare(v) < 0 {
		return fmt.Errorf("kernel is too old (have: %v but want at least: %v)", kernelVersion, v)
	}

	return nil
}

// SupportsNfQueueWithBypass returns true if the kernel version supports NFQUEUE with the queue-bypass option,
func SupportsNfQueueWithBypass() error {
	return isAtLeastKernel(v3Dot13Dot0)
}
