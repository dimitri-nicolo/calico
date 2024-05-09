// Project Calico BPF dataplane programs.
// Copyright (c) 2020-2023 Tigera, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0 OR GPL-2.0-or-later

#include <linux/types.h>
#include <linux/bpf.h>

// stdbool.h has no deps so it's OK to include; stdint.h pulls in parts
// of the std lib that aren't compatible with BPF.
#include <stdbool.h>


#include "bpf.h"
#include "dns_reply.h"

CALI_MAP(cali_dns_data, 1,
		BPF_MAP_TYPE_PERCPU_ARRAY,
		__u32, char[512],
		1, 0)

SEC("tc")
int calico_dns_parser(struct __sk_buff *skb)
{
	struct cali_tc_state *state = state_get();
	if (!state) {
		CALI_LOG_IF(CALI_LOG_LEVEL_DEBUG, "State map lookup failed: DROP\n");
		bpf_exit(TC_ACT_SHOT);
	}

	struct cali_tc_globals *gl = state_get_globals_tc();
	if (!gl) {
		CALI_LOG_IF(CALI_LOG_LEVEL_DEBUG, "no globals: DROP\n");
		bpf_exit(TC_ACT_SHOT);
	}

	/* We do not need to initialize more, we only need the packet, global
	 * state for logging and already parsed IHL. we will have our own scratch.
	 */
	struct cali_tc_ctx ctx ={
		.skb = skb,
		.state = state,
		.globals = gl,
		.ipheader_len = state->ihl,
	}

	dns_process_datagram(ctx);

	/* XXX jump to forward XXX */

	return TC_ACT_SHOT;
}
