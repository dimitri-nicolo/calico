// Copyright (c) 2019 Tigera, Inc. All rights reserved.
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

package calc

import (
	"strings"

	"gopkg.in/tchap/go-patricia.v2/patricia"

	"github.com/projectcalico/felix/ip"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/libcalico-go/lib/set"
)

//Node is represented by cidr as KEY and networkset endpoint data stored in edLists
type IPTrieNode struct {
	cidr    ip.CIDR
	edLists []*EndpointData
}

//Root of IpTree
type IpTrie struct {
	lpmCache      *patricia.Trie
	existingCidrs set.Set
}

//NewIpTrie creates new Patricia trie and Initializes
func NewIpTrie() *IpTrie {
	return &IpTrie{
		lpmCache:      patricia.NewTrie(),
		existingCidrs: set.New(),
	}
}

//newIPTrieNode Function creates new empty node, as place holder for Key and endpoint data
func newIPTrieNode(cidr ip.CIDR, ed *EndpointData) *IPTrieNode {
	return &IPTrieNode{cidr: cidr, edLists: []*EndpointData{ed}}
}

//GetLongestPrefixCidr finds longest prefix match CIDR for the Given IP addr
//and if successful return the last endpoint data
func (t *IpTrie) GetLongestPrefixCidr(ipAddr ip.Addr) (*EndpointData, bool) {
	var longestPrefix patricia.Prefix
	var longestItem patricia.Item
	ptrie := t.lpmCache

	ptrie.VisitPrefixes(patricia.Prefix(ipAddr.AsBinary()),
		func(prefix patricia.Prefix, item patricia.Item) error {
			if len(prefix) > len(longestPrefix) {
				longestPrefix = prefix
				longestItem = item
			}
			return nil
		})
	if longestItem == nil {
		return nil, false
	}
	node := longestItem.(*IPTrieNode)
	return node.edLists[len(node.edLists)-1], true
}

//GetNetworksets return list of Endpoint data for the Given CIDR
func (t *IpTrie) GetNetworksets(cidr ip.CIDR) ([]*EndpointData, bool) {
	ptrie := t.lpmCache
	cidrb := cidr.AsBinary()
	val := ptrie.Get(patricia.Prefix(cidrb))

	if val != nil {
		node := val.(*IPTrieNode)
		return node.edLists, true
	}

	return nil, false
}

//DeleteNetworkset walk through the trie, finds the key CIDR and delete corresponding
//networkSet
func (t *IpTrie) DeleteNetworkset(cidr ip.CIDR, key model.Key) {
	ptrie := t.lpmCache
	cidrb := cidr.AsBinary()

	val := ptrie.Get(patricia.Prefix(cidrb))

	if val == nil {
		return
	}
	node := val.(*IPTrieNode)
	if len(node.edLists) == 1 {
		t.existingCidrs.Discard(cidr)
		ptrie.Delete(patricia.Prefix(cidrb))
	} else {
		ii := 0
		for _, val := range node.edLists {
			if val.Key != key {
				node.edLists[ii] = val
				ii++
			}
		}
		node.edLists = node.edLists[:ii]
	}
}

//InsertNetworkset Inserts the given CIDR in Trie and store networkset in List
//Check if this CIDR already has a corresponding networkset.
//If it has one, then append the networkset to it.
//Else, create a new CIDR to networkset
func (t *IpTrie) InsertNetworkset(cidr ip.CIDR, ed *EndpointData) {
	ptrie := t.lpmCache
	cidrb := cidr.AsBinary()

	t.existingCidrs.Add(cidr)
	val := ptrie.Get(patricia.Prefix(cidrb))
	if val == nil {
		newNode := newIPTrieNode(cidr, ed)
		ptrie.Insert(patricia.Prefix(cidrb), newNode)
	} else {
		node := val.(*IPTrieNode)
		isExistingNetset := false
		for i, val := range node.edLists {
			if ed.Key == val.Key {
				node.edLists[i] = ed
				isExistingNetset = true
				break
			}
		}
		if !isExistingNetset {
			node.edLists = append(node.edLists, ed)
		}
	}
	return
}

//DumpCIDRNetworksets returns slices of string with Cidr and
//corresponding networksetNames
func (t *IpTrie) DumpCIDRNetworksets() []string {
	ec := t.existingCidrs
	lines := []string{}
	ec.Iter(func(item interface{}) error {
		cidr := item.(ip.CIDR)
		edNames := []string{}
		eds, _ := t.GetNetworksets(cidr)
		for _, ed := range eds {
			edNames = append(edNames, ed.Key.(model.NetworkSetKey).Name)
		}
		lines = append(lines, cidr.String()+": "+strings.Join(edNames, ","))

		return nil
	})

	return lines
}
