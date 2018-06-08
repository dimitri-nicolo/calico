// Copyright (c) 2018 Tigera, Inc. All rights reserved.
// Copyright 2017 flannel authors
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

package ipsec

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"context"

	"io/ioutil"
	"strconv"

	"strings"

	"github.com/bronze1man/goStrongswanVici"
	log "github.com/sirupsen/logrus"
)

type Uri struct {
	network, address string
}

type CharonIKEDaemon struct {
	viciUri     Uri
	espProposal string
	ikeProposal string
	ctx         context.Context
}

func NewCharonIKEDaemon(ctx context.Context, wg *sync.WaitGroup, espProposal, ikeProposal string) (*CharonIKEDaemon, error) {
	// FIXME: Reevaluate directory permissions.
	os.MkdirAll("/var/run/", 0700)
	if f, err := os.Open("/var/run/charon.pid"); err == nil {
		defer f.Close()
		bs, err := ioutil.ReadAll(f)
		if err != nil {
			return nil, err
		}
		pid, err := strconv.Atoi(strings.TrimSpace(string(bs)))
		if err != nil {
			return nil, err
		}
		log.WithField("pid", pid).Info("charon already running, killing it")
		proc, err := os.FindProcess(pid)
		if err == nil {
			err = proc.Kill()
			if err != nil {
				log.WithError(err).Error("Failed to kill old Charon")
				return nil, err
			}
		}
		os.Remove("/var/run/charon.pid")
	}

	charon := &CharonIKEDaemon{
		ctx:         ctx,
		espProposal: espProposal,
		ikeProposal: ikeProposal,
	}
	charon.viciUri = Uri{"unix", "/var/run/charon.vici"}

	cmd, err := charon.runAndCaptureLogs("/usr/lib/strongswan/charon")

	if err != nil {
		log.Errorf("Error starting charon daemon: %v", err)
		return nil, err
	} else {
		log.Info("Charon daemon started")
	}

	wg.Add(1)
	go func() {
		select {
		case <-ctx.Done():
			cmd.Process.Signal(syscall.SIGTERM)
			cmd.Wait()
			log.Infof("Stopped charon daemon")
			wg.Done()
		}
	}()
	return charon, nil
}

func (charon *CharonIKEDaemon) getClient(wait bool) (client *goStrongswanVici.ClientConn, err error) {
	for {
		socket_conn, err := net.Dial(charon.viciUri.network, charon.viciUri.address)
		if err == nil {
			return goStrongswanVici.NewClientConn(socket_conn), nil
		} else {
			if wait {
				select {
				case <-charon.ctx.Done():
					log.Error("Cancel waiting for charon")
					return nil, err
				default:
					log.Errorf("ClientConnection failed: %v", err)
				}

				log.Info("Retrying in a second ...")
				time.Sleep(time.Second)
			} else {
				return nil, err
			}
		}
	}
}

func (charon *CharonIKEDaemon) runAndCaptureLogs(execPath string) (cmd *exec.Cmd, err error) {
	path, err := exec.LookPath(execPath)
	if err != nil {
		return nil, err
	}
	cmd = &exec.Cmd{
		Path: path,
		SysProcAttr: &syscall.SysProcAttr{
			Pdeathsig: syscall.SIGTERM,
		},
	}

	// Start charon log collector
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Errorf("Error get stdout pipe: %v", err)
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Errorf("Error get sterr pipe: %v", err)
		return nil, err
	}
	go copyOutputToLog("stdout", stdout)
	go copyOutputToLog("stderr", stderr)

	err = cmd.Start()
	return
}

func (charon *CharonIKEDaemon) LoadSharedKey(remoteIP, password string) error {
	var err error
	var client *goStrongswanVici.ClientConn

	client, err = charon.getClient(true)
	if err != nil {
		log.Errorf("Failed to acquire Vici client: %v", err)
		return err
	}

	defer client.Close()

	sharedKey := &goStrongswanVici.Key{
		Typ:    "IKE",
		Data:   password,
		Owners: []string{remoteIP},
	}

	for {
		err = client.LoadShared(sharedKey)
		if err != nil {
			log.Errorf("Failed to load key for %v. Retrying. %v", remoteIP, err)
			time.Sleep(time.Second)
			continue
		}
		break
	}

	log.Infof("Loaded shared key for: %v", remoteIP)
	return nil
}

func (charon *CharonIKEDaemon) LoadConnection(localIP, remoteIP string) error {
	var err error
	var client *goStrongswanVici.ClientConn

	if localIP == "" || remoteIP == "" {
		log.WithFields(log.Fields{
			"localIP":  localIP,
			"remoteIP": remoteIP,
		}).Panic("Missing local or remote address")
	}

	client, err = charon.getClient(true)
	if err != nil {
		log.Errorf("Failed to acquire Vici client: %s", err)
		return err
	}
	defer client.Close()

	childConfMap := make(map[string]goStrongswanVici.ChildSAConf)
	childSAConf := goStrongswanVici.ChildSAConf{
		Local_ts:     []string{"0.0.0.0/0"},
		Remote_ts:    []string{"0.0.0.0/0"},
		ESPProposals: []string{charon.espProposal},
		StartAction:  "start",
		CloseAction:  "none",
		Mode:         "tunnel",
		ReqID:        fmt.Sprint(ReqID),
		//RekeyTime:     "5", //Can set this to a low time to check that rekeys are handled properly
		InstallPolicy: "no",
	}

	childSAConfName := formatName(localIP, remoteIP)
	childConfMap[childSAConfName] = childSAConf

	localAuthConf := goStrongswanVici.AuthConf{
		AuthMethod: "psk",
	}
	remoteAuthConf := goStrongswanVici.AuthConf{
		AuthMethod: "psk",
	}

	ikeConf := goStrongswanVici.IKEConf{
		LocalAddrs:  []string{localIP},
		RemoteAddrs: []string{remoteIP},
		Proposals:   []string{charon.ikeProposal},
		Version:     "2",
		KeyingTries: "0", //continues to retry
		LocalAuth:   localAuthConf,
		RemoteAuth:  remoteAuthConf,
		Children:    childConfMap,
		Encap:       "no",
		Mobike:      "no",
	}
	ikeConfMap := make(map[string]goStrongswanVici.IKEConf)

	connectionName := formatName(localIP, remoteIP)
	ikeConfMap[connectionName] = ikeConf

	err = client.LoadConn(&ikeConfMap)
	if err != nil {
		return err
	}

	log.Infof("Loaded connection: %v", connectionName)
	return nil
}

func (charon *CharonIKEDaemon) UnloadCharonConnection(localIP, remoteIP string) error {
	client, err := charon.getClient(false)
	if err != nil {
		log.Errorf("Failed to acquire Vici client: %s", err)
		return err
	}
	defer client.Close()

	connectionName := formatName(localIP, remoteIP)
	unloadConnRequest := &goStrongswanVici.UnloadConnRequest{
		Name: connectionName,
	}

	err = client.UnloadConn(unloadConnRequest)
	if err != nil {
		return err
	}

	log.Infof("Unloaded connection: %v", connectionName)
	return nil
}

func copyOutputToLog(streamName string, stream io.Reader) {
	scanner := bufio.NewScanner(stream)
	scanner.Buffer(nil, 4*1024*1024) // Increase maximum buffer size (but don't pre-alloc).
	for scanner.Scan() {
		line := scanner.Text()
		log.Info("[", streamName, "] ", line)
	}
	logCxt := log.WithFields(log.Fields{
		"name":   "charon",
		"stream": stream,
	})
	if err := scanner.Err(); err != nil {
		log.Panicf("Non-EOF error reading charon [%s], err %v", streamName, err)
	}
	logCxt.Info("Stream finished")
}

func formatName(local, remote string) string {
	return fmt.Sprintf("%s-%s", local, remote)
}
