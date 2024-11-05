// Copyright 2024 Tigera Inc. All rights reserved.

package health

import (
	"context"
	"errors"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

const (
	PingChannelClosed = "ping_channel_closed"
	PingChannelBusy   = "ping_channel_busy"
)

// PingPonger is responsible for sending a Ponger on a ping channel and listen for response on pong channel.
// If PingPonger is closed, it sends an error on receiving a Ping.
type PingPonger interface {
	Ping(context.Context) error
	ListenForPings() chan Ponger
	Close()
}

// NewPingPonger initiates and returns a PingPonger.
func NewPingPonger() PingPonger {
	return &pingPonger{
		pings:           make(chan Ponger, 1),
		closePingPonger: make(chan struct{}),
	}
}

type pingPonger struct {
	pings           chan Ponger
	closePingPonger chan struct{}
}

// Ping sends a Ponger on the ping channel and listen for response on pong channel.
// If closePingPonger channel is closed, return error and close the ping channel.
func (p *pingPonger) Ping(ctx context.Context) error {
	select {
	case _, ok := <-p.closePingPonger:
		if !ok {
			log.Warnf("closing the pings channel for expired pingPonger %+v", p)
			close(p.pings)
			return k8serrors.NewResourceExpired(PingChannelClosed)
		}
	default:
	}

	ponger := newPonger()
	select {
	case p.pings <- ponger:
	case <-ctx.Done():
		return ctx.Err()
	default:
		return k8serrors.NewInternalError(errors.New(PingChannelBusy))
	}

	// Wait for the pong or return an error if the context finishes before we receive one
	t := time.NewTicker(5 * time.Second)
	select {
	case <-t.C:
		return fmt.Errorf("%v", "error timeout: ping never returned a pong")
	case <-ctx.Done():
		return ctx.Err()
	case <-ponger.WaitForPong():
		return nil
	}
}

func (p *pingPonger) ListenForPings() chan Ponger {
	return p.pings
}

// Close sends on closePingPonger channel and closes it.
func (p *pingPonger) Close() {
	defer close(p.closePingPonger)
	select {
	case p.closePingPonger <- struct{}{}:
	default:
	}
}

type Ponger interface {
	Pong()
	WaitForPong() chan struct{}
}

type ponger struct {
	pongChan chan struct{}
}

func newPonger() Ponger {
	return &ponger{
		pongChan: make(chan struct{}),
	}
}

func (p *ponger) Pong() {
	defer close(p.pongChan)
	select {
	case p.pongChan <- struct{}{}:
	default:
	}
}

func (p *ponger) WaitForPong() chan struct{} {
	return p.pongChan
}
