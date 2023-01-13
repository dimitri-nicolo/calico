// Project Calico BPF dataplane programs.
// Copyright (c) 2021-2023 Tigera, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0 OR GPL-2.0-or-later

#ifndef __CALI_LOOKUP_H__
#define __CALI_LOOKUP_H__

static CALI_BPF_INLINE void socket_lookup(struct cali_tc_ctx *ctx) {
	struct bpf_sock *sk = NULL;
	struct bpf_sock_tuple tuple={};
	struct bpf_tcp_sock *tsk = NULL;

	if (CALI_F_FROM_WEP) {
		tuple.ipv4.saddr = ip_hdr(ctx)->daddr;
		tuple.ipv4.daddr = ip_hdr(ctx)->saddr;
		tuple.ipv4.sport = tcp_hdr(ctx)->dest;
		tuple.ipv4.dport = tcp_hdr(ctx)->source;
	} else if (CALI_F_TO_WEP) {
		tuple.ipv4.saddr = ip_hdr(ctx)->saddr;
		tuple.ipv4.daddr = ip_hdr(ctx)->daddr;
		tuple.ipv4.sport = tcp_hdr(ctx)->source;
		tuple.ipv4.dport = tcp_hdr(ctx)->dest;
	}
	sk = bpf_sk_lookup_tcp(ctx->skb, &tuple, sizeof(tuple.ipv4), IF_NS, 0);
	if (!sk) {
		tuple.ipv6.saddr[0] = tuple.ipv6.saddr[1] = tuple.ipv6.daddr[0] = tuple.ipv6.daddr[1] = 0;
		tuple.ipv6.saddr[2] = tuple.ipv6.daddr[2] = 0x0000ffff;
		if (CALI_F_FROM_WEP) {
			tuple.ipv6.saddr[3] = ip_hdr(ctx)->daddr;
			tuple.ipv6.daddr[3] = ip_hdr(ctx)->saddr;
			tuple.ipv6.sport = tcp_hdr(ctx)->dest;
			tuple.ipv6.dport = tcp_hdr(ctx)->source;
		} else if (CALI_F_TO_WEP) {
			tuple.ipv6.saddr[3] = ip_hdr(ctx)->saddr;
			tuple.ipv6.daddr[3] = ip_hdr(ctx)->daddr;
			tuple.ipv6.sport = tcp_hdr(ctx)->source;
			tuple.ipv6.dport = tcp_hdr(ctx)->dest;
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
