// Project Calico BPF dataplane programs.
// Copyright (c) 2020-2021 Tigera, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0 OR GPL-2.0-or-later

#include "bpf.h"
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
