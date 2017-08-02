// Copyright (c) 2016 Tigera, Inc. All rights reserved.

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

package model

import (
	"fmt"
	"regexp"

	"reflect"

	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/projectcalico/libcalico-go/lib/errors"
)

var (
	matchPolicy = regexp.MustCompile("^/?calico/v1/policy/tier/([^/]+)/policy/([^/]+)$")
	typePolicy  = reflect.TypeOf(Policy{})
)

type PolicyKey struct {
	Name string `json:"-" validate:"required,name"`
	Tier string `json:"-" validate:"required,name"`
}

func (key PolicyKey) defaultPath() (string, error) {
	if key.Tier == "" {
		return "", errors.ErrorInsufficientIdentifiers{Name: "tier"}
	}
	if key.Name == "" {
		return "", errors.ErrorInsufficientIdentifiers{Name: "name"}
	}
	e := fmt.Sprintf("/calico/v1/policy/tier/%s/policy/%s",
		key.Tier, escapeName(key.Name))
	return e, nil
}

func (key PolicyKey) defaultDeletePath() (string, error) {
	return key.defaultPath()
}

func (key PolicyKey) defaultDeleteParentPaths() ([]string, error) {
	return nil, nil
}

func (key PolicyKey) valueType() reflect.Type {
	return typePolicy
}

func (key PolicyKey) String() string {
	return fmt.Sprintf("Policy(tier=%s, name=%s)", key.Tier, key.Name)
}

type PolicyListOptions struct {
	Name string
	Tier string
}

func (options PolicyListOptions) defaultPathRoot() string {
	k := "/calico/v1/policy/tier"
	if options.Tier == "" {
		return k
	}
	k = k + fmt.Sprintf("/%s/policy", options.Tier)
	if options.Name == "" {
		return k
	}
	k = k + fmt.Sprintf("/%s", escapeName(options.Name))
	return k
}

func (options PolicyListOptions) KeyFromDefaultPath(path string) Key {
	log.Debugf("Get Policy key from %s", path)
	r := matchPolicy.FindAllStringSubmatch(path, -1)
	if len(r) != 1 {
		log.Debugf("Didn't match regex")
		return nil
	}
	tier := r[0][1]
	name := unescapeName(r[0][2])
	if options.Tier != "" && tier != options.Tier {
		log.Infof("Didn't match tier %s != %s", options.Tier, tier)
		return nil
	}
	if options.Name != "" && name != options.Name {
		log.Debugf("Didn't match name %s != %s", options.Name, name)
		return nil
	}
	return PolicyKey{Tier: tier, Name: name}
}

type Policy struct {
	Order         *float64          `json:"order,omitempty" validate:"omitempty"`
	InboundRules  []Rule            `json:"inbound_rules,omitempty" validate:"omitempty,dive"`
	OutboundRules []Rule            `json:"outbound_rules,omitempty" validate:"omitempty,dive"`
	Selector      string            `json:"selector" validate:"selector"`
	DoNotTrack    bool              `json:"untracked,omitempty"`
	Annotations   map[string]string `json:"annotations,omitempty"`
	PreDNAT       bool              `json:"pre_dnat,omitempty"`
}

func (p Policy) String() string {
	parts := make([]string, 0)
	if p.Order != nil {
		parts = append(parts, fmt.Sprintf("order:%v", *p.Order))
	}
	parts = append(parts, fmt.Sprintf("selector:%#v", p.Selector))
	inRules := make([]string, len(p.InboundRules))
	for ii, rule := range p.InboundRules {
		inRules[ii] = rule.String()
	}
	parts = append(parts, fmt.Sprintf("inbound:%v", strings.Join(inRules, ";")))
	outRules := make([]string, len(p.OutboundRules))
	for ii, rule := range p.OutboundRules {
		outRules[ii] = rule.String()
	}
	parts = append(parts, fmt.Sprintf("outbound:%v", strings.Join(outRules, ";")))
	parts = append(parts, fmt.Sprintf("untracked:%v", p.DoNotTrack))
	parts = append(parts, fmt.Sprintf("pre_dnat:%v", p.PreDNAT))
	return strings.Join(parts, ",")
}
