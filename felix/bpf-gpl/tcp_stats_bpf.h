// Project Calico BPF dataplane programs.
// Copyright (c) 2021 Tigera, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0 OR GPL-2.0-or-later

#ifndef __CALI_TCPSTATS_BPF_H__
#define __CALI_TCPSTATS_BPF_H__

#define SEND_TCP_STATS_INTERVAL 500000

static CALI_BPF_INLINE void send_tcp_stats(struct bpf_sock *sk, struct bpf_tcp_sock *tsk, struct cali_tc_ctx *ctx) {
	if (tsk) {
		if (BPF_TCP_ESTABLISHED == sk->state) {
			if (bpf_ktime_get_ns() - ctx->state->ct_result.prev_ts <= SEND_TCP_STATS_INTERVAL) {
				return;
			}
		}
		struct event_tcp_stats event = {
			.sport = sk->src_port,
			.dport = bpf_ntohs(sk->dst_port),
			.hdr.len = sizeof(struct event_tcp_stats),
			.hdr.type = EVENT_TCP_STATS,
			.snd_cwnd = tsk->snd_cwnd,
			.srtt_us = tsk->srtt_us,
			.rtt_min = tsk->rtt_min,
			.total_retrans = tsk->total_retrans,
			.lost_out = tsk->lost_out,
			.icsk_retransmits = tsk->icsk_retransmits,
			.mss_cache = tsk->mss_cache,
		};
		if (sk->family == 2) {
			event.saddr[10] = event.saddr[11] = event.daddr[10] = event.daddr[11] = 0xff;
			__builtin_memcpy(&event.saddr[12], &sk->src_ip4, 4);
			__builtin_memcpy(&event.daddr[12], &sk->dst_ip4, 4);
		} else {
			__builtin_memcpy(event.saddr, sk->src_ip6, 16);
			__builtin_memcpy(event.daddr, sk->dst_ip6, 16);
		}
		CALI_DEBUG("TCP stats: event sent for SIP: 0x%x DIP: 0x%x", event.saddr, event.daddr);
		event_tcp_stats(ctx->skb, &event);
	}
}

#endif /* __CALI_LOOKUP_BPF_H__ */
