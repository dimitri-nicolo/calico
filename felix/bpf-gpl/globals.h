// Project Calico BPF dataplane programs.
// Copyright (c) 2020-2022 Tigera, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0 OR GPL-2.0-or-later

#ifndef __CALI_GLOBALS_H__
#define __CALI_GLOBALS_H__

struct cali_tc_globals {
	__be32 host_ip;
	__be16 tunnel_mtu;
	__be16 vxlan_port;
	__be32 intf_ip;
	__be32 ext_to_svc_mark;
	__be16 psnat_start;
	__be16 psnat_len;
	__be16 if_ns;
	__be32 host_tunnel_ip;
	__be32 flags;
	__be16 wg_port;
	__u32 natin_idx;
	__u32 natout_idx;
	__be16 egw_vxlan_port;
	__be16 egw_health_port;
};

enum cali_globals_flags {
	/* CALI_GLOBALS_IPV6_ENABLED is set when IPv6 is enabled by Felix */
	CALI_GLOBALS_IPV6_ENABLED 	= 0x00000001,
	CALI_GLOBALS_TCP_STATS_ENABLED 	= 0x00000002,
	CALI_GLOBALS_IS_EGRESS_GATEWAY 	= 0x00000004,
	CALI_GLOBALS_IS_EGRESS_CLIENT 	= 0x00000008,
	CALI_GLOBALS_RPF_STRICT_ENABLED = 0x00000010,
	CALI_GLOBALS_IS_EGRESS_IP_ENABLED = 0x00000020,
};

struct cali_ctlb_globals {
	__be32 udp_not_seen_timeo;
};

#endif /* __CALI_GLOBALS_H__ */
