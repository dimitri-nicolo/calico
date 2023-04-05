// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package tls

import (
	"net"
	"net/http"
)

type MultiListener interface {
	net.Listener
	Send(net.Conn)
}

// NewMultiListener implements the listneer interface, and allows us to programatically channel connections
// to the server via the Send method.
func NewMultiListener() MultiListener {
	l := &multiList{
		ch:        make(chan net.Conn, 1),
		closeChan: make(chan bool, 1),
	}
	return l
}

type multiList struct {
	ch        chan net.Conn
	closeChan chan bool
}

// Accept implements net.Listener
func (l *multiList) Accept() (net.Conn, error) {
	select {
	case conn := <-l.ch:
		return conn, nil
	case <-l.closeChan:
		return nil, http.ErrServerClosed
	}
}

func (l *multiList) Addr() net.Addr {
	return nil
}

func (l *multiList) Close() error {
	l.closeChan <- true
	return nil
}

// Send adds the given connection to the listener, so that future calls to Accept will
// accept the connection.
func (l *multiList) Send(c net.Conn) {
	l.ch <- c
}
