// Project Calico BPF dataplane programs.
// Copyright (c) 2024 Tigera, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0 OR GPL-2.0-or-later

#include <linux/bpf.h>
#include <linux/if_ether.h>
#include <linux/ip.h>
#include <linux/ipv6.h>
#include "bpf.h"
#include "log.h"
#include "types.h"
#include "counters.h"
#include "ip_addr.h"
#include "parsing.h"
#include "dns_response.h"

SEC("socket")
int cali_ipt_parse_dns(struct __sk_buff *skb)
{
	struct iphdr iph;
	if (bpf_skb_load_bytes(skb, 0, &iph, sizeof(struct iphdr))) {
		return 1;
	}
	struct cali_tc_ctx ctx;
	ctx.skb = skb;
	CALI_DEBUG("IP header len %d", iph.ihl);
	ctx.ipheader_len = iph.ihl * 4;
	dns_process_datagram(&ctx);
	return 1;
}
