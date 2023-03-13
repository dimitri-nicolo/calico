// Project Calico BPF dataplane programs.
// Copyright (c) 2021 Tigera, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0 OR GPL-2.0-or-later

#include "bpf.h"
#include "sock.h"
#include "events_kprobe.h"
#include "kprobe.h"

#include <bpf_helpers.h>
#include <bpf_tracing.h>

/* The kernel functions tcp_sendmsg and tcp_cleanup_rbuf are serialized.
 * Hence we should not be running into any race condition.
 */
SEC("kprobe/tcp_cleanup_rbuf")
int BPF_KPROBE(tcp_cleanup_rbuf)
{
	return kprobe_stats_body(ctx, IPPROTO_TCP, 0, false);
}

SEC("kprobe/tcp_sendmsg")
int BPF_KPROBE(tcp_sendmsg)
{
	return kprobe_stats_body(ctx, IPPROTO_TCP, 1, false);
}

SEC("kprobe/tcp_connect")
int BPF_KPROBE(tcp_connect) {
	return kprobe_stats_body(ctx, IPPROTO_TCP, 1, true);
}
