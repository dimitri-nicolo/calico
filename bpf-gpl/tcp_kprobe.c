// Project Calico BPF dataplane programs.
// Copyright (c) 2020 Tigera, Inc. All rights reserved.
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

#include "bpf.h"
#include "log.h"
#include "sock.h"
#include <linux/if_ether.h>
#include "events.h"
#include "kprobe.h"
#include <linux/bpf_perf_event.h>

static int CALI_BPF_INLINE tcp_collect_stats(struct pt_regs *ctx, struct sock_common *sk_cmn, int bytes, int tx) {
	__u32 saddr = 0, daddr = 0, pid = 0;
	__u16 family = 0, sport = 0, dport = 0;
	__u64 ts = 0; __u64 diff = 0;
	int ret = 0;
	struct calico_kprobe_proto_v4_value v4_value = {};
	struct calico_kprobe_proto_v4_value *val = NULL;
	struct calico_kprobe_proto_v4_key key = {};

	if (sk_cmn) {
		bpf_probe_read(&family, 2, &sk_cmn->skc_family);
		bpf_probe_read(&sport, 2, &sk_cmn->skc_num);
		bpf_probe_read(&dport, 2, &sk_cmn->skc_dport);
		bpf_probe_read(&saddr, 4, &sk_cmn->skc_rcv_saddr);
		bpf_probe_read(&daddr, 4, &sk_cmn->skc_daddr);
		pid = bpf_get_current_pid_tgid() >> 32;
		ts = bpf_ktime_get_ns();
		if (family == AF_INET) {
			key.pid = pid;
			key.saddr = saddr;
			key.sport = sport;
			key.daddr = daddr;
			key.dport = dport;
			val = cali_v4_stats_lookup_elem(&key);
			if (val == NULL) {
				v4_value.timestamp = ts;
				if (tx) {
					v4_value.txBytes = bytes;
				} else {
					v4_value.rxBytes = bytes;
				}
				event_bpf_v4stats(ctx, pid, saddr, sport, daddr, dport, v4_value.txBytes, v4_value.rxBytes, IPPROTO_TCP);
				ret = cali_v4_stats_update_elem(&key, &v4_value, 0);
				if (ret < 0) {
					goto error;
				}
			} else {
				diff = ts - val->timestamp;
				if (diff >= SEND_DATA_INTERVAL) {
					event_bpf_v4stats(ctx, pid, saddr, sport, daddr, dport, val->txBytes, val->rxBytes, IPPROTO_TCP);
					val->timestamp = ts;
				}
				if (tx) {
					val->txBytes += bytes;
				} else {
					val->rxBytes += bytes;
				}
				ret = cali_v4_stats_update_elem(&key, val, BPF_F_LOCK);
				if (ret < 0) {
					goto error;
				}
			}
			return 0;
		}
	}
error:
	return -1;
}

__attribute__((section("kprobe/tcp_cleanup_rbuf")))
int kprobe__tcp_cleanup_rbuf(struct pt_regs *ctx)
{
	int bytes = 0;
	struct sock_common *sk_cmn = NULL;
	if (ctx) {
		sk_cmn = (struct sock_common*)PT_REGS_PARM1(ctx);
		bytes = (int)PT_REGS_PARM2(ctx);
		if (bytes < 0) {
			return 0;
		}
		return tcp_collect_stats(ctx, sk_cmn, bytes, 0);
	}
	return -1;
}

__attribute__((section("kprobe/tcp_sendmsg")))
int kprobe__tcp_sendmsg(struct pt_regs *ctx)
{
	int bytes = 0;
	struct sock_common *sk_cmn = NULL;
	if (ctx) {
		sk_cmn = (struct sock_common*)PT_REGS_PARM1(ctx);
		bytes = (int)PT_REGS_PARM3(ctx);
		return tcp_collect_stats(ctx, sk_cmn, bytes, 1);
	}
	return -1;
}

char ____license[] __attribute__((section("license"), used)) = "GPL";

