// Project Calico BPF dataplane programs.
// Copyright (c) 2021 Tigera, Inc. All rights reserved.
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

#ifndef __CALI_EVENTS_KPROBE_H__
#define __CALI_EVENTS_KPROBE_H__

#include "log.h"
#include "bpf.h"
#include "perf.h"
#include "sock.h"
#include <linux/bpf_perf_event.h>
#include "events_type.h"

#define TASK_COMM_LEN 16


struct event_proto_stats {
	struct perf_event_header hdr;
	__u32 pid;
	__u32 proto;
	__u8  saddr[16];
	__u8  daddr[16];
	__u16 sport;
	__u16 dport;
	__u32 bytes;
	__u32 sndBuf;
	__u32 rcvBuf;
	char taskName[TASK_COMM_LEN];
	__u32 isRx;
};

static CALI_BPF_INLINE int event_bpf_stats(struct pt_regs *ctx, __u32 pid,
					      __u8 *saddr, __u16 sport, __u8 *daddr,
					      __u16 dport, __u32 bytes, __u32 proto, __u32 isRx)
{
	struct event_proto_stats event = {
		.hdr.len = sizeof(struct event_proto_stats),
		.hdr.type = EVENT_PROTO_STATS,
		.pid = pid,
		.proto = proto,
		.sport = sport,
		.dport = bpf_ntohs(dport),
		.bytes = bytes,
		.isRx = isRx,
	};

	bpf_get_current_comm(&event.taskName, sizeof(event.taskName));
	__builtin_memcpy(event.saddr, saddr, 16);
	__builtin_memcpy(event.daddr, daddr, 16);
	int err = perf_commit_event(ctx, &event, sizeof(event));
	if (err != 0) {
		CALI_DEBUG("event_proto_stats: perf_commit_event returns %d\n", err);
	}

	return err;
}

#endif /* __CALI_EVENTS_KPROBE_H__ */
