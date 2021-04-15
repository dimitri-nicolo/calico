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
                511000, 0, MAP_PIN_GLOBAL)

#endif
