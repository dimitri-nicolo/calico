// Copyright (c) 2016,2019 Tigera, Inc. All rights reserved.

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
	"reflect"
)

const (
	IPAMConfigGlobalName = "default"
)

var (
	typeIPAMConfig = reflect.TypeOf(IPAMConfig{})
)

type IPAMConfigKey struct{}

func (key IPAMConfigKey) defaultPath() (string, error) {
	return "/calico/ipam/v2/config", nil
}

func (key IPAMConfigKey) defaultDeletePath() (string, error) {
	return key.defaultPath()
}

func (key IPAMConfigKey) defaultDeleteParentPaths() ([]string, error) {
	return nil, nil
}

func (key IPAMConfigKey) valueType() (reflect.Type, error) {
	return typeIPAMConfig, nil
}

func (key IPAMConfigKey) String() string {
	return "IPAMConfigKey()"
}

type IPAMConfig struct {
	StrictAffinity     bool `json:"strict_affinity,omitempty"`
	AutoAllocateBlocks bool `json:"auto_allocate_blocks,omitempty"`
}
