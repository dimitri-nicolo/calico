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

#ifndef __CALI_DNS_H__
#define __CALI_DNS_H__

#include "bpf.h"
#include "perf.h"
#include "events.h"
#include "sock.h"
#include <linux/bpf_perf_event.h>

static CALI_BPF_INLINE void calico_report_dns(struct cali_tc_ctx *ctx)
{
	int plen = ctx->skb->len;
	struct perf_event_timestamp_header hdr;
	__builtin_memset(&hdr, 0, sizeof(hdr));
	hdr.h.type = EVENT_DNS;
	hdr.h.len = sizeof(hdr) + plen;
	hdr.timestamp_ns = bpf_ktime_get_ns();
	int err = perf_commit_event_ctx(ctx->skb, plen, &hdr, sizeof(hdr));
	if (err) {
		CALI_DEBUG("perf_commit_event_ctx error %d\n", err);
	}
}

static CALI_BPF_INLINE void calico_check_for_dns(struct cali_tc_ctx *ctx)
{
	// Support UDP only; bail for TCP or any other IP protocol.
	if (ctx->state->ip_proto != IPPROTO_UDP) {
		return;
	}

	// Get the sending socket's cookie.  We need this to look up the apparent
	// destination IP and port in the cali_v4_srmsg map, to discover if a DNAT was
	// performed by our connect-time load balancer.
	__u64 cookie = bpf_get_socket_cookie(ctx->skb);
	if (!cookie) {
		CALI_DEBUG("failed to get socket cookie for possible DNS request");
		return;
	}
	CALI_DEBUG("Got socket cookie 0x%lx for possible DNS\n", cookie);

	// Lookup dst IP and port in cali_v4_srmsg map.  If there's a hit, henceforth use
	// dst IP and port from the map entry.  (Hit implies that a DNAT has already
	// happened, because of CTLB being in use, but now we have the pre-DNAT IP and
	// port.  Miss implies that CTLB isn't in use or DNAT hasn't happened yet; either
	// way the message in hand already had the dst IP and port that we need.)
	__be32 dst_ip = ctx->state->ip_dst;
	__be16 dst_port = ctx->state->dport;
	struct sendrecv4_key key = {
		.ip	= dst_ip,
		.port	= dst_port,
		.cookie	= cookie,
	};
	struct sendrecv4_val *revnat = cali_v4_srmsg_lookup_elem(&key);
	if (revnat) {
		CALI_DEBUG("Got cali_v4_srmsg entry\n");
		dst_ip = revnat->ip;
		dst_port = bpf_htons(ctx_port_to_host(revnat->port));
	} else {
		CALI_DEBUG("No cali_v4_srmsg entry\n");
	}
	CALI_DEBUG("Now have dst IP 0x%x port %d\n", bpf_ntohl(dst_ip), bpf_ntohs(dst_port));

	// Compare dst IP and port against 'ipset' for trusted DNS servers.
	union ip4_set_lpm_key sip;
	__builtin_memset(&sip, 0, sizeof(sip));
	sip.ip.mask = 32 /* IP prefix length */ + 64 /* Match ID */ + 16 /* Match port */ + 8 /* Match protocol */;
	sip.ip.set_id = bpf_cpu_to_be64(TRUSTED_DNS_SERVERS_ID);
	sip.ip.addr = dst_ip;
	sip.ip.port = bpf_ntohs(dst_port);
	sip.ip.protocol = 17;
	if (bpf_map_lookup_elem(&cali_v4_ip_sets, &sip)) {
		CALI_DEBUG("Dst IP/port are trusted for DNS\n");
		// Store 'trusted DNS connection' status in conntrack entry.
		ctx->state->ct_result.flags |= CALI_CT_FLAG_TRUST_DNS;
		// Emit event to pass (presumed) DNS request up to Felix
		// userspace.
		CALI_DEBUG("report probable DNS request\n");
		calico_report_dns(ctx);
	} else {
		CALI_DEBUG("Dst IP/port are not trusted for DNS\n");
	}

	return;
}

#endif /* __CALI_DNS_H__ */
