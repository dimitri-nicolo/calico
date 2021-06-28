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

/* The kernel functions tcp_sendmsg and tcp_cleanup_rbuf are serialized.
 * Hence we should not be running into any race condition.
 */
__attribute__((section("kprobe/tcp_cleanup_rbuf")))
int kprobe__tcp_cleanup_rbuf(struct pt_regs *ctx)
{
	return kprobe_stats_body(ctx, IPPROTO_TCP, 0, false);
}

__attribute__((section("kprobe/tcp_sendmsg")))
int kprobe__tcp_sendmsg(struct pt_regs *ctx)
{
	return kprobe_stats_body(ctx, IPPROTO_TCP, 1, false);
}

__attribute__((section("kprobe/tcp_connect")))
int kprobe__tcp_connect(struct pt_regs *ctx) {
	return kprobe_stats_body(ctx, IPPROTO_TCP, 1, true);
}

char ____license[] __attribute__((section("license"), used)) = "GPL";

