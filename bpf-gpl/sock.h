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

// This struct is based on struct sock_common from the kernel with the below
// license

/* SPDX-License-Identifier: GPL-2.0-or-later */
/*
 * INET		An implementation of the TCP/IP protocol suite for the LINUX
 *		operating system.  INET is implemented using the  BSD Socket
 *		interface as the means of communication with the user level.
 *
 *		Definitions for the AF_INET socket handler.
 *
 * Version:	@(#)sock.h	1.0.4	05/13/93
 *
 * Authors:	Ross Biro
 *		Fred N. van Kempen, <waltje@uWalt.NL.Mugnet.ORG>
 *		Corey Minyard <wf-rch!minyard@relay.EU.net>
 *		Florian La Roche <flla@stud.uni-sb.de>
 *
 * Fixes:
 *		Alan Cox	:	Volatiles in skbuff pointers. See
 *					skbuff comments. May be overdone,
 *					better to prove they can be removed
 *					than the reverse.
 *		Alan Cox	:	Added a zapped field for tcp to note
 *					a socket is reset and must stay shut up
 *		Alan Cox	:	New fields for options
 *	Pauline Middelink	:	identd support
 *		Alan Cox	:	Eliminate low level recv/recvfrom
 *		David S. Miller	:	New socket lookup architecture.
 *              Steve Whitehouse:       Default routines for sock_ops
 *              Arnaldo C. Melo :	removed net_pinfo, tp_pinfo and made
 *              			protinfo be just a void pointer, as the
 *              			protocol specific parts were moved to
 *              			respective headers and ipv4/v6, etc now
 *              			use private slabcaches for its socks
 *              Pedro Hortas	:	New flags field for socket options
 */
#ifndef _SOCK_H
#define _SOCK_H

#include <linux/in6.h>
typedef __u32 __bitwise __portpair;
typedef __u64 __bitwise __addrpair;

#if 0
struct in6_addr {
           __u8  s6_addr[16];  /* IPv6 address */
   };
#endif
struct hlist_node {
	struct hlist_node *next, **pprev;
};

typedef struct {
	void *net;
} possible_net_t;

struct sock_common {
        /* skc_daddr and skc_rcv_saddr must be grouped on a 8 bytes aligned
         * address on 64bit arches : cf INET_MATCH()
         */
        union {
                __addrpair      skc_addrpair;
                struct {
                        __be32  skc_daddr;
                        __be32  skc_rcv_saddr;
                };
        };
        union  {
                unsigned int    skc_hash;
                __u16           skc_u16hashes[2];
        };
        /* skc_dport && skc_num must be grouped as well */
        union {
                __portpair      skc_portpair;
                struct {
                        __be16  skc_dport;
                        __u16   skc_num;
                };
        };

        unsigned short          skc_family;
        volatile unsigned char  skc_state;
	unsigned char		skc_reuse:4;
	unsigned char		skc_reuseport:1;
	unsigned char		skc_ipv6only:1;
	unsigned char		skc_net_refcnt:1;
	int			skc_bound_dev_if;
	union {
		struct hlist_node	skc_bind_node;
		struct hlist_node	skc_portaddr_node;
	};
	struct proto		*skc_prot;
	possible_net_t		skc_net;

	struct in6_addr		skc_v6_daddr;
	struct in6_addr		skc_v6_rcv_saddr;

};

#endif

