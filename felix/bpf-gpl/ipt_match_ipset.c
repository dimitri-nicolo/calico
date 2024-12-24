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
	struct ipv6hdr iph;
	int mask = 192;
	int bytes_to_load = sizeof(struct ipv6hdr);
#else
	struct iphdr iph;
	int mask = 96;
	int bytes_to_load = sizeof(struct iphdr);
#endif
	if (bpf_skb_load_bytes(skb, 0, &iph, bytes_to_load)) {
		return 1;
	}
	__be64 ipset_id = __globals.ip_set_id;
	struct ip_set_key key = {
		.mask = mask,
		.set_id = ((__be64)bpf_ntohl(ipset_id & 0xFFFFFFFF) << 32) | bpf_ntohl(ipset_id >> 32),
	};
#ifdef IPVER6
	ipv6hdr_ip_to_ipv6_addr_t(&key.addr, &iph.daddr);
#else
	key.addr = iph.daddr;
#endif
	__u32 *ret = cali_ip_sets_lookup_elem(&key);
	if (ret) {
		CALI_DEBUG("Dst IP " IP_FMT " matches ip set 0x%x", debug_ip(key.addr), ipset_id);
		return 1;
	}
	CALI_DEBUG("Dst IP " IP_FMT " does not match ip set 0x%x", debug_ip(key.addr), ipset_id);
	return 0;
}

