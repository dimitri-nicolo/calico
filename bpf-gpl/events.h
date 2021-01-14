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
#include "jump.h"
#include <linux/bpf_perf_event.h>

#define TASK_COMM_LEN 16

#define EVENT_PROTO_STATS_V4	1
#define EVENT_DNS		2
#define EVENT_POLICY_VERDICT	3

struct event_proto_stats_v4 {
	struct perf_event_header hdr;
	__u32 pid;
	__u32 proto;
	__u32 saddr;
	__u32 daddr;
	__u16 sport;
	__u16 dport;
	__u32 bytes;
	__u32 sndBuf;
	__u32 rcvBuf;
	char taskName[TASK_COMM_LEN];
	__u32 isRx;
};

static CALI_BPF_INLINE void event_bpf_v4stats (struct pt_regs *ctx, __u32 pid,
					__u32 saddr, __u16 sport, __u32 daddr,
					__u16 dport, __u32 bytes, __u32 proto, __u32 isRx)
{
	struct event_proto_stats_v4 event;

	__builtin_memset(&event, 0, sizeof(event));
	event.hdr.len = sizeof(struct event_proto_stats_v4);
	event.hdr.type = EVENT_PROTO_STATS_V4;
	bpf_get_current_comm(&event.taskName, sizeof(event.taskName));
	event.pid = pid;
	event.proto = proto;
	event.saddr = bpf_ntohl(saddr);
	event.daddr = bpf_ntohl(daddr);
	event.sport = sport;
	event.dport = bpf_ntohs(dport);
	event.bytes = bytes;
	event.isRx = isRx;
	int err = perf_commit_event(ctx, &event, sizeof(event));
	if (err != 0) {
		CALI_DEBUG("kprobe: perf_commit_event returns %d\n", err);
	}
}

static CALI_BPF_INLINE void event_flow_log(struct __sk_buff *skb, struct cali_tc_state *state)
{
	state->eventhdr.type = EVENT_POLICY_VERDICT,
	state->eventhdr.len = offsetof(struct cali_tc_state, rule_ids) + sizeof(__u64) * MAX_RULE_IDS;

	/* Due to stack space limitations, the begining of the state is structured as the
	 * event and so we can send the data straight without copying in BPF.
	 */
	int err = perf_commit_event(skb, state, state->eventhdr.len);

	if (err != 0) {
		CALI_DEBUG("flowlog: perf_commit_event returns %d\n", err);
	}
}

#endif /* __CALI_EVETNS_H__ */
