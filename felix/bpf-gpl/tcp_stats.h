// Project Calico BPF dataplane programs.
// Copyright (c) 2022 Tigera, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0 OR GPL-2.0-or-later

#ifndef __CALI_TCPSTATS_H__
#define __CALI_TCPSTATS_H__

#include "sstats.h"

#define SEND_TCP_STATS_INTERVAL 5000000000

static CALI_BPF_INLINE void send_tcp_stats(struct bpf_sock *sk, struct bpf_tcp_sock *tsk, struct cali_tc_ctx *ctx) {
	struct calico_socket_stats_key key = {};
	struct calico_socket_stats_value *val = NULL;
	struct calico_socket_stats_value value = {};
	__u64 ts = 0;
	int ret = 0;

	if (tsk) {
		if (BPF_TCP_ESTABLISHED == sk->state) {
			ts = bpf_ktime_get_ns();
			if (sk->family == 2) {
                               	key.saddr[10] = key.saddr[11] = key.daddr[10] = key.daddr[11] = 0xff;
                               	__builtin_memcpy(&key.saddr[12], &sk->src_ip4, 4);
                               	__builtin_memcpy(&key.daddr[12], &sk->dst_ip4, 4);
                       	} else {
                               	__builtin_memcpy(&key.saddr, sk->src_ip6, 16);
                               	__builtin_memcpy(&key.daddr, sk->dst_ip6, 16);
                       	}
			key.sport = sk->src_port;
			key.dport = bpf_ntohs(sk->dst_port);
			val = cali_sstats_lookup_elem(&key);
			if (val == NULL) {
				value.timestamp = ts;
				ret = cali_sstats_update_elem(&key, &value, 0);
			} else {
				if (ts - val->timestamp <= SEND_TCP_STATS_INTERVAL) {
					return;
				}
				val->timestamp = ts;
			}
		}
		struct event_tcp_stats event = {
			.sport = key.sport,
			.dport = key.dport,
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
		__builtin_memcpy(event.saddr, &key.saddr, 16);
		__builtin_memcpy(event.daddr, &key.daddr, 16);
		CALI_DEBUG("TCP stats: event sent for SIP: 0x%x DIP: 0x%x", event.saddr, event.daddr);
		event_tcp_stats(ctx->skb, &event);
	}
}

#endif /* __CALI_LOOKUP_IPTABLES_H__ */
