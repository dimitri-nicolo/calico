// Project Calico BPF dataplane programs.
// Copyright (c) 2022 Tigera, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0 OR GPL-2.0-or-later

#ifndef __CALI_EGW_H__
#define __CALI_EGW_H__

#include "bpf.h"


static CALI_BPF_INLINE bool is_egw_health_packet(__be32 ip, __be16 port)
{
	union ip4_set_lpm_key sip = {
		.ip.mask = 32 /* IP prefix length */ + 64 /* Match ID */ + 16 /* Match port */ + 8 /* Match protocol */,
		.ip.set_id = bpf_cpu_to_be64(EGRESS_GW_HEALTH_ID),
		.ip.addr = ip,
		.ip.port = port,
		.ip.protocol = 6
	};
	if (bpf_map_lookup_elem(&cali_v4_ip_sets, &sip)) {
		CALI_DEBUG("IP 0x%x port 0x%x are for EGW\n",ip, port);
		return true;
	}
	return false;
}
#endif

