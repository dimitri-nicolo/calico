// Project Calico BPF dataplane programs.
// Copyright (c) 2021 Tigera, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0 OR GPL-2.0-or-later

#ifndef __CALI_DNS_REPLY_H__
#define __CALI_DNS_REPLY_H__

#include "policy.h"

#define DNS_NAME_LEN		256
#define DNS_SCRATCH_SIZE 	256

#define DNS_ANSWERS_MAX		1000

union dns_lpm_key {
        struct bpf_lpm_trie_key lpm;
	struct {
		__u32 len;
		unsigned char rev_name[DNS_NAME_LEN];
	};
};

#define DNS_LPM_IPSETS_MAX	8

struct dns_lpm_value {
	__u32 count;
	__u32 __pad; /* XXX flags for v4/v6 ? */
	__u64 ipsets[DNS_LPM_IPSETS_MAX]; /* If we want to expand beyond
					   * DNS_LPM_IPSETS_MAX sets in which the prefix
					   *  can live, we will need a prefix ID.
					   */
};

CALI_MAP(cali_dns_pfx, 2,
	 BPF_MAP_TYPE_LPM_TRIE,
	 union dns_lpm_key, struct dns_lpm_value,
	 64*1024, BPF_F_NO_PREALLOC)


struct dns_scratch {
	int name_len;
	unsigned char name[DNS_NAME_LEN];
	char ip[32];
	unsigned char buf[DNS_SCRATCH_SIZE];
	union dns_lpm_key lpm_key;
};

struct dns_iter_ctx {
	struct dns_scratch *scratch;
	struct cali_tc_ctx *ctx;
	int off;
	bool failed;
	unsigned int answers;
	struct dns_lpm_value v;
};

CALI_MAP(cali_dns_data, 1,
		BPF_MAP_TYPE_PERCPU_ARRAY,
		__u32, struct dns_scratch,
		1, 0);

struct dnshdr {
	__be16 id;
	__u16 reserved:3,
	      rcode:4,
	      qr:1,
	      opcode:4,
	      aa:1,
	      tc:1,
	      rd:1,
	      ra:1;
	__be16 queries;
	__be16 answers;
	__be16 authority;
	__be16 additional;
};

struct dns_query {
	__be16 qtype;
	__be16 qclass;
};

struct dns_rr {
	__be16 type;
	__be16 class;
	__be32 ttl;
	__be16 rdlength;
} __attribute__((packed));

#define CLASS_IN	1
#define CLASS_ANY	255

#define TYPE_A		1
#define TYPE_AAAA	28

static CALI_BPF_INLINE struct dns_scratch *dns_scratch_get()
{
	__u32 key = 0;
	return cali_dns_data_lookup_elem(&key);
}

static CALI_BPF_INLINE void *dns_load_bytes(struct cali_tc_ctx *ctx, struct dns_scratch *scratch,
					    unsigned int off, unsigned int size)
{
	if (!bpf_load_bytes(ctx, off, scratch->buf, size)) {
		return scratch->buf;
	}

	return NULL;
}

static CALI_BPF_INLINE int dns_skip_name(struct cali_tc_ctx *ctx, struct dns_scratch *scratch, int off)
{
	int size = ctx->skb->len - off;

	if (size <= 0) {
		CALI_DEBUG("DNS: read beyond the data\n");
		return -1;
	}

	if (size > DNS_NAME_LEN) {
		size = DNS_NAME_LEN;
	}

	if (!dns_load_bytes(ctx, scratch, off, size)) {
		CALI_DEBUG("DNS: failed to load %s bytes at off %d\n", size, off);
		return -1;
	}

	unsigned int i;

	/* We could have jump from size to size over the labes, but verifier wouldn't be happy */
	for (i = 0; i < DNS_SCRATCH_SIZE && scratch->buf[i] != 0; i++) {
		if ((scratch->buf[i] & 0xc0) == 0xc0) {
			CALI_DEBUG("DNS: pointer in name\n");
			i++; /* skip the offset */
			break;
		}
	}

	if (i >= DNS_SCRATCH_SIZE) {
		CALI_DEBUG("DNS: name too long\n");
		return -1;
	}

	return i; /* returns how many bytes were skipped */
}

static CALI_BPF_INLINE bool dns_get_name(struct cali_tc_ctx *ctx, struct dns_scratch *scratch, int off)
{
	int size = ctx->skb->len - off;

	if (size <= 0) {
		CALI_DEBUG("DNS: read beyond the data len %d off %d\n", ctx->skb->len, off);
		return false;
	}

	if (size > DNS_NAME_LEN) {
		size = DNS_NAME_LEN;
	}

	if (!dns_load_bytes(ctx, scratch, off, size)) {
		return false;
	}

	unsigned int i, next_len = scratch->buf[0] + 1;

	for (i = 1; i < DNS_SCRATCH_SIZE && scratch->buf[i] != 0; i++) {
		unsigned char c = scratch->buf[i];
		if (i == next_len) {
			/* add dots */
			next_len += c + 1;
			scratch->buf[i] = '.';
		} else if (c >= 'A' && c <= 'Z') {
			/* conert to lowercase */
			scratch->buf[i] = c + 'a' - 'A';
		}

		scratch->name[i - 1] = scratch->buf[i];
	}

	if (i >= DNS_SCRATCH_SIZE) {
		CALI_DEBUG("DNS: name too long\n");
		return false;
	}

	scratch->name_len = i - 1;
	scratch->name[i - 1] = '\0';

	return true;
}

static long dns_process_answer(__u32 i, void *__ctx)
{
	struct dns_iter_ctx *ictx = (struct dns_iter_ctx *)__ctx;
	struct dns_scratch *scratch = ictx->scratch;
	struct cali_tc_ctx *ctx = ictx->ctx;
	int off = ictx->off;

	if (ictx->answers == i) {
		return 1;
	}

	int bytes = dns_skip_name(ctx, scratch, off);

	if (bytes == -1) {
		CALI_DEBUG("DNS: failed skipping name in asnwer %d\n", i);
		goto failed;
	}

	CALI_DEBUG("DNS: skipped %d bytes of name\n", bytes);
	off += bytes + 1;

	struct dns_rr *rr = (void *) scratch->buf;

	if (!dns_load_bytes(ctx, scratch, off, sizeof(struct dns_rr))) {
		CALI_DEBUG("DNS: failed to read rr in asnwer %d\n", i);
		goto failed;
	}
			
	__u32 len = 4;

	switch (bpf_ntohs(rr->type)) {
	case TYPE_AAAA:
		len = 32;
		/* fallthrough */
	case TYPE_A:
		if (bpf_load_bytes(ctx, off + sizeof(struct dns_rr), scratch->ip, len)) {
			CALI_DEBUG("DNS: failed to read data type %d class %d\n",
					bpf_ntohs(rr->type), bpf_ntohs(rr->class));
			goto failed;
		}
		CALI_DEBUG("DNS: IP 0x%x\n", *(__u32*)scratch->ip);

		union ip_set_lpm_key k = {
			.ip = {
				.mask = (8 + len) * 8,
#ifndef IPVER6
				.addr = *(__u32*)scratch->ip,
#else
				.addr = {*(__u32*)scratch->ip},
#endif
			},
		};

		int i;

		for (i = 0; i < ictx->v.count && i < DNS_LPM_IPSETS_MAX; i++) {
			__u32 v = 0;
			int ret;

			k.ip.set_id = ictx->v.ipsets[i];
			if ((ret = cali_ip_sets_update_elem(&k, &v, 0))) {
				CALI_DEBUG("DNS: Failed to update ipset 0x%x err %d\n", k.ip.set_id, ret);
			}
		}
		break;
	default:
		CALI_DEBUG("DNS: skipping rr type %d class %d\n", bpf_ntohs(rr->type), bpf_ntohs(rr->class));
	};

	ictx->off = off + sizeof(struct dns_rr) + bpf_ntohs(rr->rdlength);

	return 0;

failed:
	ictx->failed = true;
	return 1;
}

static CALI_BPF_INLINE void dns_get_lpm_key(struct dns_scratch *scratch)
{
	unsigned char *name = scratch->name;
	unsigned char *key = scratch->lpm_key.rev_name;

	unsigned int len;

	if (scratch->name_len < 1 || scratch->name_len >= DNS_NAME_LEN) {
		return; /* we know that this is not true, but tell the verifier */
	}
	len = scratch->name_len - 1;

	unsigned char *k = key, *n = name + len;
	long d = len;

	/* Needs to be written with calculating and checking d indendently so
	 * that verifier is happy.
	 */
	for (; d >= 0;) {
		*k = *n;
		k++;
		n--;
		d = n - name;
	}

	key[len + 1] = '\0';

	scratch->lpm_key.len = (len + 1) * 8;
}

static CALI_BPF_INLINE void dns_process_datagram(struct cali_tc_ctx *ctx)
{
	int off = skb_iphdr_offset(ctx) + ctx->ipheader_len + UDP_SIZE;
	struct dns_scratch *scratch;
	struct dnshdr dnshdr;

	if (!(scratch = dns_scratch_get())) {
		CALI_DEBUG("DNS: could not get scratch.\n");
		return;
	}

	if (bpf_load_bytes(ctx, off, &dnshdr, sizeof(dnshdr))) {
		CALI_DEBUG("DNS: could not read header.\n");
		return;
	}

	if (!dnshdr.qr) {
		/* not interested in queries */
		CALI_DEBUG("DNS: ignoring query.\n");
		return;
	}

	if (dnshdr.rcode != 0) {
		/* not interested in errors */
		CALI_DEBUG("DNS: ignoring error 0x%x.\n", dnshdr.rcode);
		return;
	}

	dnshdr.queries = bpf_ntohs(dnshdr.queries);
	dnshdr.answers = bpf_ntohs(dnshdr.answers);
	dnshdr.authority = bpf_ntohs(dnshdr.authority);
	dnshdr.additional = bpf_ntohs(dnshdr.additional);

	if (dnshdr.queries != 1) {
		CALI_DEBUG("DNS: queries %d != 1\n", dnshdr.queries);
		return;
	}

	CALI_DEBUG("DNS: Queries: %d\n", dnshdr.queries);

	unsigned int answers = dnshdr.answers + dnshdr.authority + dnshdr.additional;
	if (answers == 0) {
		CALI_DEBUG("DNS: no answers or data in the response\n");
	}

	CALI_DEBUG("DNS: Answers: %d\n", dnshdr.answers);
	CALI_DEBUG("DNS: Auth: %d\n", dnshdr.authority);
	CALI_DEBUG("DNS: Add: %d\n", dnshdr.additional);

	off += sizeof(struct dnshdr);
	if (!dns_get_name(ctx, scratch, off)) {
		CALI_DEBUG("DNS: Failed to get query name\n");
		return;
	}

	CALI_DEBUG("DNS: name '%s' %d\n", scratch->name, scratch->name_len);

	off += scratch->name_len + 2; /* skip the size of the first label and last 0 */

	struct dns_query * q = (void *) scratch->buf;

	if (!dns_load_bytes(ctx, scratch, off, sizeof(struct dns_query))) {
		CALI_DEBUG("DNS: Could not read rest of the query\n");
		return;
	}

	CALI_DEBUG("DNS: type %d class %d\n", bpf_ntohs(q->qtype), bpf_ntohs(q->qclass));
	
	switch (bpf_ntohs(q->qclass)) {
	case CLASS_IN:
	case CLASS_ANY:
		break;
	default:
		CALI_DEBUG("DNS: Not interested in qclass %d\n", bpf_ntohs(q->qclass));
		return;
	}

	switch (bpf_ntohs(q->qtype)) {
#ifdef IPVER6
	case TYPE_AAAA:
#else
	case TYPE_A:
#endif
		break;
	default:
		CALI_DEBUG("DNS: Not interested in qtype %d\n", bpf_ntohs(q->qtype));
		return;
	}

	dns_get_lpm_key(scratch);

	struct dns_lpm_value *v = cali_dns_pfx_lookup_elem(&scratch->lpm_key);
	if (v) {
		CALI_DEBUG("DNS: HIT key '%s' len '%d'\n", scratch->lpm_key.rev_name, scratch->lpm_key.len);
		CALI_DEBUG("DNS: HIT sets count %d\n", v->count);
	} else {
		CALI_DEBUG("MISS key '%s' len '%d'\n", scratch->lpm_key.rev_name, scratch->lpm_key.len);
		return;
	}

	off += sizeof(struct dns_query);

	/* Now start parsing answers. All sections carry RRs so just process
	 * them one by one, does not matter if it is an answer or auth etc.
	 */

	struct dns_iter_ctx ictx = {
		.scratch = scratch,
		.ctx = ctx,
		.off = off,
		.answers = answers,
		.v = *v,
	};

	if (bpf_loop(DNS_ANSWERS_MAX, dns_process_answer, &ictx, 0) < 0) {
		CALI_DEBUG("DNS: bpf_loop failed\n");
		return;
	}

	if (ictx.failed) {
		CALI_DEBUG("DNS: bpf_loop callback failed\n");
		return;
	}
}

#endif /* __CALI_DNS_REPLY_H__*/
