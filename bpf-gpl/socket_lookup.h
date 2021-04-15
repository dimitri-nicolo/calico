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

static CALI_BPF_INLINE void socket_lookup(struct cali_tc_ctx *ctx) {
	struct bpf_sock *sk = NULL;
	struct bpf_sock_tuple tuple={};
	struct bpf_tcp_sock *tsk = NULL;
	
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
	}
	if (sk && ((sk->state == BPF_TCP_ESTABLISHED) || (sk->state >= BPF_TCP_FIN_WAIT1 && sk->state <= BPF_TCP_LAST_ACK))) {
		tsk = bpf_tcp_sock(sk);
		if (tsk) {
			send_tcp_stats(sk, tsk, ctx);
		}
	}
	if (sk) {
		bpf_sk_release(sk);
	}
}


#endif /* __CALI_LOOKUP_H__ */
