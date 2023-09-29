// Project Calico BPF dataplane programs.
// Copyright (c) 2020-2023 Tigera, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0 OR GPL-2.0-or-later

#include "bpf.h"
#include "globals.h"

const volatile struct cali_stats_globals __globals;

#include "events.h"
#include "tcp_stats.h"
#include "socket_lookup.h"
#include "skb.h"

SEC("tc")
int calico_tcp_stats(struct __sk_buff *skb)
{
	struct cali_tc_ctx ctx = {
		.skb = skb,
		.ipheader_len = IP_SIZE,
	};
	if (!skb_refresh_validate_ptrs(&ctx, TCP_SIZE)) {
		if ((ip_hdr(&ctx)->ihl == 5) && (ip_hdr(&ctx)->protocol == IPPROTO_TCP)) {
			socket_lookup(&ctx);
		}
	}

	return TC_ACT_UNSPEC;
}
