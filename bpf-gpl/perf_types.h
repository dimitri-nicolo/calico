// Project Calico BPF dataplane programs.
// Copyright (c) 2020 Tigera, Inc. All rights reserved.
//
// This program is free software; you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation; either version 2 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License along
// with this program; if not, write to the Free Software Foundation, Inc.,
// 51 Franklin Street, Fifth Floor, Boston, MA 02110-1301 USA.

#ifndef __CALI_PERF_TYPES_H__
#define __CALI_PERF_TYPES_H__

#include "bpf.h"

struct bpf_map_def_extended __attribute__((section("maps"))) cali_perf_evnt = {
	.type = BPF_MAP_TYPE_PERF_EVENT_ARRAY,
	.key_size = 4,
	.value_size = 4,
	.max_entries = 512,
	CALI_MAP_TC_EXT_PIN(MAP_PIN_GLOBAL)
};

/* We need the header to be 64bit of size so that any 64bit fields in the
 * message structures that embed this header are also aligned.
 */
struct perf_event_header {
	__u32 type;
	__u32 len;
};

struct perf_event_timestamp_header {
	struct perf_event_header h;
	__u64 timestamp_ns;
};

#endif /* __CALI_PERF_TYPES_H__ */
