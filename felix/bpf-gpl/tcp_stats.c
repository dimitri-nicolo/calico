// Project Calico BPF dataplane programs.
// Copyright (c) 2020-2021 Tigera, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0 OR GPL-2.0-or-later

#include "events.h"
#include "tcp_stats.h"
#include "socket_lookup.h"
#include "globals.h"

const volatile struct cali_tc_globals __globals;
SEC("classifier/tc/calico_tcp_stats")
int calico_tcp_stats(struct __sk_buff *skb)
{
	struct cali_tc_ctx ctx = {
		.skb = skb,
	};
	if (!skb_refresh_validate_ptrs(&ctx, UDP_SIZE)) {
		if ((ctx.ip_header->ihl == 5) && (ctx.ip_header->protocol == IPPROTO_TCP)) {
			socket_lookup(&ctx);
		}
	}

	return TC_ACT_UNSPEC;
}

char ____license[] __attribute__((section("license"), used)) = "GPL";
