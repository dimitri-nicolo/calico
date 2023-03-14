// Project Calico BPF dataplane programs.
// Copyright (c) 2021 Tigera, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0 OR GPL-2.0-or-later

#include "bpf.h"
#include "sock.h"
#include "events_kprobe.h"
#include "kprobe.h"

#include <bpf_helpers.h>
#include <bpf_tracing.h>

/* The kernel functions udp_sendmsg and udp_recvmsg are serialized.
 * Hence we should not be running into any race condition.
 */
SEC("kprobe/udp_recvmsg")
int BPF_KPROBE(udp_recvmsg)
{
	return kprobe_stats_body(ctx, IPPROTO_UDP, 0, false);
}

SEC("kprobe/udp_sendmsg")
int BPF_KPROBE(udp_sendmsg)
{
	return kprobe_stats_body(ctx, IPPROTO_UDP, 1, false);
}

SEC("kprobe/udpv6_recvmsg")
int BPF_KPROBE(udpv6_recvmsg)
{
        return kprobe_stats_body(ctx, IPPROTO_UDP, 0, false);
}

SEC("kprobe/udpv6_sendmsg")
int BPF_KPROBE(udpv6_sendmsg)
{
        return kprobe_stats_body(ctx, IPPROTO_UDP, 1, false);
}
