// Project Calico BPF dataplane programs.
// Copyright (c) 2022-2023 Tigera, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0 OR GPL-2.0-or-later

#ifndef _CTLB_H_
#define _CTLB_H_

const volatile struct cali_ctlb_globals __globals;
#define CTLB_UDP_NOT_SEEN_TIMEO __globals.udp_not_seen_timeo
#define CTLB_EXCLUDE_UDP __globals.exclude_udp

#endif /* _CTLB_H_ */
