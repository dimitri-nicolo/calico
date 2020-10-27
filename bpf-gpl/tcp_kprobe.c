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
#include "sock.h"
#include "kprobe.h"
#include "log.h"
#include <linux/bpf_perf_event.h>

__attribute__((section("kprobe/tcp_sendmsg")))
int kprobe__tcp_sendmsg(struct pt_regs *ctx, struct sock_common *sk_cmn)
{
	__u32 saddr = 0, daddr = 0, pid = 0;
	__u16 family = 0, sport = 0, dport = 0;
	size_t size = 0;
	__u64 ts = 0; __u64 diff = 0;
	int ret = 0;
	struct calico_tcp_kprobe_v4_value v4_value = {};
	struct calico_tcp_kprobe_v4_value *val = NULL;

	struct calico_tcp_kprobe_v4_key key = {};
	if (ctx) {
		size = (size_t)PT_REGS_PARM3(ctx);
		struct sock_common *sk_cmn = (struct sock_common*)PT_REGS_PARM1(ctx);
		if (sk_cmn) {
			bpf_probe_read (&family, 2, &sk_cmn->skc_family);
                        bpf_probe_read (&sport, 2, &sk_cmn->skc_num);
                        bpf_probe_read (&dport, 2, &sk_cmn->skc_dport);
                        bpf_probe_read (&saddr, 4, &sk_cmn->skc_rcv_saddr);
                        bpf_probe_read (&daddr, 4, &sk_cmn->skc_daddr);
			pid = bpf_get_current_pid_tgid() >> 32;
			ts = bpf_ktime_get_ns();
			if (family == 2) {
				key.pid = pid;
                                key.saddr = saddr;
                                key.sport = sport;
                                key.daddr = daddr;
                                key.dport = dport;
				val = cali_v4_tcp_kp_lookup_elem(&key);
				if (val == NULL)
				{
                                	v4_value.timestamp = ts;
                                	v4_value.txBytes = size;
                                	ret = cali_v4_tcp_kp_update_elem(&key, &v4_value, 0);
					if (ret < 0) {
						goto error;
					}
				}
				else {
					diff = ts - val->timestamp;
					if (diff >= 2000000000)
					{
						//event_tcp_flow(ctx, saddr, sport, daddr, dport, val->txBytes);	
					}
                                        val->timestamp = ts;
                                        val->txBytes += size;
                                        ret = cali_v4_tcp_kp_update_elem(&key, val, BPF_F_LOCK);
					if (ret < 0) {
						goto error;
					}
                                }
				return 0;	
			}

		}
	}
error:
	return -1;
}

char ____license[] __attribute__((section("license"), used)) = "GPL";

