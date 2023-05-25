// Project Calico BPF dataplane programs.
// Copyright (c) 2020-2023 Tigera, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0 OR GPL-2.0-or-later

#include <linux/ipv6.h>

#include "bpf.h"
#include "globals.h"

const volatile struct cali_stats_globals __globals;

#include "events.h"
#include "tcp_stats.h"
#include "socket_lookup.h"
#include "skb.h"
#undef GLOBAL_FLAGS
#define GLOBAL_FLAGS 0
#define deny_reason(...) /* used in parsin.h but only for tc.c, we do not care here */
#include "parsing.h"

SEC("tc")
int calico_tcp_stats(struct __sk_buff *skb)
{
	__u8 scratch[] = { /* zero it to shut up verifier */ };

	struct cali_tc_ctx ctx = {
		.skb = skb,
		.ipheader_len = IP_SIZE,
		.scratch = (void *)scratch,
	};

	switch (parse_packet_ip(&ctx)) {
	case PARSING_OK:
		// IPv4 Packet.
		break;
	default:
		goto skip;
	}

	tc_state_fill_from_iphdr(&ctx);
	if (ip_hdr(&ctx)->protocol == IPPROTO_TCP) {
		if (tc_state_fill_from_nexthdr(&ctx, false) == PARSING_ERROR) {
			goto skip;
		}
		socket_lookup(&ctx);
	}

skip:
	return TC_ACT_UNSPEC;
}
