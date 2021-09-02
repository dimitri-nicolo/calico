// Copyright (c) 2019-2021 Tigera, Inc. All rights reserved.

package intdataplane

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"syscall"

	log "github.com/sirupsen/logrus"
)

const SockAddr = "/var/run/calico/felix/debug"

// startDebugConsole opens a unix socket (at the location of SockAddr) and listens for commands over that socket.
// Commands can be sent to the debug console using netcat as follows:
//
//   echo <command> <arguments> | nc -U /var/run/calico/felix/debug
//
// If the command was executed successfully "success" will be printed out, otherwise "fail: <error-message>" will be
// printed.
//
// Available commands:
//
// close-fd <file-descriptor-id>: close the file descriptor specified by <file-descriptor-id>. If felix doesn't own the
//     given file descriptor the behaviour is undefined.
func startDebugConsole() {
	if err := os.MkdirAll(SockAddr, os.ModeSocket); err != nil {
		log.WithError(err).Fatal("failed to create socket path")
	}

	if err := os.RemoveAll(SockAddr); err != nil {
		log.WithError(err).Fatal("failed to remove socket")
	}

	l, err := net.Listen("unix", SockAddr)
	if err != nil {
		log.WithError(err).Error("failed to listen on socket")
		return
	}
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			log.WithError(err).Error("failed to accept connections")
			return
		}

		go runCommand(conn)
	}
}

func runCommand(c net.Conn) {
	var bytes [2048]byte

	i, err := c.Read(bytes[:])
	if err != nil {
		log.WithError(err).Error("failed to read request")
		return
	}

	args := strings.Split(string(bytes[0:i-1]), " ")

	if len(args) < 1 {
		if _, err := c.Write([]byte("no arguments passed")); err != nil {
			log.WithError(err).Error("failed to write response")
		}
	}

	switch args[0] {
	case "close-fd":
		fd, err := strconv.Atoi(args[1])
		if err != nil {
			log.WithError(err).Error("file descriptor id is not an integer")
			break
		}

		if err := syscall.Close(fd); err != nil {
			log.WithError(err).Error("failed to close fd")
			if _, err := c.Write([]byte("fail: failed to close fd")); err != nil {
				log.WithError(err).Error("failed to write response")
			}
		}
	default:
		msg := fmt.Sprintf("unknown command %s", args[0])
		if _, err := c.Write([]byte(fmt.Sprintf("fail: %s", msg))); err != nil {
			log.WithError(err).Error("failed to write response")
		}
	}

	if _, err := c.Write([]byte("success")); err != nil {
		log.WithError(err).Error("failed to write response")
	}

	if err := c.Close(); err != nil {
		log.WithError(err).Error("failed to close connection")
	}
}
