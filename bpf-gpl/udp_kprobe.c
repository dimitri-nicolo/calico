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

#include <linux/in.h>

#include "bpf.h"
#include "log.h"
#include "sock.h"
#include "events.h"
#include "kprobe.h"

static int CALI_BPF_INLINE udp_collect_stats(struct pt_regs *ctx, struct sock_common *sk_cmn, int bytes, int tx) {
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
		/* Do not send data when any of src ip,src port, dst ip, dst port is 0.
		 * This being the socket data, value of 0 indicates a socket in listening
		 * state. Further data cannot be correlated in felix.
		 */
		if (!sport || !dport || !saddr || !daddr) {
			return 0;
		}
		pid = bpf_get_current_pid_tgid() >> 32;
		ts = bpf_ktime_get_ns();
		if (family == 2 /* AF_INET */) {
			key.pid = pid;
			key.saddr = saddr;
			key.sport = sport;
			key.daddr = daddr;
			key.dport = dport;
			if (tx) {
				val = cali_v4_txstats_lookup_elem(&key);
			} else {
				val = cali_v4_rxstats_lookup_elem(&key);
			}
			if (val == NULL) {
				v4_value.timestamp = ts;
				v4_value.bytes = bytes;
				event_bpf_v4stats(ctx, pid, saddr, sport, daddr, dport, v4_value.bytes, IPPROTO_UDP, !tx);
				if (tx) {
					ret = cali_v4_txstats_update_elem(&key, &v4_value, 0);
				} else {
					ret = cali_v4_rxstats_update_elem(&key, &v4_value, 0);
				}

				if (ret < 0) {
					goto error;
				}
			} else {
				diff = ts - val->timestamp;
				if (diff >= SEND_DATA_INTERVAL) {
					event_bpf_v4stats(ctx, pid, saddr, sport, daddr, dport, val->bytes, IPPROTO_UDP, !tx);
					val->timestamp = ts;
				}
				val->bytes += bytes;
			}
			return 0;
		}
	}
error:
	return -1;
}

/* The kernel functions udp_sendmsg and udp_recvmsg are serialized.
 * Hence we should not be running into any race condition.
 */
__attribute__((section("kprobe/udp_recvmsg")))
int kprobe__udp_recvmsg(struct pt_regs *ctx)
{
	int bytes = 0;
	struct sock_common *sk_cmn = NULL;
	if (ctx) {
		sk_cmn = (struct sock_common*)PT_REGS_PARM1(ctx);
		bytes = (int)PT_REGS_PARM3(ctx);
		if (bytes < 0) {
			return 0;
		}
		return udp_collect_stats(ctx, sk_cmn, bytes, 0);
	}
	return -1;
}

__attribute__((section("kprobe/udp_sendmsg")))
int kprobe__udp_sendmsg(struct pt_regs *ctx)
{
	int bytes = 0;
	struct sock_common *sk_cmn = NULL;
	if (ctx) {
		sk_cmn = (struct sock_common*)PT_REGS_PARM1(ctx);
		bytes = (int)PT_REGS_PARM3(ctx);
		return udp_collect_stats(ctx, sk_cmn, bytes, 1);
	}
	return -1;
}

char ____license[] __attribute__((section("license"), used)) = "GPL";

