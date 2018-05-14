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
	"fmt"
	"net"

	log "github.com/golang/glog"
	"github.com/vishvananda/netlink"
)

func AddXFRMPolicy(src, dst, tunnelLeft, tunnelRight string, dir netlink.Dir, reqID int) error {
	_, srcNet, _ := net.ParseCIDR(src)
	_, dstNet, _ := net.ParseCIDR(dst)
	policy := netlink.XfrmPolicy{
		Src: srcNet,
		Dst: dstNet,
		Dir: dir,
	}

	tmpl := netlink.XfrmPolicyTmpl{
		Src:   net.ParseIP(tunnelLeft),
		Dst:   net.ParseIP(tunnelRight),
		Proto: netlink.XFRM_PROTO_ESP,
		Mode:  netlink.XFRM_MODE_TUNNEL,
		Reqid: reqID,
	}

	log.Infof("Adding ipsec policy: %+v", tmpl)

	policy.Tmpls = append(policy.Tmpls, tmpl)

	if err := netlink.XfrmPolicyAdd(&policy); err != nil {
		fmt.Println(fmt.Errorf("error adding policy: %+v err: %v", policy, err))
	}

	return nil
}

func DeleteXFRMPolicy(src, dst, tunnelLeft, tunnelRight string, dir netlink.Dir, reqID int) error {
	_, srcNet, _ := net.ParseCIDR(src)
	_, dstNet, _ := net.ParseCIDR(dst)

	policy := netlink.XfrmPolicy{
		Src: srcNet,
		Dst: dstNet,
		Dir: dir,
	}

	tmpl := netlink.XfrmPolicyTmpl{
		Src:   net.ParseIP(tunnelLeft),
		Dst:   net.ParseIP(tunnelRight),
		Proto: netlink.XFRM_PROTO_ESP,
		Mode:  netlink.XFRM_MODE_TUNNEL,
		Reqid: reqID,
	}

	log.Infof("Deleting ipsec policy: %+v", tmpl)

	policy.Tmpls = append(policy.Tmpls, tmpl)

	if err := netlink.XfrmPolicyDel(&policy); err != nil {
		return fmt.Errorf("error deleting policy: %+v err: %v", policy, err)
	}

	return nil
}
