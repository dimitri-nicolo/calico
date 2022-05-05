// Copyright (c) 2020 Tigera, Inc. All rights reserved.

// Package state assists in communicating with stateful go routines through channels. You simple declare a type SendToStateChan,
// have the go routine listen on that channel, and send data through that channel with the Send function. The SendToStateChan
// is a channel of SendInterface, so in the goroutine you can get the object sent with the Get() method, and return a response
// over the channel with the Return method
//
// To help with casting the interfaces sent over the channel their are helper functions in the cast.go file.
package state

import "time"

const (
	DefaultChannelWaitTimeout = 5 * time.Second
)

type ErrChannelWriteTimeout struct{}

func (err *ErrChannelWriteTimeout) Error() string {
	return "timed out writing to the sender channel"
}

type ErrChannelReadTimeout struct{}

func (err *ErrChannelReadTimeout) Error() string {
	return "timed out reading from the receiver channel"
}

// SendToStateChan is a chan SendInterface.
type SendToStateChan chan SendInterface

// SendInterface implements methods to easily communicate with a stateful goroutine
type SendInterface interface {
	Get() interface{}
	Return(interface{})
	Close()
}

type sendStruct struct {
	obj interface{}
	r   chan interface{}
}

func (s *sendStruct) Get() interface{} {
	return s.obj
}

func (s *sendStruct) Return(obj interface{}) {
	s.r <- obj
}

func (s *sendStruct) Close() {
	close(s.r)
}

// Send is a wrapper function around SendWithTimeout using the timeout defined by DefaultChannelWaitTimeout.
func Send(ch SendToStateChan, obj interface{}) interface{} {
	return SendWithTimeout(ch, obj, DefaultChannelWaitTimeout)
}

// SendWithTimeout creates an implementation of the SendInterface from the given interface and sends it over the channel.
// It waits for a maximum of duration of "timeout" to write to ch and for a response from the receiver channel in the
// sendStruct. If a timeout has occurred either ErrChannelWriteTimeout or ErrChannelReadTimeout is returned, depending
// on which channel it timeout on.
func SendWithTimeout(ch SendToStateChan, obj interface{}, timeout time.Duration) interface{} {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	r := make(chan interface{})

	select {
	case ch <- &sendStruct{obj: obj, r: r}:
	case <-timer.C:
		return new(ErrChannelWriteTimeout)
	}

	// This is needed to reset the timer
	if !timer.Stop() {
		<-timer.C
	}
	timer.Reset(timeout)

	select {
	case result := <-r:
		return result
	case <-timer.C:
		return new(ErrChannelReadTimeout)
	}
}
