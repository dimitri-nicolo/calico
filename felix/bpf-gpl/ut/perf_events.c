// Project Calico BPF dataplane programs.
// Copyright (c) 2020-2021 Tigera, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0 OR GPL-2.0-or-later

#include "ut.h"
#include "bpf.h"
#include "perf.h"

#include <linux/ip.h>
#include <linux/udp.h>

struct tuple {
	struct perf_event_header hdr;
	__u32 ip_src;
	__u32 ip_dst;
	__u16 port_src;
	__u16 port_dst;
	__u8 proto;
	__u8 _pad[3];
};

static CALI_BPF_INLINE int calico_unittest_entry (struct __sk_buff *skb)
{
	int err;
	struct cali_tc_ctx ctx = {
		.skb = skb,
		.ipheader_len = IP_SIZE,
	};

	if (skb_refresh_validate_ptrs(&ctx, UDP_SIZE)) {
		ctx.fwd.reason = CALI_REASON_SHORT;
		CALI_DEBUG("Too short\n");
		return -1;
	}
	struct iphdr *ip = ctx.ip_header;

	struct tuple tp = {
		.hdr = {
			.type = 0xdead,
			.len = sizeof(struct tuple),
		},
		.ip_src = bpf_ntohl(ip->saddr),
		.ip_dst = bpf_ntohl(ip->daddr),
		.proto = ip->protocol,
	};

	switch (ip->protocol) {
	case IPPROTO_TCP:
		{
			struct tcphdr *tcp = (void*)(ip + 1);
			tp.port_src = bpf_ntohs(tcp->source);
			tp.port_dst = bpf_ntohs(tcp->dest);
		}
		break;
	case IPPROTO_UDP:
		{
			struct udphdr *udp = (void*)(ip + 1);
			tp.port_src = bpf_ntohs(udp->source);
			tp.port_dst = bpf_ntohs(udp->dest);
		}
		break;
	}

	if (ip->protocol == IPPROTO_ICMP) {
		tp.hdr.type++;
		tp.hdr.len += skb->len;
		err = perf_commit_event_ctx(skb, skb->len, &tp, sizeof(tp));
	} else {
		err = perf_commit_event(skb, &tp, sizeof(tp));
	}
	CALI_DEBUG("perf_commit_event returns %d\n", err);

	return err == 0 ? TC_ACT_UNSPEC : TC_ACT_SHOT;
}
