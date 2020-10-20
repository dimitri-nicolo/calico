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

#include <asm/types.h>
#include <linux/bpf.h>
#include <linux/pkt_cls.h>
#include <linux/ip.h>
#include <linux/socket.h>
#include <linux/icmp.h>
#include <linux/in.h>
#include <linux/udp.h>
#include <linux/if_ether.h>
#include <iproute2/bpf_elf.h>
#include <stdbool.h>
#include <stdint.h>
#include <stddef.h>
#include "bpf.h"
#include "log.h"
#include <linux/bpf_perf_event.h>

struct __attribute__((__packed__)) calico_test_kp_key {
        __u32 pid;
};

struct calico_test_kp_value {
        __u64   timestamp;
};

CALI_MAP_V1(cali_test_kp,
                BPF_MAP_TYPE_HASH,
                struct calico_test_kp_key, struct calico_test_kp_value,
                511000, BPF_F_NO_PREALLOC, MAP_PIN_GLOBAL)

__attribute__((section("kprobe/tcp_sendmsg")))
int kprobe__tcp_sendmsg(struct pt_regs *ctx)
{
	__u32 pid = 0;
	__u64 ts = 0;
	int ret = 0;
	struct calico_test_kp_key key = {};
	struct calico_test_kp_value value = {};
	struct calico_test_kp_value *elem = NULL;
	pid = bpf_get_current_pid_tgid();
	ts = bpf_ktime_get_ns();
	key.pid = pid;

	elem = cali_test_kp_lookup_elem(&key);
	if (NULL == elem) {
		value.timestamp = ts;
		ret = cali_test_kp_update_elem(&key, &value, 0);
		if (ret < 0) {
			return -1;
		}
	} else {
		ret = cali_test_kp_delete_elem(&key);
		if (ret < 0) {
			return -1;
		}
	}

	return 0;
}

char ____license[] __attribute__((section("license"), used)) = "GPL";

