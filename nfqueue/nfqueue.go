// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package nfqueue

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/prometheus/client_golang/prometheus"

	gonfqueue "github.com/florianl/go-nfqueue"
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

	PrometheusNfqueueVerdictFailCount = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "felix_dns_policy_nfqueue_monitor_nf_verdict_failed",
		Help: "Count of the number of times that the monitor has failed to set the verdict",
	})
)

func init() {
	prometheus.MustRegister(queueLength)
	prometheus.MustRegister(PrometheusNfqueueVerdictFailCount)
}

type nfQueue struct {
	*gonfqueue.Nfqueue

	attrsChannel chan gonfqueue.Attribute
}

type Nfqueue interface {
	// SetVerdict signals the kernel the next action for a specified package id. Implementations of this must be
	// thread safe.
	SetVerdict(id uint32, verdict int) error
	SetVerdictWithMark(id uint32, verdict, mark int) error
	PacketAttributesChannel() <-chan gonfqueue.Attribute
}

func NewNfqueue(queueID int) (Nfqueue, error) {
	nfqueueAttrChan := make(chan gonfqueue.Attribute, 1000)

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
		Nfqueue:      nfRaw,
		attrsChannel: nfqueueAttrChan,
	}

	err = nf.Register(context.Background(), func(a gonfqueue.Attribute) int {
		queueLength.Set(float64(len(nfqueueAttrChan)))
		select {
		case nfqueueAttrChan <- a:
		default:
			log.Warning("dropping packet because nfqueue channel is full")
			if err := nf.SetVerdict(*a.PacketID, gonfqueue.NfDrop); err != nil {
				log.WithError(err).Error("failed to set verdict for packet")
			}
		}

		return 0
	})

	if err != nil {
		return nil, err
	}

	return nf, nil
}

func (nf *nfQueue) PacketAttributesChannel() <-chan gonfqueue.Attribute {
	return nf.attrsChannel
}
