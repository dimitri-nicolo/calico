// Project Calico BPF dataplane programs.
// Copyright (c) 2021 Tigera, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0 OR GPL-2.0-or-later

#ifndef __CALI_SSTATS_H__
#define __CALI_SSTATS_H__

#include "bpf.h"

struct __attribute__((__packed__)) calico_socket_stats_key {
        __u8  saddr[16];
        __u8  daddr[16];
        __u16 sport;
        __u16 dport;
};

struct calico_socket_stats_value {
        __u64   timestamp;
};

CALI_MAP(cali_sstats, 2,
                BPF_MAP_TYPE_LRU_HASH,
                struct calico_socket_stats_key, struct calico_socket_stats_value,
                511000, 0)

#endif
