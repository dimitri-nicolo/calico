// Project Calico BPF dataplane programs.
// Copyright (c) 2020-2021 Tigera, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0 OR GPL-2.0-or-later

#include "bpf.h"
#include "log.h"
#include <linux/bpf_perf_event.h>

__attribute__((section("kprobe/tcp_sendmsg")))
int kprobe__tcp_sendmsg(struct pt_regs *ctx)
{
	CALI_DEBUG("Test loader: tcp_sendmsg");
	return 0;
}
