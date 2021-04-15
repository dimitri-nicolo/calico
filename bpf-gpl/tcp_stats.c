// Project Calico BPF dataplane programs.
// Copyright (c) 2020-2021 Tigera, Inc. All rights reserved.
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

#include "events.h"
#include "tcp_stats_iptables.h"
#include "socket_lookup.h"

__attribute__((section("calico_tcp_stats")))
int tc_calico_entry(struct __sk_buff *skb)
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
