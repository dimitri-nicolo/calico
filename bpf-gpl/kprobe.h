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

#ifndef __CALI_KPROBE_H__
#define __CALI_KPROBE_H__

#include "bpf.h"
#include "tracing.h"

#define SEND_DATA_INTERVAL 10000000000
struct __attribute__((__packed__)) calico_kprobe_proto_v4_key {
	__u32 pid;
	__u32 saddr;
	__u16 sport;
	__u32 daddr;
	__u16 dport;
};

struct calico_kprobe_proto_v4_value {
	__u32 bytes;
	__u64	timestamp;
};

CALI_MAP_V1(cali_v4_txstats,
		BPF_MAP_TYPE_LRU_HASH,
		struct calico_kprobe_proto_v4_key, struct calico_kprobe_proto_v4_value,
		511000, 0, MAP_PIN_GLOBAL)

CALI_MAP_V1(cali_v4_rxstats,
		BPF_MAP_TYPE_LRU_HASH,
		struct calico_kprobe_proto_v4_key, struct calico_kprobe_proto_v4_value,
		511000, 0, MAP_PIN_GLOBAL)

static int CALI_BPF_INLINE kprobe_collect_stats(struct pt_regs *ctx,
						struct sock_common *sk_cmn,
						__u32 proto,
						int bytes,
						int tx)
{
	__u32 saddr = 0, daddr = 0, pid = 0;
	__u16 family = 0, sport = 0, dport = 0;
	__u64 ts = 0; __u64 diff = 0;
	int ret = 0;
	struct calico_kprobe_proto_v4_value v4_value = {};
	struct calico_kprobe_proto_v4_value *val = NULL;
	struct calico_kprobe_proto_v4_key key = {};

	if (!sk_cmn) {
		goto error;
	}

	bpf_probe_read(&family, 2, &sk_cmn->skc_family);
	if (family != 2 /* AF_INET */) {
		goto error;
	}

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
		event_bpf_v4stats(ctx, pid, saddr, sport, daddr, dport, v4_value.bytes, proto, !tx);
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
			event_bpf_v4stats(ctx, pid, saddr, sport, daddr, dport, val->bytes, proto, !tx);
			val->timestamp = ts;
		}
		val->bytes += bytes;
	}

	return 0;

error:
	return -1;
}

#endif /* __CALI_KPROBE_H__ */
