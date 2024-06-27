// Project Calico BPF dataplane programs.
// Copyright (c) 2020-2022 Tigera, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0 OR GPL-2.0-or-later

#ifndef __CALI_BPF_H__
#define __CALI_BPF_H__

#include <linux/types.h>
#include <linux/bpf.h>
#include <bpf_helpers.h>   /* For bpf_xxx helper functions. */
#include <bpf_endian.h>    /* For bpf_ntohX etc. */
#include <stdbool.h>
#include <bpf_core_read.h>
#include <stddef.h>
#include <linux/ip.h>

/* CALI_BPF_INLINE must be defined before we include any of our headers. They
 * assume it exists!
 */
#define CALI_BPF_INLINE inline __attribute__((always_inline))

#include "globals.h"

#define BPF_REDIR_EGRESS 0
#define BPF_REDIR_INGRESS 1

/* These constants must be kept in sync with the calculate-flags script. */

// CALI_TC_HOST_EP is set for all host interfaces including tunnels.
#define CALI_TC_HOST_EP		(1<<0)
// CALI_TC_INGRESS is set when compiling a program in the "ingress" direction as defined by
// policy.  For host endpoints, ingress has its natural meaning (towards the host namespace)
// and it agrees with TC's definition of ingress. For workload endpoint programs, ingress is
// relative to the workload so the ingress program is applied at egress from the host namespace
// and vice-versa.
#define CALI_TC_INGRESS		(1<<1)
// CALI_TC_TUNNEL is set when compiling the program for the IPIP tunnel. It is *not* set
// when compiling the wireguard or tunnel program (or VXLAN).  IPIP is a special case because
// it is a layer 3 device, so we don't see an ethernet header on packets arriving from the IPIP
// device.
#define CALI_TC_TUNNEL		(1<<2)
// CALI_CGROUP is set when compiling the cgroup connect-time load balancer programs.
#define CALI_CGROUP		(1<<3)
// CALI_TC_DSR is set when compiling programs for DSR mode.  In DSR mode, traffic to node
// ports is encapped on the "request" leg but the response is returned directly from the
// node with the backing workload.
#define CALI_TC_DSR		(1<<4)
// CALI_L3_DEV is set for any L3 device such as wireguard and IPIP tunnels that act fully
// at layer 3. In kernels before 5.14 (rhel 4.18.0-330) IPIP tunnels on inbound
// direction were acting differently, where they could see outer ethernet and ip headers.
#define CALI_TC_L3_DEV 	(1<<5)
// CALI_XDP_PROG is set for programs attached to the XDP hook
#define CALI_XDP_PROG 	(1<<6)
#define CALI_TC_NAT_IF	(1<<7)
#define CALI_TC_LO	(1<<8)

/* EE only */
#define CALI_KPROBE	(1<<24)
#define CALI_STATS	(1<<25)

#ifndef CALI_DROP_WORKLOAD_TO_HOST
#define CALI_DROP_WORKLOAD_TO_HOST false
#endif

#ifndef CALI_COMPILE_FLAGS
#define CALI_COMPILE_FLAGS 0
#endif

#define CALI_F_INGRESS ((CALI_COMPILE_FLAGS) & CALI_TC_INGRESS)
#define CALI_F_EGRESS  (!CALI_F_INGRESS)

#define CALI_F_HEP     	 ((CALI_COMPILE_FLAGS) & (CALI_TC_HOST_EP | CALI_TC_NAT_IF))
#define CALI_F_WEP     	 (!CALI_F_HEP)
#define CALI_F_TUNNEL  	 (((CALI_COMPILE_FLAGS) & CALI_TC_TUNNEL) != 0)
#define CALI_F_L3_DEV    (((CALI_COMPILE_FLAGS) & CALI_TC_L3_DEV) != 0)
#define CALI_F_NAT_IF    (((CALI_COMPILE_FLAGS) & CALI_TC_NAT_IF) != 0)
#define CALI_F_LO        (((CALI_COMPILE_FLAGS) & CALI_TC_LO) != 0)

#define CALI_F_MAIN	(CALI_F_HEP && !CALI_F_TUNNEL && !CALI_F_L3_DEV && !CALI_F_NAT_IF && !CALI_F_LO)

#define CALI_F_XDP ((CALI_COMPILE_FLAGS) & CALI_XDP_PROG)

#define CALI_F_FROM_HEP (CALI_F_HEP && CALI_F_INGRESS)
#define CALI_F_TO_HEP   (CALI_F_HEP && !CALI_F_INGRESS)

#define CALI_F_FROM_WEP (CALI_F_WEP && CALI_F_EGRESS)
#define CALI_F_TO_WEP   (CALI_F_WEP && CALI_F_INGRESS)

#define CALI_F_TO_HOST       ((CALI_F_FROM_HEP || CALI_F_FROM_WEP) != 0)
#define CALI_F_FROM_HOST     (!CALI_F_TO_HOST)
#define CALI_F_L3            ((CALI_F_TO_HEP && CALI_F_TUNNEL) || CALI_F_L3_DEV)
#define CALI_F_IPIP_ENCAPPED ((CALI_F_INGRESS && CALI_F_TUNNEL))
#define CALI_F_L3_INGRESS    (CALI_F_INGRESS && CALI_F_L3_DEV)

#define CALI_F_CGROUP	(((CALI_COMPILE_FLAGS) & CALI_CGROUP) != 0)
#define CALI_F_DSR	((CALI_COMPILE_FLAGS & CALI_TC_DSR) != 0)

/* EE only */
#define CALI_F_KPROBE	(((CALI_COMPILE_FLAGS) & CALI_KPROBE) != 0)
#define CALI_F_STATS	(((CALI_COMPILE_FLAGS) & CALI_STATS) != 0)

#define CALI_RES_REDIR_BACK	108 /* packet should be sent back the same iface */
#define CALI_RES_REDIR_IFINDEX	109 /* packet should be sent straight to
				     * state->ct_result->ifindex_fwd
				     */
#if CALI_RES_REDIR_BACK <= TC_ACT_VALUE_MAX
#error CALI_RES_ values need to be increased above TC_ACT_VALUE_MAX
#endif

#ifndef CALI_FIB_LOOKUP_ENABLED
#define CALI_FIB_LOOKUP_ENABLED true
#endif

#define CALI_FIB_ENABLED (!CALI_F_L3 && CALI_FIB_LOOKUP_ENABLED && (CALI_F_TO_HOST || CALI_F_TO_HEP))

#define COMPILE_TIME_ASSERT(expr) {typedef char array[(expr) ? 1 : -1];}
static CALI_BPF_INLINE void __compile_asserts(void) {
#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Wunused-local-typedef"
	/* Either CALI_CGROUP or CALI_KPROBE or CALI_STATS is set or the other TC flags */
	COMPILE_TIME_ASSERT(
		CALI_COMPILE_FLAGS == 0 ||
		!!(CALI_COMPILE_FLAGS & (CALI_CGROUP | CALI_KPROBE | CALI_STATS)) !=
		!!(CALI_COMPILE_FLAGS & (CALI_TC_HOST_EP | CALI_TC_INGRESS | CALI_TC_TUNNEL | CALI_TC_DSR | CALI_XDP_PROG))
	);
	COMPILE_TIME_ASSERT(!CALI_F_DSR || (CALI_F_DSR && CALI_F_FROM_WEP) || (CALI_F_DSR && CALI_F_HEP));
	COMPILE_TIME_ASSERT(CALI_F_TO_HOST || CALI_F_FROM_HOST);
#pragma clang diagnostic pop
}

/* Calico BPF mode uses bits in the top 3 nibbles of the 32-bit packet mark, specifically
 * within 0x1FF00000.  To run successfully in BPF mode, Felix's IptablesMarkMask must be
 * configured to _include_ that mask _and_ to have some bits over for use by the
 * remaining iptables rules that do not interact with the BPF C code.  (Felix golang code
 * checks this at start of day and will shutdown and restart if IptablesMarkMask is
 * insufficient.)
 *
 * Bits used only by C code, or for interaction between C and golang code, must come out
 * of the 0x1FF00000, and must be defined compatibly here and in bpf/tc/tc_defs.go.
 *
 * The internal structure of the top 3 nibbles is as follows:

     . . . .  . . . 1  . . . .       packet SEEN by at least one TC program

     . . . .  . . 1 1  . . . .       BYPASS => SEEN and no further policy checking needed;
                                     remaining bits indicate options for how to treat such
                                     packets: FWD, FWD_SRC_FIXUP and NAT_OUT

     . . . .  . 1 0 1  . . . .       FALLTHROUGH => SEEN but no BPF CT state; need to check
                                     against Linux CT state

	 . . . .  . . . .  1 . . .		 SKIP_FIB => skip fib and send packet to host

     . . . .  1 . . .  . . . .       CT_ESTABLISHED: set by iptables to indicate match
                                     against Linux CT state

     . . . 1  . . . .  . . . .       EGRESS => packet should be routed via an egress gateway

     . . 1 .  . . . .  . . . .       conflicts with WG mark

     . 1 . .  . . . .  . . . .       packet should go back to bpfnatout

     1 . . .  . . . .  . . . .       packet passed through bpfnatout

 */

enum calico_skb_mark {
	/* The "SEEN" bit is set by any BPF program that allows a packet through.  It allows
	 * a second BPF program that handles the same packet to determine that another program
	 * handled it first. */
	CALI_SKB_MARK_SEEN                   = 0x01000000,
	CALI_SKB_MARK_SEEN_MASK              = CALI_SKB_MARK_SEEN,
	/* The "BYPASS" bit is an even stronger indication than "SEEN". It is set by BPF programs
	 * that have determined that the packet is approved and any downstream programs do not need
	 * to further validate the packet. */
	CALI_SKB_MARK_BYPASS                 = CALI_SKB_MARK_SEEN    | 0x02000000,
	/* "BYPASS_FWD" is a special case of "BYPASS" used when a packet returns from one of our
	 * VXLAN tunnels.  It tells the downstream program to forward the packet. */
	CALI_SKB_MARK_BYPASS_FWD             = CALI_SKB_MARK_BYPASS  | 0x00300000,
	/* Now unused mark */
	CALI_SKB_MARK_free_to_use            = CALI_SKB_MARK_BYPASS  | 0x00500000,
	CALI_SKB_MARK_BYPASS_MASK            = CALI_SKB_MARK_SEEN_MASK | 0x02700000,
	/* The FALLTHROUGH bit is used by programs that are towards the host namespace to indicate
	 * that the packet is not known in BPF conntrack. We have iptables rules to drop or allow
	 * such packets based on their Linux conntrack state. This allows for us to handle flows that
	 * were live before BPF was enabled. */
	CALI_SKB_MARK_FALLTHROUGH            = CALI_SKB_MARK_SEEN    | 0x04000000,
	/* The NAT_OUT bit is used by programs that are towards the host namespace to tell iptables to
	 * do SNAT for this flow.  Subsequent packets will also be allowed to fall through to the host
	 * netns. */
	CALI_SKB_MARK_NAT_OUT                = CALI_SKB_MARK_BYPASS  | 0x00800000,
	/* CALI_SKB_MARK_MASQ enforces MASQ on the connection. */
	CALI_SKB_MARK_MASQ                   = CALI_SKB_MARK_BYPASS  | 0x00600000,
	/* CALI_SKB_MARK_SKIP_FIB is used for packets that should pass through host IP stack. */
	CALI_SKB_MARK_SKIP_FIB               = CALI_SKB_MARK_SEEN | 0x00100000,
	/* CT_ESTABLISHED is used by iptables to tell the BPF programs that the packet is part of an
	 * established Linux conntrack flow. This allows the BPF program to let through preexisting
	 * flows at start of day. */
	CALI_SKB_MARK_CT_ESTABLISHED         = 0x08000000,
	CALI_SKB_MARK_CT_ESTABLISHED_MASK    = 0x08000000,
	CALI_SKB_MARK_RESERVED                = 0x11000000,
	/* CALI_SKB_MARK_RELATED_RESOLVED mean it is related traffic, we already
	 * know that and we have resolved NAT etc.
	 */
	CALI_SKB_MARK_RELATED_RESOLVED        = 0x21000000,
	/* CALI_SKB_MARK_TO_NAT_IFACE_OUT signals to routing that this packet should to
	 * to the bpfnatout interface.
	 */
	CALI_SKB_MARK_TO_NAT_IFACE_OUT        = 0x41000000,
	/* CALI_SKB_MARK_FROM_NAT_IFACE_OUT signals to the next hop that the packet passed
	 * through bpfnatout so that it can set its conntrack correctly.
	 */
	CALI_SKB_MARK_FROM_NAT_IFACE_OUT      = 0x81000000,

	/* EGRESS => packet should be routed via an egress gateway. */
	CALI_SKB_MARK_EGRESS                 = 0x10000000,
	CALI_SKB_MARK_EGRESS_MASK            = 0x10000000,
};

/* bpf_exit inserts a BPF exit instruction with the given return value. In a fully-inlined
 * BPF program this allows us to terminate early.  However(!) the exit instruction is also used
 * for function return so we need to be careful if we ever start using non-inlined
 * functions in anger. */
#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Winvalid-noreturn"
static CALI_BPF_INLINE __attribute__((noreturn)) void bpf_exit(int rc) {
	// Need volatile here because we don't use rc after this assembler fragment.
	// The BPF assembler rejects an input-only operand so we make r0 an in/out operand.
	asm volatile ( \
		"exit" \
		: "=r0" (rc) /*out*/ \
		: "0" (rc) /*in*/ \
		: /*clobber*/ \
	);
}
#pragma clang diagnostic pop

#ifdef IPVER6

#define debug_ip(ip) (bpf_htonl((ip).d))
#define ip_is_dnf(ip) (true)

#else

#define debug_ip(ip) bpf_htonl(ip)

#define ip_is_dnf(ip) ((ip)->frag_off & bpf_htons(0x4000))
#define ip_frag_no(ip) ((ip)->frag_off & bpf_htons(0x1fff))
#endif

static CALI_BPF_INLINE void ip_dec_ttl(struct iphdr *ip)
{
	ip->ttl--;
	/* since we change only a single byte, as per RFC-1141 we an adjust it
	 * inline without helpers.
	 */
	__u32 sum = ip->check;
	sum += bpf_htons(0x0100);
	ip->check = (__be16) (sum + (sum >> 16));
}

#ifdef IPVER6
#define ip_ttl_exceeded(ip) (CALI_F_TO_HOST && !CALI_F_TUNNEL && (ip)->hop_limit <= 1)
#else
#define ip_ttl_exceeded(ip) (CALI_F_TO_HOST && !CALI_F_TUNNEL && (ip)->ttl <= 1)
#endif

#if CALI_F_XDP

extern const volatile struct cali_xdp_preamble_globals __globals;
#define CALI_CONFIGURABLE(name) 1 /* any value will do, it is not configured */
#define CALI_CONFIGURABLE_IP(name) 1

#elif CALI_F_STATS

extern const volatile struct cali_stats_globals __globals;
#define CALI_CONFIGURABLE(name)  __globals.name

#elif (!CALI_F_CGROUP && !CALI_F_KPROBE && !CALI_F_STATS) || defined(UNITTEST)

extern const volatile struct cali_tc_preamble_globals __globals;
#define CALI_CONFIGURABLE(name) ctx->globals->data.name
#ifdef IPVER6
#define CALI_CONFIGURABLE_IP(name) CALI_CONFIGURABLE(name)
#else
#define CALI_CONFIGURABLE_IP(name) ctx->globals->data.name.a
#endif
#else

#define CALI_CONFIGURABLE(name) 1 /* any value will do, it is not configured */
#define CALI_CONFIGURABLE_IP(name) 1

#endif /* loader */

#define HOST_IP		CALI_CONFIGURABLE_IP(host_ip)
#define TUNNEL_MTU 	CALI_CONFIGURABLE(tunnel_mtu)
#define IF_NS           CALI_CONFIGURABLE(if_ns)
#define VXLAN_PORT 	CALI_CONFIGURABLE(vxlan_port)
#define INTF_IP		CALI_CONFIGURABLE_IP(intf_ip)
#define EXT_TO_SVC_MARK	CALI_CONFIGURABLE(ext_to_svc_mark)
#define PSNAT_START	CALI_CONFIGURABLE(psnat_start)
#define PSNAT_LEN	CALI_CONFIGURABLE(psnat_len)
#define GLOBAL_FLAGS 	CALI_CONFIGURABLE(flags)
#define HOST_TUNNEL_IP	CALI_CONFIGURABLE_IP(host_tunnel_ip)
#define WG_PORT		CALI_CONFIGURABLE(wg_port)
#define NATIN_IFACE	CALI_CONFIGURABLE(natin_idx)
#define EGW_VXLAN_PORT  CALI_CONFIGURABLE(egw_vxlan_port)
#define EGW_HEALTH_PORT  CALI_CONFIGURABLE(egw_health_port)

#define EGRESS_GATEWAY		(GLOBAL_FLAGS & CALI_GLOBALS_IS_EGRESS_GATEWAY)
#define EGRESS_CLIENT 		(GLOBAL_FLAGS & CALI_GLOBALS_IS_EGRESS_CLIENT)
#define ENABLE_TCP_STATS 	(GLOBAL_FLAGS & CALI_GLOBALS_TCP_STATS_ENABLED)
#define EGRESS_IP_ENABLED	(GLOBAL_FLAGS & CALI_GLOBALS_IS_EGRESS_IP_ENABLED)

#ifdef UNITTEST
#define CALI_PATCH_DEFINE(name, pattern)							\
static CALI_BPF_INLINE __be32 cali_patch_##name()					\
{												\
	__u32 ret;										\
	asm("%0 = " #pattern ";" : "=r"(ret) /* output */ : /* no inputs */ : /* no clobber */);\
	return ret;										\
}
#define CALI_PATCH(name)	cali_patch_##name()

CALI_PATCH_DEFINE(__skb_mark, 0x4d424b53) /* be 0x4d424b53 = ASCII(SKBM) */
#define SKB_MARK	CALI_PATCH(__skb_mark)
#endif

#define map_symbol(name, ver) name##ver

#define MAP_LOOKUP_FN(fname, name, ver) \
static CALI_BPF_INLINE void * fname##_lookup_elem(const void* key)	\
{									\
	return bpf_map_lookup_elem(&map_symbol(name, ver), key);	\
}

#define MAP_UPDATE_FN(fname, name, ver) \
static CALI_BPF_INLINE int fname##_update_elem(const void* key, const void* value, __u64 flags)\
{										\
	return bpf_map_update_elem(&map_symbol(name, ver), key, value, flags);	\
}

#define MAP_DELETE_FN(fname, name, ver) \
static CALI_BPF_INLINE int fname##_delete_elem(const void* key)	\
{									\
	return bpf_map_delete_elem(&map_symbol(name, ver), key);	\
}

#define CALI_MAP_NAMED(name, fname, ver,  map_type, key_type, val_type, size, flags)		\
struct {										\
	__uint(type, map_type);								\
	__type(key, key_type);								\
	__type(value, val_type);							\
	__uint(max_entries, size);							\
	__uint(map_flags, flags);							\
}map_symbol(name, ver) SEC(".maps");							\
	MAP_LOOKUP_FN(fname, name, ver)							\
	MAP_UPDATE_FN(fname, name, ver)							\
	MAP_DELETE_FN(fname, name, ver)

#define CALI_MAP(name, ver, map_type, key_type, val_type, size, flags)			\
		CALI_MAP_NAMED(name, name, ver,  map_type, key_type, val_type, size, flags)

#define CALI_MAP_V1(name, map_type, key_type, val_type, size, flags)			\
		CALI_MAP(name,, map_type, key_type, val_type, size, flags)

char ____license[] __attribute__((section("license"), used)) = "GPL";

#define NEXTHDR_HOP		0
#define NEXTHDR_ROUTING		43
#define NEXTHDR_FRAGMENT	44
#define NEXTHDR_GRE		47
#define NEXTHDR_ESP		50
#define NEXTHDR_AUTH		51
#define NEXTHDR_NONE		59
#define NEXTHDR_DEST		60
#define NEXTHDR_MOBILITY	135

#endif /* __CALI_BPF_H__ */
