// Project Calico BPF dataplane programs.
// Copyright (c) 2024 Tigera, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0 OR GPL-2.0-or-later

#include <linux/bpf.h>
#include <stdbool.h>
#include <linux/ip.h>
#include <linux/ipv6.h>
#include "bpf.h"
#include "ip_addr.h"
#include "globals.h"
#include "log.h"
#include "policy.h"

const volatile struct cali_ipt_dns_globals __globals;

SEC("socket")
int cali_ipt_match_ipset(struct __sk_buff *skb)
{
#ifdef IPVER6
	int mask = 192;
	int offset = offsetof(struct ipv6hdr, daddr);
#else
	int mask = 96;
	int offset = offsetof(struct iphdr, daddr);
#endif
	struct ip_set_key key = {
		.mask = mask,
		.set_id = bpf_be64_to_cpu(__globals.ip_set_id),
	};

	if (bpf_skb_load_bytes(skb, offset, &key.addr, sizeof(ipv46_addr_t))) {
		return 1;
	}
	__u32 *ret = cali_ip_sets_lookup_elem(&key);
	if (ret) {
		CALI_DEBUG("Dst IP " IP_FMT " matches ip set 0x%x", debug_ip(key.addr), __globals.ip_set_id);
		return 1;
	}
	CALI_DEBUG("Dst IP " IP_FMT " does not match ip set 0x%x", debug_ip(key.addr), __globals.ip_set_id);
	return 0;
}

