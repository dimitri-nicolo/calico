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
		return kprobe_collect_stats(ctx, sk_cmn, IPPROTO_UDP, bytes, 0);
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
		return kprobe_collect_stats(ctx, sk_cmn, IPPROTO_UDP, bytes, 1);
	}
	return -1;
}

char ____license[] __attribute__((section("license"), used)) = "GPL";

