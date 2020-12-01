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

#ifndef __CALI_KPROBE_H__
#define __CALI_KPROBE_H__

#include <stddef.h>
#include <stdbool.h>
#include <linux/in.h>
#include <linux/ip.h>
#include <linux/if_ether.h>
#include <linux/udp.h>
#include <sys/socket.h>

#include "bpf.h"
#include "tracing.h"

#define SEND_DATA_INTERVAL 2000000000
struct __attribute__((__packed__)) calico_kprobe_proto_v4_key {
	__u32 pid;
	uint32_t saddr;
	uint16_t sport;
	uint32_t daddr;
	uint16_t dport;
};

struct calico_kprobe_proto_v4_value {
	uint32_t txBytes;
	uint32_t rxBytes;
	__u64	timestamp;
};

CALI_MAP_V1(cali_v4_tcpkp,
		BPF_MAP_TYPE_LRU_HASH,
		struct calico_kprobe_proto_v4_key, struct calico_kprobe_proto_v4_value,
		511000, 0, MAP_PIN_GLOBAL)

CALI_MAP_V1(cali_v4_udpkp,
		BPF_MAP_TYPE_LRU_HASH,
		struct calico_kprobe_proto_v4_key, struct calico_kprobe_proto_v4_value,
		511000, 0, MAP_PIN_GLOBAL)

#endif /* __CALI_KPROBE_H__ */

