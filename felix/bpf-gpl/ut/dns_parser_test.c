// Project Calico BPF dataplane programs.
// Copyright (c) 2023 Tigera, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0 OR GPL-2.0-or-later

#include "ut.h"
#include "parsing.h"
#include "dns_reply.h"

const volatile struct cali_tc_preamble_globals __globals;

static CALI_BPF_INLINE int calico_unittest_entry(struct __sk_buff *skb)
{
	volatile struct cali_tc_globals *globals = state_get_globals_tc();

	if (!globals) {
		return TC_ACT_SHOT;
	}

	/* Set the globals for the rest of the prog chain. */
	globals->data = __globals.v4;
	DECLARE_TC_CTX(_ctx,
		.skb = skb,
		.ipheader_len = IP_SIZE,
	);
	struct cali_tc_ctx *ctx = &_ctx;

	if (!ctx->counters) {
		CALI_DEBUG("Counters map lookup failed: DROP\n");
		return TC_ACT_SHOT;
	}


	if (parse_packet_ip(ctx) != PARSING_OK) {
		goto deny;
	}

	tc_state_fill_from_iphdr(ctx);

	/* --- above just for test, filled by the caller --- */

	dns_process_datagram(ctx);

	return TC_ACT_UNSPEC;

deny:
	return TC_ACT_SHOT;
}
