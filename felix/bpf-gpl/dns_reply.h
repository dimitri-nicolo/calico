// Project Calico BPF dataplane programs.
// Copyright (c) 2021 Tigera, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0 OR GPL-2.0-or-later

#ifndef __CALI_DNS_REPLY_H__
#define __CALI_DNS_REPLY_H__

#define DNS_NAME_LEN		256
#define DNS_SCRATCH_SIZE 	256

#define DNS_ANSWERS_MAX		1000

struct dns_scratch {
	int name_len;
	unsigned char name[DNS_NAME_LEN];
	char ip[32];
	unsigned char buf[DNS_SCRATCH_SIZE];
};

struct dns_iter_ctx {
	struct dns_scratch *scratch;
	struct cali_tc_ctx *ctx;
	int off;
	bool failed;
	unsigned int answers;
};

CALI_MAP(cali_dns_data, 1,
		BPF_MAP_TYPE_PERCPU_ARRAY,
		__u32, struct dns_scratch,
		1, 0);

struct dnshdr {
	__be16 id;
	int qr:1;
	int opcode:4;
	int aa:1;
	int tc:1;
	int rd:1;
	int reserved:3;
	int rcode:4;
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

static CALI_BPF_INLINE unsigned int dns_skip_name(struct cali_tc_ctx *ctx, struct dns_scratch *scratch, int off)
{
	int size = ctx->skb->len - off;

	if (size <= 0) {
		CALI_DEBUG("DNS: read beyond the data\n");
		return 0;
	}

	if (size > DNS_NAME_LEN) {
		size = DNS_NAME_LEN;
	}

	if (!dns_load_bytes(ctx, scratch, off, size)) {
		return 0;
	}

	unsigned int i;

	/* We could have jump from size to size over the labes, but verifier wouldn't be happy */
	for (i = 1; i < DNS_SCRATCH_SIZE && scratch->buf[i] != 0; i++);

	if (i >= DNS_SCRATCH_SIZE) {
		CALI_DEBUG("DNS: name too long\n");
		return 0;
	}

	return i; /* returns how many bytes were skipped */
}

static CALI_BPF_INLINE bool dns_get_name(struct cali_tc_ctx *ctx, struct dns_scratch *scratch, int off)
{
	int size = ctx->skb->len - off;

	if (size <= 0) {
		CALI_DEBUG("DNS: read beyond the data\n");
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
	struct dns_scratch *scratch = ((struct dns_iter_ctx *)__ctx)->scratch;
	struct cali_tc_ctx *ctx = ((struct dns_iter_ctx *)__ctx)->ctx;
	int off = ((struct dns_iter_ctx *)__ctx)->off;

	if (((struct dns_iter_ctx *)__ctx)->answers == i) {
		return 1;
	}

	int bytes = dns_skip_name(ctx, scratch, off);

	if (bytes == 0) {
		CALI_DEBUG("DNS: failed skipping name in asnwer %d", i);
		goto failed;
	}

	CALI_DEBUG("DNS: skipped %d bytes of name\n", bytes);
	off += bytes + 1;

	struct dns_rr *rr = (void *) scratch->buf;

	if (!dns_load_bytes(ctx, scratch, off, sizeof(struct dns_rr))) {
		CALI_DEBUG("DNS: failed to read rr in asnwer %d", i);
		goto failed;
	}

	switch (bpf_ntohs(rr->type)) {
#ifdef IPVER6
	case TYPE_AAAA:
#else
	case TYPE_A:
#endif
		{
#ifdef IPVER6
			__u32 len = 32;
#else
			__u32 len = 4;
#endif
			if (bpf_load_bytes(ctx, off + sizeof(struct dns_rr), scratch->ip, len)) {
				CALI_DEBUG("DNS: failed to read data type %d class %d\n",
						bpf_ntohs(rr->type), bpf_ntohs(rr->class));
				goto failed;
			}
			CALI_DEBUG("DNS: IP 0x%x\n", *(__u32*)scratch->ip);
		}
		break;
	default:
		CALI_DEBUG("DNS: skipping rr type %d class %d\n", bpf_ntohs(rr->type), bpf_ntohs(rr->class));
	};

	((struct dns_iter_ctx *)__ctx)->off = off + sizeof(struct dns_rr) + bpf_ntohs(rr->rdlength);

	return 0;

failed:
	((struct dns_iter_ctx *)__ctx)->failed = true;
	return 1;
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
		return;
	}

	if (dnshdr.rcode != 0) {
		/* not interested in errors */
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

	CALI_DEBUG("Queries: %d\n", dnshdr.queries);

	unsigned int answers = dnshdr.answers + dnshdr.authority + dnshdr.additional;
	if (answers == 0) {
		CALI_DEBUG("DNS: no answers or data in the response\n");
	}

	CALI_DEBUG("Answers: %d\n", dnshdr.answers);
	CALI_DEBUG("Auth: %d\n", dnshdr.authority);
	CALI_DEBUG("Add: %d\n", dnshdr.additional);

	off += sizeof(struct dnshdr);
	if (!dns_get_name(ctx, scratch, off)) {
		CALI_DEBUG("DNS: Failed to get query name\n");
		return;
	}

	CALI_DEBUG("name '%s' %d\n", scratch->name, scratch->name_len);

	off += scratch->name_len + 2; /* skip the size of the first label and last 0 */

	struct dns_query * q = (void *) scratch->buf;

	if (!dns_load_bytes(ctx, scratch, off, sizeof(struct dns_query))) {
		CALI_DEBUG("DNS: Could not read rest of the query\n");
		return;
	}

	CALI_DEBUG("type %d class %d\n", bpf_ntohs(q->qtype), bpf_ntohs(q->qclass));
	
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

	off += sizeof(struct dns_query);

	/* Now start parsing answers. All sections carry RRs so just process
	 * them one by one, does not matter if it is an answer or auth etc.
	 */

	struct dns_iter_ctx ictx = {
		.scratch = scratch,
		.ctx = ctx,
		.off = off,
		.answers = answers,
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
