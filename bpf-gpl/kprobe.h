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
	__u32 pid;
	__u8  saddr[16];
	__u16 sport;
	__u8  daddr[16];
	__u16 dport;
};

struct calico_kprobe_stats_value {
	__u32 bytes;
	__u64	timestamp;
};

CALI_MAP(cali_txstats, 2,
		BPF_MAP_TYPE_LRU_HASH,
		struct calico_kprobe_stats_key, struct calico_kprobe_stats_value,
		511000, 0, MAP_PIN_GLOBAL)

CALI_MAP(cali_rxstats, 2,
		BPF_MAP_TYPE_LRU_HASH,
		struct calico_kprobe_stats_key, struct calico_kprobe_stats_value,
		511000, 0, MAP_PIN_GLOBAL)

static int CALI_BPF_INLINE IpAddrIsZero(__u8 *addr, __u16 family) {
	__u32 zeroV4Addr = 0;
	int result = 0;

	if (family == 2) {
		result = __builtin_memcmp(addr, &zeroV4Addr, sizeof(zeroV4Addr));
	} else {
		/* Check if IPv6 address is 0 */
		if ((__builtin_memcmp(addr, &zeroV4Addr, sizeof(zeroV4Addr)) == 0) &&
			(__builtin_memcmp(&addr[4], &zeroV4Addr, sizeof(zeroV4Addr)) == 0) &&
			(__builtin_memcmp(&addr[8], &zeroV4Addr, sizeof(zeroV4Addr)) == 0) &&
			(__builtin_memcmp(&addr[12], &zeroV4Addr, sizeof(zeroV4Addr)) == 0)) {
			result = 0;
		}
	}
	if (result == 0) {
		return 1;
	}
	return 0;
}

static int CALI_BPF_INLINE kprobe_collect_stats(struct pt_regs *ctx,
						struct sock_common *sk_cmn,
						__u32 proto,
						int bytes,
						int tx)
{
	__u8 saddr[16] = {0}, daddr[16] = {0};
	__u32 pid = 0;
	__u16 family = 0, sport = 0, dport = 0;
	__u64 ts = 0; __u64 diff = 0;
	int ret = 0;
	struct calico_kprobe_stats_value v4_value = {};
	struct calico_kprobe_stats_value *val = NULL;
	struct calico_kprobe_stats_key key = {};

	if (!sk_cmn) {
		return 0;
	}

	bpf_probe_read(&family, 2, &sk_cmn->skc_family);
	if (family != 2 /* AF_INET */) {
		return 0;
	}

	bpf_probe_read(&sport, 2, &sk_cmn->skc_num);
	bpf_probe_read(&dport, 2, &sk_cmn->skc_dport);
	bpf_probe_read(saddr, 4, &sk_cmn->skc_rcv_saddr);
	bpf_probe_read(daddr, 4, &sk_cmn->skc_daddr);

	/* Do not send data when any of src ip,src port, dst ip, dst port is 0.
	 * This being the socket data, value of 0 indicates a socket in listening
	 * state. Further data cannot be correlated in felix.
	 */

	if (!sport || !dport || IpAddrIsZero(saddr, family) || IpAddrIsZero(daddr, family)) {
		return 0;
	}

	pid = bpf_get_current_pid_tgid() >> 32;
	ts = bpf_ktime_get_ns();
	key.pid = pid;
	// v4Inv6Prefix {0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xff, 0xff}
	key.saddr[10] = key.saddr[11] = key.daddr[10] = key.daddr[11] = 0xff;
	__builtin_memcpy(&key.saddr[12], saddr, 4);
	__builtin_memcpy(&key.daddr[12], daddr, 4);

	key.sport = sport;
	key.dport = dport;

	if (tx) {
		val = cali_txstats_lookup_elem(&key);
	} else {
		val = cali_rxstats_lookup_elem(&key);
	}

	if (val == NULL) {
		v4_value.bytes = bytes;
		ret = event_bpf_v4stats(ctx, pid, key.saddr, sport, key.daddr, dport, v4_value.bytes, proto, !tx);
		if (ret == 0) {
			/* Set the timestamp only if we managed to send the event.
			 * Otherwise zero timestamp makes the next call to try to send the
			 * event again.
			 */
			v4_value.timestamp = ts;
		}
		if (tx) {
			ret = cali_txstats_update_elem(&key, &v4_value, 0);
		} else {
			ret = cali_rxstats_update_elem(&key, &v4_value, 0);
		}
	} else {
		diff = ts - val->timestamp;
		if (diff >= SEND_DATA_INTERVAL) {
			ret = event_bpf_v4stats(ctx, pid, key.saddr, sport, key.daddr, dport, val->bytes, proto, !tx);
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

static int CALI_BPF_INLINE kprobe_stats_body(struct pt_regs *ctx, __u32 proto, int tx)
{
	int bytes = 0;
	struct sock_common *sk_cmn = NULL;

	sk_cmn = (struct sock_common*)PT_REGS_PARM1(ctx);
	/* In case tcp_cleanup_rbuf, second argument is the number of bytes copied
	 * to user space
	 */
	if (proto == IPPROTO_TCP && !tx) {
		bytes = (int)PT_REGS_PARM2(ctx);
	} else {
		bytes = (int)PT_REGS_PARM3(ctx);
	}
	return kprobe_collect_stats(ctx, sk_cmn, proto, bytes, tx);
}


#endif /* __CALI_KPROBE_H__ */
