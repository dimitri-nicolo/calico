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

#ifndef __CALI_EVETNS_H__
#define __CALI_EVETNS_H__

#include "bpf.h"
#include "perf.h"
#include "sock.h"
#include <linux/bpf_perf_event.h>

#define TASK_COMM_LEN 16

struct event_proto_stats_v4 {
	struct perf_event_header hdr;
	__u32 pid;
	__u32 proto;
	__u32 saddr;
	__u32 daddr;
	__u16 sport;
	__u16 dport;
	__u32 txBytes;
	__u32 rxBytes;
	__u32 sndBuf;
	__u32 rcvBuf;
	char taskName[TASK_COMM_LEN];
};

static CALI_BPF_INLINE void event_bpf_v4stats (struct pt_regs *ctx, __u32 pid,
					__u32 saddr, __u16 sport, __u32 daddr,
					__u16 dport, __u32 txBytes, __u32 rxBytes, __u32 proto)
{
	struct event_proto_stats_v4 event;

	__builtin_memset(&event, 0, sizeof(event));
	event.hdr.len = sizeof(struct event_proto_stats_v4);
	event.hdr.type = BPF_EVENT_PROTO_STATS_V4;
	bpf_get_current_comm(&event.taskName, sizeof(event.taskName));
	event.pid = pid;
	event.proto = proto;
	event.saddr = bpf_ntohl(saddr);
	event.daddr = bpf_ntohl(daddr);
	event.sport = sport;
	event.dport = bpf_ntohs(dport);
	event.txBytes = txBytes;
	event.rxBytes = rxBytes;
	int err = perf_commit_event(ctx, &event, sizeof(event));
	if (err != 0) {
		CALI_DEBUG("kprobe: perf_commit_event returns %d\n", err);
	}
}

#endif /* __CALI_EVETNS_H__ */
