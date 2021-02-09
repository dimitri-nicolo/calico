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

#ifndef __CALI_LOOKUP_H__
#define __CALI_LOOKUP_H__

#define SEND_TCP_STATS_INTERVAL 500000

static CALI_BPF_INLINE void socket_lookup(struct cali_tc_ctx *ctx) {
	struct bpf_sock *sk = NULL;
	struct bpf_sock_tuple tuple={};
	struct bpf_tcp_sock *tsk = NULL;
	__u16 sk_dport = 0;
	bool srcLTDest = 0;
	struct calico_ct_key k = {};
	struct calico_ct_value *v = NULL;
	
	if (CALI_F_FROM_WEP) {
		tuple.ipv4.saddr = ctx->ip_header->daddr;
		tuple.ipv4.daddr = ctx->ip_header->saddr;
		tuple.ipv4.sport = ctx->tcp_header->dest;
		tuple.ipv4.dport = ctx->tcp_header->source;
	} else if (CALI_F_TO_WEP) {
		tuple.ipv4.saddr = ctx->ip_header->saddr;
		tuple.ipv4.daddr = ctx->ip_header->daddr;
		tuple.ipv4.sport = ctx->tcp_header->source;
		tuple.ipv4.dport = ctx->tcp_header->dest;
	}
	
	sk = bpf_sk_lookup_tcp(ctx->skb, &tuple, sizeof(tuple.ipv4), IF_NS, 0);
	if (!sk) {
		tuple.ipv6.saddr[0] = tuple.ipv6.saddr[1] = tuple.ipv6.daddr[0] = tuple.ipv6.daddr[1] = 0;
		tuple.ipv6.saddr[2] = tuple.ipv6.daddr[2] = 0x0000ffff;
		CALI_DEBUG("Sridharv6:\n");
		if (CALI_F_FROM_WEP) {
			tuple.ipv6.saddr[3] = ctx->ip_header->daddr;
			tuple.ipv6.daddr[3] = ctx->ip_header->saddr;
			tuple.ipv6.sport = ctx->tcp_header->dest;
			tuple.ipv6.dport = ctx->tcp_header->source;
		} else if (CALI_F_TO_WEP) {
			tuple.ipv6.saddr[3] = ctx->ip_header->saddr;
			tuple.ipv6.daddr[3] = ctx->ip_header->daddr;
			tuple.ipv6.sport = ctx->tcp_header->source;
			tuple.ipv6.dport = ctx->tcp_header->dest;
		}
		sk = bpf_sk_lookup_tcp(ctx->skb, &tuple, sizeof(tuple.ipv6), IF_NS, 0);
		if (sk && sk->family == 10) {
			CALI_DEBUG("Socketv6 addr 0x%x 0x%x\n", sk->src_ip6[3], sk->dst_ip6[3]);
		}
	}
	if (sk && ((sk->state == BPF_TCP_ESTABLISHED) || (sk->state >= BPF_TCP_FIN_WAIT1 && sk->state <= BPF_TCP_LAST_ACK))) {
		tsk = bpf_tcp_sock(sk);
		sk_dport = bpf_ntohs(sk->dst_port);
		if (tsk) {
			if (BPF_TCP_ESTABLISHED == sk->state) {
				if (sk->family == 2) {
					srcLTDest = src_lt_dest(sk->src_ip4, sk->dst_ip4, sk->src_port, sk_dport);
					k = ct_make_key(srcLTDest, IPPROTO_TCP, sk->src_ip4, sk->dst_ip4, sk->src_port, sk_dport);
				} else {
					srcLTDest = src_lt_dest(sk->src_ip6[3], sk->dst_ip6[3], sk->src_port, sk_dport);
					k = ct_make_key(srcLTDest, IPPROTO_TCP, sk->src_ip6[3], sk->dst_ip6[3], sk->src_port, sk_dport);
				}
				v = cali_v4_ct_lookup_elem(&k);
				if (!v) {
					if (sk->family == 10) {
					CALI_DEBUG("Socketv6 ct lookup miss\n");} else {
						CALI_DEBUG("Socketv4 ct lookup miss 0x%x 0x%x\n", sk->src_ip4, sk->dst_ip4);
					}
					goto release;
				} else {
					if (bpf_ktime_get_ns() - v->last_seen <= SEND_TCP_STATS_INTERVAL) {
						goto release;
					}
				}
			}
			struct event_tcp_stats event = {
				.sport = sk->src_port,
				.dport = sk_dport,
				.hdr.len = sizeof(struct event_tcp_stats),
				.hdr.type = EVENT_TCP_STATS,
				.snd_cwnd = tsk->snd_cwnd,
				.srtt_us = tsk->srtt_us,
				.rtt_min = tsk->rtt_min,
				.snd_ssthresh = tsk->snd_ssthresh,
				.total_retrans = tsk->total_retrans,
				.lost_out = tsk->lost_out,
				.icsk_retransmits = tsk->icsk_retransmits,
				.mss_cache = tsk->mss_cache,
				.ecn_flags = tsk->ecn_flags,
			};
			if (sk->family == 2) {
				event.saddr[10] = event.saddr[11] = event.daddr[10] = event.daddr[11] = 0xff;
				__builtin_memcpy(&event.saddr[12], &sk->src_ip4, 4);
				__builtin_memcpy(&event.daddr[12], &sk->dst_ip4, 4);
			} else {
				__builtin_memcpy(event.saddr, sk->src_ip6, 16);
				__builtin_memcpy(event.daddr, sk->dst_ip6, 16);
				CALI_DEBUG("Sridhar ipv6 0x%x 0x%x 0x%x\n",event.saddr[13], event.saddr[14], event.saddr[15]);
				CALI_DEBUG("Sridhar ipv6 0x%x 0x%x 0x%x\n",event.daddr[13], event.daddr[14], event.daddr[15]);
				CALI_DEBUG("Sridhar ipv6 0x%x 0x%x\n",sk->src_ip4, sk->dst_ip4);
			}
			event_tcp_stats(ctx->skb, &event);
			//event_tcp_stats(ctx->skb, v6_saddr, v6_daddr, sk->src_port, sk_dport, tsk);
		}
	}
release:
	if (sk) {
		bpf_sk_release(sk);
	}
	return;
}

#endif /* __CALI_LOOKUP_H__ */
