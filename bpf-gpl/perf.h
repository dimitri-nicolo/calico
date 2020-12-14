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

#ifndef __CALI_PERF_H__
#define __CALI_PERF_H__

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

/* perf_commit_event commits an event with the provided data */
static CALI_BPF_INLINE int perf_commit_event(void *ctx, void *data, __u64 size)
{
	return bpf_perf_event_output(ctx, &cali_perf_evnt, BPF_F_CURRENT_CPU, data, size);
}

/* perf_commit_event_ctx commits an event and includes ctx_send_size bytes of the context */
static CALI_BPF_INLINE int perf_commit_event_ctx(void *ctx, __u32 ctx_send_size, void *data, __u64 size)
{
	__u64 flags = BPF_F_CURRENT_CPU | (((__u64)ctx_send_size << 32) & BPF_F_CTXLEN_MASK);

	return bpf_perf_event_output(ctx, &cali_perf_evnt, flags, data, size);
}

#endif /* __CALI_PERF_H__ */
