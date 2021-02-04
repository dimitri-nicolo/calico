// +build !windows

// Copyright (c) 2021 Tigera, Inc. All rights reserved.
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

package kprobe

import (
	"github.com/projectcalico/felix/bpf"
)

const statsKeySize = 44
const statsValueSize = 16

var KpStatsMapParameters = bpf.MapParameters{
	Filename:   "/sys/fs/bpf/tc/globals/cali_kpstats",
	Type:       "lru_hash",
	KeySize:    statsKeySize,
	ValueSize:  statsValueSize,
	MaxEntries: 511000,
	Name:       "cali_kpstats",
	Version:    2,
}

func MapKpStats(mc *bpf.MapContext) bpf.Map {
	return mc.NewPinnedMap(KpStatsMapParameters)
}
