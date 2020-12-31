// Project Calico BPF dataplane programs.
// Copyright (c) 2020 Tigera, Inc. All rights reserved.
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

#ifndef __CALI_BPF_JUMP_H__
#define __CALI_BPF_JUMP_H__

#include "conntrack.h"
#include "policy.h"
#include "perf.h"

#define MAX_RULE_IDS	32

// struct cali_tc_state holds state that is passed between the BPF programs.
// WARNING: must be kept in sync with the definitions in bpf/polprog/pol_prog_builder.go.
struct cali_tc_state {
	struct perf_event_header eventhdr;

	__be32 ip_src;
	__be32 ip_dst;
	__be32 post_nat_ip_dst;
	__be32 tun_ip;
	__s32 pol_rc;
	__u16 sport;
	union
	{
		__u16 dport;
		struct
		{
			__u8 icmp_type;
			__u8 icmp_code;
		};
	};
	__u16 post_nat_dport;
	__u8 ip_proto;
	__u8 flags;

	/* XXX we could name it in a more generic way perhaps, size (in u64) and
	 * data to allow arbitrary comunication between programs
	 */
	__u32 rules_hit;
	__u64 rule_ids[MAX_RULE_IDS];

	/* We must not scatter the above ^^^ to copy it in a single memcpy */

	struct calico_ct_result ct_result;
	__u32 _pad32;
	struct calico_nat_dest nat_dest;
	__u64 prog_start_time;
};

enum cali_state_flags {
	CALI_ST_NAT_OUTGOING	= (1 << 0),
	CALI_ST_SKIP_FIB	= (1 << 1),
};

CALI_MAP_V1(cali_v4_state,
		BPF_MAP_TYPE_PERCPU_ARRAY,
		__u32, struct cali_tc_state,
		1, 0, MAP_PIN_GLOBAL)


struct bpf_map_def_extended __attribute__((section("maps"))) cali_jump = {
	.type = BPF_MAP_TYPE_PROG_ARRAY,
	.key_size = 4,
	.value_size = 4,
	.max_entries = 8,
#ifndef __BPFTOOL_LOADER__
	.map_id = 1,
	.pinning_strategy = 1 /* object namespace */,
#endif
};

static CALI_BPF_INLINE void tc_state_fill_from_iphdr(struct cali_tc_state *state, struct iphdr *ip)
{
	state->ip_src = ip->saddr;
	state->ip_dst = ip->daddr;
	state->ip_proto = ip->protocol;
}

/* Add new values to the end as these are program indices */
enum cali_jump_index {
	PROG_INDEX_POLICY,
	PROG_INDEX_EPILOGUE,
	PROG_INDEX_ICMP,
	PROG_INDEX_DROP,
};
#endif /* __CALI_BPF_JUMP_H__ */
