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
struct __attribute__((__packed__)) calico_kprobe_stats_key {
	__u8  saddr[16];
	__u8  daddr[16];
	__u16 sport;
	__u16 dport;
	__u32 pid;
	__u16 proto;
	__u16  dir;
};

struct calico_kprobe_stats_value {
	__u32 bytes;
	__u64	timestamp;
};

CALI_MAP(cali_kpstats, 2,
		BPF_MAP_TYPE_LRU_HASH,
		struct calico_kprobe_stats_key, struct calico_kprobe_stats_value,
		511000, 0, MAP_PIN_GLOBAL)

static int CALI_BPF_INLINE ip_addr_is_zero(__u8 *addr) {
	__u64 *a64 = (__u64*)addr;

	return (a64[0] == 0 && a64[1] == 0);
}

static int CALI_BPF_INLINE kprobe_collect_stats(struct pt_regs *ctx,
						struct sock_common *sk_cmn,
						__u16 proto,
						int bytes,
						__u16 tx)
{
	__u16 family = 0;
	__u64 ts = 0, diff = 0;
	int ret = 0;
	struct calico_kprobe_stats_value value = {};
	struct calico_kprobe_stats_value *val = NULL;
	struct calico_kprobe_stats_key key = {};

	if (!sk_cmn) {
		return 0;
	}

	bpf_probe_read(&family, 2, &sk_cmn->skc_family);
	if (family == 2 /* AF_INET */) {
		bpf_probe_read(&key.saddr[12], 4, &sk_cmn->skc_rcv_saddr);
		bpf_probe_read(&key.daddr[12], 4, &sk_cmn->skc_daddr);
	} else if (family == 10 /* AF_INET6 */) {
		bpf_probe_read(key.saddr, 16, sk_cmn->skc_v6_rcv_saddr.in6_u.u6_addr8);
		bpf_probe_read(key.daddr, 16, sk_cmn->skc_v6_daddr.in6_u.u6_addr8);
	} else {
		CALI_DEBUG("unknown IP family.Ignoring\n");
		return 0;
	}

	bpf_probe_read(&key.sport, 2, &sk_cmn->skc_num);
	bpf_probe_read(&key.dport, 2, &sk_cmn->skc_dport);

	/* Do not send data when any of src ip,src port, dst ip, dst port is 0.
	 * This being the socket data, value of 0 indicates a socket in listening
	 * state. Further data cannot be correlated in felix.
	 */
	if (!key.sport || !key.dport || ip_addr_is_zero(key.saddr) || ip_addr_is_zero(key.daddr)) {
		return 0;
	}

	key.pid = bpf_get_current_pid_tgid() >> 32;
	ts = bpf_ktime_get_ns();
	if (family == 2) {
		// v4Inv6Prefix {0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xff, 0xff}
		key.saddr[10] = key.saddr[11] = key.daddr[10] = key.daddr[11] = 0xff;
	}

	key.proto = proto;
	key.dir = !tx;

	val = cali_kpstats_lookup_elem(&key);
	if (val == NULL) {
		value.bytes = bytes;
		ret = event_bpf_stats(ctx, key.pid, key.saddr, key.sport, key.daddr,
					key.dport, value.bytes, proto, !tx);
		if (ret == 0) {
			/* Set the timestamp only if we managed to send the event.
			 * Otherwise zero timestamp makes the next call to try to send the
			 * event again.
			 */
			value.timestamp = ts;
		}
		ret = cali_kpstats_update_elem(&key, &value, 0);
	} else {
		diff = ts - val->timestamp;
		if (diff >= SEND_DATA_INTERVAL) {
			ret = event_bpf_stats(ctx, key.pid, key.saddr, key.sport,
						key.daddr, key.dport, value.bytes, proto, !tx);
			if (ret == 0) {
				/* Update the timestamp only if we managed to send the
				 * event. Otherwise keep the old timestamp so that next
				 * call will try to send the event again.
				 */
				val->timestamp = ts;
			}
		}
		val->bytes += bytes;
	}
	return 0;
}

static int CALI_BPF_INLINE kprobe_stats_body(struct pt_regs *ctx, __u16 proto, __u16 tx, bool is_connect)
{
	int bytes = 0;
	struct sock_common *sk_cmn = NULL;

	sk_cmn = (struct sock_common*)PT_REGS_PARM1(ctx);
	/* In case tcp_cleanup_rbuf, second argument is the number of bytes copied
	 * to user space
	 */
	if (is_connect) {
		bytes = 0;
	} else if (proto == IPPROTO_TCP && !tx) {
		bytes = (int)PT_REGS_PARM2(ctx);
	} else {
		bytes = (int)PT_REGS_PARM3(ctx);
	}
	return kprobe_collect_stats(ctx, sk_cmn, proto, bytes, tx);
}


#endif /* __CALI_KPROBE_H__ */
