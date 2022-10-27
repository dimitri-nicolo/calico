// Project Calico BPF dataplane programs.
// Copyright (c) 2022 Tigera, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0 OR GPL-2.0-or-later

#ifndef __CALI_EGW_H__
#define __CALI_EGW_H__

#include "bpf.h"


static CALI_BPF_INLINE bool calico_check_for_egw_health(struct cali_tc_ctx *ctx)
{
	__be32 dst_ip = ctx->state->ip_dst;
	__be16 dst_port = ctx->state->dport;

	union ip4_set_lpm_key sip;
	__builtin_memset(&sip, 0, sizeof(sip));
	sip.ip.mask = 32 /* IP prefix length */ + 64 /* Match ID */ + 16 /* Match port */ + 8 /* Match protocol */;
	sip.ip.set_id = bpf_cpu_to_be64(EGRESS_GW_HEALTH_ID);
	sip.ip.addr = dst_ip;
	sip.ip.port = dst_port;
	sip.ip.protocol = 6;
	if (bpf_map_lookup_elem(&cali_v4_ip_sets, &sip)) 
	{
		CALI_DEBUG("Dst IP/port are for EGW\n");
		return true;
	}
	return false;
}
#endif

