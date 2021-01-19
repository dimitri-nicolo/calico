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

#ifndef __CALI_BPF_FAILSAFE_H__
#define __CALI_BPF_FAILSAFE_H__

#include "bpf.h"
#include "types.h"

struct failsafe_key {
	__u16 port;
};

struct failsafe_val {
	__u32 unused;
};

CALI_MAP_V1(cali_v4_failsafe_in,
		BPF_MAP_TYPE_HASH,
		struct failsafe_key, struct failsafe_val,
		65536,
		BPF_F_NO_PREALLOC,
		MAP_PIN_GLOBAL)

CALI_MAP_V1(cali_v4_failsafe_out,
		BPF_MAP_TYPE_HASH,
		struct failsafe_key, struct failsafe_val,
		65536,
		BPF_F_NO_PREALLOC,
		MAP_PIN_GLOBAL)

static CALI_BPF_INLINE bool is_failsafe_in(__u16 dport) {
	struct failsafe_key key = {
		.port = dport,
	};
	if (cali_v4_failsafe_in_lookup_elem(&key)) {
		return true;
	}
	return false;
}

static CALI_BPF_INLINE bool is_failsafe_out(__u16 dport) {
	struct failsafe_key key = {
		.port = dport,
	};
	if (cali_v4_failsafe_out_lookup_elem(&key)) {
		return true;
	}
	return false;
}

#endif /* __CALI_BPF_FAILSAFE_H__ */
