// Copyright (c) 2020 Tigera, Inc. All rights reserved.
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

package commands

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/projectcalico/felix/bpf"

	"github.com/projectcalico/felix/bpf/conntrack"

	"github.com/docopt/docopt-go"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	conntrackCmd.AddCommand(newConntrackDumpCmd())
	conntrackCmd.AddCommand(newConntrackRemoveCmd())
	rootCmd.AddCommand(conntrackCmd)
}

// conntrackCmd represents the conntrack command
var conntrackCmd = &cobra.Command{
	Use:   "conntrack",
	Short: "Manipulates connection tracking",
}

type conntrackDumpCmd struct {
	*cobra.Command
}

func newConntrackDumpCmd() *cobra.Command {
	cmd := &conntrackDumpCmd{
		Command: &cobra.Command{
			Use:   "dump",
			Short: "Dumps connection tracking table",
		},
	}

	cmd.Command.Args = cmd.Args
	cmd.Command.Run = cmd.Run

	return cmd.Command
}

func (cmd *conntrackDumpCmd) Args(c *cobra.Command, args []string) error {
	a, err := docopt.ParseArgs(makeDocUsage(c), args, "")
	if err != nil {
		return errors.New(err.Error())
	}

	err = a.Bind(cmd)
	if err != nil {
		return errors.New(err.Error())
	}

	return nil
}

func (cmd *conntrackDumpCmd) Run(c *cobra.Command, _ []string) {
	mc := &bpf.MapContext{}
	ctMap := conntrack.Map(mc)
	if err := ctMap.Open(); err != nil {
		log.WithError(err).Error("Failed to access ConntrackMap")
	}
	err := ctMap.Iter(func(k, v []byte) {
		var ctKey conntrack.Key
		if len(k) != len(ctKey) {
			log.Panic("Key has unexpected length")
		}
		copy(ctKey[:], k[:])

		var ctVal conntrack.Value
		if len(v) != len(ctVal) {
			log.Panic("Value has unexpected length")
		}
		copy(ctVal[:], v[:])

		fmt.Printf("%v -> %v", ctKey, ctVal)
		dumpExtra(ctKey, ctVal)
		fmt.Printf("\n")
	})
	if err != nil {
		log.WithError(err).Fatal("Failed to iterate over conntrack entries")
	}
}

func dumpExtra(k conntrack.Key, v conntrack.Value) {
	now := bpf.KTimeNanos()

	fmt.Printf(" Age: %s Active ago %s",
		time.Duration(now-v.Created()), time.Duration(now-v.LastSeen()))

	if k.Proto() != conntrack.ProtoTCP {
		return
	}

	if v.Type() == conntrack.TypeNATForward {
		return
	}

	data := v.Data()

	if (v.IsForwardDSR() && data.FINsSeenDSR()) || data.FINsSeen() {
		fmt.Printf(" CLOSED")
		return
	}

	if data.Established() {
		fmt.Printf(" ESTABLISHED")
		return
	}

	fmt.Printf(" SYN-SENT")
}

type conntrackRemoveCmd struct {
	*cobra.Command

	Proto string `docopt:"<proto>"`
	IP1   string `docopt:"<ip1>"`
	IP2   string `docopt:"<ip2>"`

	proto uint8
	ip1   net.IP
	ip2   net.IP
}

func newConntrackRemoveCmd() *cobra.Command {
	cmd := &conntrackRemoveCmd{
		Command: &cobra.Command{
			Use:   "remove <proto> <ip1> <ip2>",
			Short: "removes connection tracking",
		},
	}

	cmd.Command.Args = cmd.Args
	cmd.Command.Run = cmd.Run

	return cmd.Command
}

func (cmd *conntrackRemoveCmd) Args(c *cobra.Command, args []string) error {
	a, err := docopt.ParseArgs(makeDocUsage(c), args, "")
	if err != nil {
		return errors.New(err.Error())
	}

	err = a.Bind(cmd)
	if err != nil {
		return errors.New(err.Error())
	}

	switch proto := strings.ToLower(args[0]); proto {
	case "udp":
		cmd.proto = 17
	case "tcp":
		cmd.proto = 6
	default:
		return errors.Errorf("unknown protocol %s", proto)
	}

	cmd.ip1 = net.ParseIP(cmd.IP1)
	if cmd.ip1 == nil {
		return errors.Errorf("ip1: %q is not an ip", cmd.IP1)
	}

	cmd.ip2 = net.ParseIP(cmd.IP2)
	if cmd.ip2 == nil {
		return errors.Errorf("ip2: %q is not an ip", cmd.IP2)
	}

	return nil
}

func (cmd *conntrackRemoveCmd) Run(c *cobra.Command, _ []string) {
	mc := &bpf.MapContext{}
	ctMap := conntrack.Map(mc)
	if err := ctMap.Open(); err != nil {
		log.WithError(err).Error("Failed to access ConntrackMap")
	}
	var keysToRemove []conntrack.Key
	err := ctMap.Iter(func(k, v []byte) {
		var ctKey conntrack.Key
		if len(k) != len(ctKey) {
			log.Panic("Key has unexpected length")
		}
		copy(ctKey[:], k[:])

		log.Infof("Examining conntrack key: %v", ctKey)

		if ctKey.Proto() != cmd.proto {
			return
		}

		if ctKey.AddrA().Equal(cmd.ip1) && ctKey.AddrB().Equal(cmd.ip2) {
			log.Info("Match")
			keysToRemove = append(keysToRemove, ctKey)
		} else if ctKey.AddrB().Equal(cmd.ip1) && ctKey.AddrA().Equal(cmd.ip2) {
			log.Info("Match")
			keysToRemove = append(keysToRemove, ctKey)
		}
	})
	if err != nil {
		log.WithError(err).Fatal("Failed to iterate over conntrack entries")
	}

	for _, k := range keysToRemove {
		err := ctMap.Delete(k[:])
		if err != nil {
			log.WithError(err).WithField("key", k).Warning("Failed to delete entry from map")
		}
	}
}
