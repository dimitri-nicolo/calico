// Project Calico BPF dataplane programs.
// Copyright (c) 2020-2021 Tigera, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0 OR GPL-2.0-or-later

#include "bpf.h"
#include <linux/bpf_perf_event.h>

struct __attribute__((__packed__)) calico_test_map_key {
        __u32 pid;
};

struct calico_test_map1_value {
        __u64   timestamp;
};

struct calico_test_map2_value {
	__u32 	count;
};

CALI_MAP_V1(cali_test_map1,
                BPF_MAP_TYPE_HASH,
                struct calico_test_map_key, struct calico_test_map1_value,
                511000, BPF_F_NO_PREALLOC, MAP_PIN_GLOBAL)

CALI_MAP_V1(cali_test_map2,
                BPF_MAP_TYPE_HASH,
                struct calico_test_map_key, struct calico_test_map2_value,
                511000, BPF_F_NO_PREALLOC, MAP_PIN_GLOBAL)

__attribute__((section("kprobe/tcp_sendmsg")))
int kprobe__tcp_sendmsg(struct pt_regs *ctx)
{
	__u32 pid = 0;
	__u64 ts = 0;
	int ret = 0;
	struct calico_test_map_key key = {};
	struct calico_test_map1_value map1_value = {};
	struct calico_test_map1_value *map1_elem = NULL;
	struct calico_test_map2_value map2_value = {};
	struct calico_test_map2_value *map2_elem = NULL;
	pid = bpf_get_current_pid_tgid();
	ts = bpf_ktime_get_ns();
	key.pid = pid;

	map1_elem = cali_test_map1_lookup_elem(&key);
	if (NULL == map1_elem) {
		map1_value.timestamp = ts;
		ret = cali_test_map1_update_elem(&key, &map1_value, 0);
		if (ret < 0) {
			return -1;
		}
	} else {
		ret = cali_test_map1_delete_elem(&key);
		if (ret < 0) {
			return -1;
		}
	}

	map2_elem = cali_test_map2_lookup_elem(&key);
	if (NULL == map2_elem) {
		map2_value.count = map2_value.count + 1;
		ret = cali_test_map2_update_elem(&key, &map2_value, 0);
		if (ret < 0) {
			return -1;
		}
	} else {
		ret = cali_test_map2_delete_elem(&key);
		if (ret < 0) {
			return -1;
		}
	}
	return 0;
}
