// Copyright (c) 2017-2019 Tigera, Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"math"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/containernetworking/cni/pkg/ns"
	"github.com/docopt/docopt-go"
	reuse "github.com/jbenet/go-reuseport"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/felix/fv/utils"
)

const usage = `test-connection: test connection to some target, for Felix FV testing.

Usage:
  test-connection <namespace-path> <ip-address> <port> [--source-port=<source>] [--protocol=<protocol>] [--duration=<seconds>] [--loop-with-file=<file>]

Options:
  --source-port=<source>  Source port to use for the connection [default: 0].
  --protocol=<protocol>  Protocol to test [default: tcp].
  --duration=<seconds>   Total seconds test should run. 0 means run a one off connectivity check. Non-Zero means packets loss test.[default: 0]
  --loop-with-file=<file>  Whether to send messages repeatedly, file is used for synchronization

If connection is successful, test-connection exits successfully.

If connection is unsuccessful, test-connection panics and so exits with a failure status.`

// Note about the --loop-with-file=<FILE> flag:
//
// This flag takes a path to a file as a value. The file existence is
// used as a means of synchronization.
//
// Before this program is started, the file should exist. When the
// program establishes a long-running connection and sends the first
// message, it will remove the file. That way other process can assume
// that the connection is here when the file disappears and can
// perform some checks.
//
// If the other process creates the file again, it will tell this
// program to close the connection, remove the file and quit.

func main() {
	log.SetLevel(log.DebugLevel)

	arguments, err := docopt.Parse(usage, nil, true, "v0.1", false)
	if err != nil {
		println(usage)
		log.WithError(err).Fatal("Failed to parse usage")
	}
	log.WithField("args", arguments).Info("Parsed arguments")
	namespacePath := arguments["<namespace-path>"].(string)
	ipAddress := arguments["<ip-address>"].(string)
	port := arguments["<port>"].(string)
	sourcePort := arguments["--source-port"].(string)
	protocol := arguments["--protocol"].(string)
	duration := arguments["--duration"].(string)
	seconds, err := strconv.Atoi(duration)
	if err != nil {
		// panic on error
		panic(fmt.Sprintf("invalid duration argument - %s", duration))
	}
	loopFile := ""
	if arg, ok := arguments["--loop-with-file"]; ok && arg != nil {
		loopFile = arg.(string)
	}

	log.Infof("Test connection from %v:%v to IP %v port %v proto %v max duration %d seconds", namespacePath, sourcePort, ipAddress, port, protocol, seconds)

	if loopFile == "" {
		// I found that configuring the timeouts on all the network calls was a bit fiddly.  Since
		// it leaves the process hung if one of them is missed, use a global timeout instead.
		go func() {
			timeout := time.Duration(seconds + 2)
			time.Sleep(timeout * time.Second)
			panic("Timed out")
		}()
	}

	if namespacePath == "-" {
		// Test connection from wherever we are already running.
		err = tryConnect(ipAddress, port, sourcePort, protocol, seconds, loopFile)
	} else {
		// Get the specified network namespace (representing a workload).
		var namespace ns.NetNS
		namespace, err = ns.GetNS(namespacePath)
		if err != nil {
			panic(err)
		}
		log.WithField("namespace", namespace).Debug("Got namespace")

		// Now, in that namespace, try connecting to the target.
		err = namespace.Do(func(_ ns.NetNS) error {
			return tryConnect(ipAddress, port, sourcePort, protocol, seconds, loopFile)
		})
	}

	if err != nil {
		panic(err)
	}
}

type statistics struct {
	totalReq   int
	totalReply int
}

type testConn struct {
	stat statistics

	config   utils.ConnConfig
	conn     net.Conn
	protocol string
	duration time.Duration
}

func NewTestConn(ipAddress, port, sourcePort, protocol string, duration time.Duration, loopFile string) (*testConn, error) {
	err := utils.RunCommand("ip", "r")
	if err != nil {
		return nil, err
	}

	// The reuse library implements a version of net.Dialer that can reuse UDP/TCP ports, which we
	// need in order to make connection retries work.
	var d reuse.Dialer
	var localAddr string
	var remoteAddr string
	var conn net.Conn
	if strings.Contains(ipAddress, ":") {
		localAddr = "[::]:" + sourcePort
		remoteAddr = "[" + ipAddress + "]:" + port
	} else {
		localAddr = "0.0.0.0:" + sourcePort
		remoteAddr = ipAddress + ":" + port
	}

	ls := newLoopState(loopFile)
	log.Infof("Connecting from %v to %v over %s", localAddr, remoteAddr, protocol)
	if protocol == "udp" {
		d.D.LocalAddr, _ = net.ResolveUDPAddr("udp", localAddr)
		log.WithFields(log.Fields{
			"addr":     localAddr,
			"resolved": d.D.LocalAddr,
		}).Infof("Resolved udp addr")
		conn, err := d.Dial("udp", remoteAddr)
		log.Infof(`UDP "connection" established`)
		if err != nil {
			panic(err)
		}
		defer conn.Close()

		for {
			fmt.Fprintf(conn, testMessage+"\n")
			log.WithField("message", testMessage).Info("Sent message over udp")
			reply, err := bufio.NewReader(conn).ReadString('\n')
			if err != nil {
				panic(err)
			}
			reply = strings.TrimSpace(reply)
			log.WithField("reply", reply).Info("Got reply")
			if reply != testMessage {
				panic(errors.New("Unexpected reply: " + reply))
			}
			if !ls.Next() {
				break
			}
		}
	} else {
		d.D.LocalAddr, err = net.ResolveTCPAddr("tcp", localAddr)
		if err != nil {
			return nil, err
		}
		log.WithFields(log.Fields{
			"addr":     localAddr,
			"resolved": d.D.LocalAddr,
		}).Infof("Resolved tcp addr")
		conn, err := d.Dial("tcp", remoteAddr)
		if err != nil {
			return nil, err
		}
		defer conn.Close()
		log.Infof("TCP connection established")

		for {
			fmt.Fprintf(conn, testMessage+"\n")
			log.WithField("message", testMessage).Info("Sent message over tcp")
			reply, err := bufio.NewReader(conn).ReadString('\n')
			if err != nil {
				return err
			}
			reply = strings.TrimSpace(reply)
			log.WithField("reply", reply).Info("Got reply")
			if reply != testMessage {
				return errors.New("Unexpected reply: " + reply)
			}
			if !ls.Next() {
				break
			}
		}
	}

	var connType string
	if duration == time.Duration(0) {
		connType = utils.ConnectionTypePing
	} else {
		connType = utils.ConnectionTypeStream
		if protocol != "udp" {
			panic("Wrong protocol for packets loss test")
		}
	}

	log.Infof("%s connection established from %v to %v", connType, localAddr, remoteAddr)
	return &testConn{
		config:   utils.ConnConfig{connType, uuid.NewV4().String()},
		conn:     conn,
		protocol: protocol,
		duration: duration,
	}, nil

}

func tryConnect(ipAddress, port, sourcePort, protocol string, seconds int, loopFile string) error {
	tc, err := NewTestConn(ipAddress, port, sourcePort, protocol, time.Duration(seconds)*time.Second, loopFile)
	if err != nil {
		panic(err)
	}
	defer tc.conn.Close()

	if tc.config.ConnType == utils.ConnectionTypePing {
		return tc.tryConnectOnceOff()
	}

	return tc.tryConnectWithPacketLoss()
}

func (tc *testConn) tryConnectOnceOff() error {
	testMessage := tc.config.GetTestMessage(0)

	// Write test message.
	fmt.Fprintf(tc.conn, testMessage+"\n")

	// Read back and compare.
	reply, err := bufio.NewReader(tc.conn).ReadString('\n')
	if err != nil {
		return err
	}
	reply = strings.TrimSpace(reply)
	if reply != testMessage {
		return errors.New("Unexpected reply: " + reply)
	}

	return nil
}

func (tc *testConn) tryConnectWithPacketLoss() error {
	ctx, cancel := context.WithTimeout(context.Background(), tc.duration)
	defer cancel()
	reqDone := make(chan int)

	log.Info("Start packet loss testing.")

	var wg sync.WaitGroup

	// Start a reader
	wg.Add(1)
	go func() {
		defer wg.Done()
		connReader := bufio.NewReader(tc.conn)

		lastSequence := 0
		count := 0
		outOfOrder := 0
		maxGap := 0
		for {
			select {
			case reqTotal := <-reqDone:
				log.Infof("Reader completed.total req %d, total reply %d, last reply %d, outOfOrder %d, maxGap %d",
					reqTotal, count, lastSequence, outOfOrder, maxGap)

				if count > reqTotal {
					panic("Got more packets than we sent")
				}

				tc.stat.totalReq = reqTotal
				tc.stat.totalReply = count
				return
			default:
				// Deadline is point of time. Have to set it in the loop for each read.
				tc.conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
				reply, err := connReader.ReadString('\n')
				if e, ok := err.(net.Error); ok && e.Timeout() {
					// This was a timeout. Nothing to read.
					log.Infof("Nothing to read. Total reply so far %d", count)
					continue
				} else if err != nil {
					// This is an error, not a timeout
					panic(err)
				}
				// Reset buffer for next read.
				connReader.Reset(tc.conn)

				lastSequence, err = tc.config.GetTestMessageSequence(reply)
				if err != nil {
					panic(err)
				}

				if lastSequence != count {
					outOfOrder++
					if gap := int(math.Abs(float64(lastSequence - count))); gap > maxGap {
						maxGap = gap
					}
				}

				count++
			}
		}
	}()

	// start a writer
	wg.Add(1)
	go func() {
		defer wg.Done()

		count := 0
		for {
			select {
			case <-ctx.Done():
				log.Info("Timeout for writer.")

				// Grace period for reader to finish.
				time.Sleep(200 * time.Millisecond)
				reqDone <- count
				log.Info("Asked reader to complete.")

				return
			default:
				testMessage := tc.config.GetTestMessage(count)
				fmt.Fprintf(tc.conn, testMessage+"\n")
				count++

				// Slow down sending request, otherwise we may get udp buffer overflow and loss packet,
				// which is not the right kind of packet loss we want to trace.
				// watch -n 1 'cat  /proc/net/udp' to monitor udp buffer overflow.

				// Max 5000 packets per second
				time.Sleep(200 * time.Microsecond)
			}
		}

	}()

	// Wait for writer and reader to complete.
	wg.Wait()

	log.Infof("Stat -- %s", utils.FormPacketStatString(tc.stat.totalReq, tc.stat.totalReply))

	return nil
}

type loopState struct {
	sentInitial bool
	loopFile    string
}

func newLoopState(loopFile string) *loopState {
	return &loopState{
		sentInitial: false,
		loopFile:    loopFile,
	}
}

func (l *loopState) Next() bool {
	if l.loopFile == "" {
		return false
	}

	if l.sentInitial {
		// This is after the connection was established in
		// previous iteration, so we wait for the loop file to
		// appear (it should be created by other process). If
		// the file exists, it means that the other process
		// wants us to delete the file, drop the connection
		// and quit.
		if _, err := os.Stat(l.loopFile); err != nil {
			if !os.IsNotExist(err) {
				panic(fmt.Errorf("Failed to stat loop file %s: %v", l.loopFile, err))
			}
		} else {
			if err := os.Remove(l.loopFile); err != nil {
				panic(fmt.Errorf("Could not remove loop file %s: %v", l.loopFile, err))
			}
			return false
		}
	} else {
		// A connection was just established and the initial
		// message was sent so we set the flag to true and
		// delete the loop file, so other process can continue
		// with the appropriate checks
		if err := os.Remove(l.loopFile); err != nil {
			panic(fmt.Errorf("Could not remove loop file %s: %v", l.loopFile, err))
		}
		l.sentInitial = true
	}
	time.Sleep(500 * time.Millisecond)
	return true
}
